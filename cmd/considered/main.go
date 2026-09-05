package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/quitepicky/considered/internal/considered"
)

var versionOverride string

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage(os.Stderr)
		return 2
	}
	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "check":
		return runCheck(args[1:])
	case "instructions":
		return runInstructions(args[1:])
	case "variance":
		return runVariance(args[1:])
	case "version", "--version", "-v":
		fmt.Fprintln(os.Stdout, version())
		return 0
	case "-h", "--help", "help":
		usage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		usage(os.Stderr)
		return 2
	}
}

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", ".", "repository root")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	path, err := considered.WriteDefaultConfig(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "created %s\n", path)
	return 0
}

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", ".", "repository root")
	configPath := fs.String("config", "", "configuration file")
	jsonOut := fs.Bool("json", false, "write JSON output")
	sarifOut := fs.Bool("sarif", false, "write SARIF output")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	path := resolveConfigPath(*root, *configPath)
	cfg, err := considered.LoadConfig(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "no configuration at %s; run 'considered init' to create one\n", path)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		return 2
	}
	for _, warning := range cfg.Warnings() {
		fmt.Fprintln(os.Stderr, "warning: "+warning)
	}
	report, err := considered.Check(context.Background(), *root, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	switch {
	case *jsonOut && *sarifOut:
		fmt.Fprintln(os.Stderr, "--json and --sarif cannot be used together")
		return 2
	case *jsonOut:
		err = considered.WriteJSON(os.Stdout, report)
	case *sarifOut:
		err = considered.WriteSARIF(os.Stdout, report)
	default:
		if cfg.Instructions != "" {
			if err := writeInstructions(os.Stdout, cfg.Instructions, true); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 2
			}
		}
		err = considered.WriteText(os.Stdout, report)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if report.HasViolations() {
		return 1
	}
	return 0
}

func runVariance(args []string) int {
	if len(args) == 0 || args[0] != "add" {
		fmt.Fprintln(os.Stderr, "usage: considered variance add [flags]")
		return 2
	}
	fs := flag.NewFlagSet("variance add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", ".", "repository root")
	configPath := fs.String("config", "", "configuration file")
	subject := fs.String("subject", "", "subject path")
	metric := fs.String("metric", "", "metric name")
	kind := fs.String("kind", "", "variance kind")
	reason := fs.String("reason", "", "variance rationale")
	metricReason := fs.String("metric-reason", "", "rationale specific to this metric override")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	err := considered.AddVariance(context.Background(), considered.VarianceAddOptions{
		ConfigPath:   resolveConfigPath(*root, *configPath),
		Root:         *root,
		Subject:      *subject,
		Metric:       *metric,
		Kind:         *kind,
		Reason:       *reason,
		MetricReason: *metricReason,
		In:           os.Stdin,
		Out:          os.Stdout,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintln(os.Stdout, "variance added")
	return 0
}

func resolveConfigPath(root, path string) string {
	if path != "" {
		return path
	}
	return filepath.Join(root, considered.ConfigName)
}

func version() string {
	if versionOverride != "" {
		return versionOverride
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func usage(out *os.File) {
	fmt.Fprintln(out, `usage: considered <command> [flags]

Commands:
  init              create .considered.yaml
  check             evaluate repository standards
  instructions      print instructions for coding agents
  variance add      add a documented variance
  version           print the version`)
}
