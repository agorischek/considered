package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/quitepicky/considered/internal/considered"
)

func runInstructions(args []string) int {
	fs := flag.NewFlagSet("instructions", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", ".", "repository root")
	configPath := fs.String("config", "", "configuration file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := considered.LoadConfig(resolveConfigPath(*root, *configPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if cfg.Instructions == "" {
		fmt.Fprintln(os.Stdout, "(No instructions provided)")
		return 0
	}
	if err := writeInstructions(os.Stdout, cfg.Instructions, false); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return 0
}

func writeInstructions(out io.Writer, instructions string, prefixed bool) error {
	if prefixed {
		if _, err := fmt.Fprint(out, "Instructions: "); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(out, instructions); err != nil {
		return err
	}
	if !strings.HasSuffix(instructions, "\n") {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}
	if prefixed {
		_, err := fmt.Fprintln(out)
		return err
	}
	return nil
}
