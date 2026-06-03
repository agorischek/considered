package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/agorischek/considered/internal/considered"
	"github.com/boyter/scc/v3/processor"
)

type sccLanguageSummary struct {
	Files []*sccFile `json:"Files"`
}

type sccFile struct {
	Filename   string  `json:"Filename"`
	Location   string  `json:"Location"`
	Code       float64 `json:"Code"`
	Comment    float64 `json:"Comment"`
	Complexity float64 `json:"Complexity"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("considered-scc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	jsonOut := fs.Bool("json", false, "write provider JSON")
	excludeDirs := repeatFlag{}
	fs.Var(&excludeDirs, "exclude-dir", "directory suffix to exclude before collecting metrics")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "--json is required")
		return 2
	}
	records, err := collect(*root, excludeDirs)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	data, err := considered.MarshalProviderOutput(records)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(data))
	return 0
}

func collect(root string, excludeDirs []string) ([]considered.MetricRecord, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	output, err := runSCC(absRoot, excludeDirs)
	if err != nil {
		return nil, err
	}
	var summaries []sccLanguageSummary
	if err := json.Unmarshal(output, &summaries); err != nil {
		return nil, fmt.Errorf("decode scc output: %w", err)
	}
	var records []considered.MetricRecord
	for _, summary := range summaries {
		for _, file := range summary.Files {
			path := file.Location
			if path == "" {
				path = file.Filename
			}
			subject := filepath.ToSlash(filepath.Clean(path))
			if filepath.IsAbs(path) {
				var err error
				subject, err = considered.SubjectPath(absRoot, path)
				if err != nil {
					return nil, err
				}
			}
			records = append(records, considered.MetricRecord{
				Subject:  subject,
				Provider: "scc",
				Values: map[string]float64{
					"scc.code_lines":    file.Code,
					"scc.comment_lines": file.Comment,
					"scc.complexity":    file.Complexity,
				},
			})
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Subject < records[j].Subject })
	return records, nil
}

// runSCC runs scc over absRoot, which the caller has already resolved to an
// absolute path.
func runSCC(absRoot string, excludeDirs []string) ([]byte, error) {
	processor.Files = true
	processor.Format = "json"
	processor.DirFilePaths = []string{absRoot}
	processor.PathDenyList = append([]string{}, excludeDirs...)

	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	readErr := make(chan error, 1)
	go func() {
		_, err := io.Copy(&buf, reader)
		readErr <- err
	}()
	os.Stdout = writer
	processor.Process()
	writer.Close()
	os.Stdout = old
	if err := <-readErr; err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type repeatFlag []string

func (f *repeatFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *repeatFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
