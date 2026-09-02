package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agorischek/considered/internal/considered"
)

func TestRunInitAndCheck(t *testing.T) {
	dir := t.TempDir()
	if code := run([]string{"init", "--root", dir}); code != 0 {
		t.Fatalf("init exit code = %d", code)
	}
	cfg := considered.Config{
		Standards: map[string]considered.Boundary{
			"filesystem.bytes": {Max: considered.Float64(2)},
		},
		Variances: map[string]considered.Variance{},
	}
	if err := considered.SaveConfig(filepath.Join(dir, considered.ConfigName), cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"check", "--root", dir, "--json"}); code != 1 {
		t.Fatalf("check exit code = %d", code)
	}
}

func TestRunCheckSarifSuccess(t *testing.T) {
	dir := t.TempDir()
	cfg := considered.Config{
		Standards: map[string]considered.Boundary{
			"filesystem.bytes": {Max: considered.Float64(1000)},
		},
		Variances: map[string]considered.Variance{},
	}
	if err := considered.SaveConfig(filepath.Join(dir, considered.ConfigName), cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"check", "--root", dir, "--sarif"}); code != 0 {
		t.Fatalf("check exit code = %d", code)
	}
}

func TestRunCheckWarningsDoNotFail(t *testing.T) {
	dir := t.TempDir()
	cfg := considered.Config{
		Standards: map[string]considered.Boundary{
			"filesystem.bytes": {Max: considered.Float64(100)},
		},
		WarningThresholds: considered.WarningConfig{
			PercentBelowMax: considered.Float64(10),
		},
		Variances: map[string]considered.Variance{},
	}
	if err := considered.SaveConfig(filepath.Join(dir, considered.ConfigName), cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte(strings.Repeat("a", 95)), 0644); err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		if code := run([]string{"check", "--root", dir}); code != 0 {
			t.Fatalf("check exit code = %d", code)
		}
	})
	if !strings.Contains(out, "Warnings") || !strings.Contains(out, "within 10% of max") {
		t.Fatalf("expected warning output: %q", out)
	}
}

func TestRunCheckPrintsInstructionsBeforeViolations(t *testing.T) {
	dir := t.TempDir()
	cfg := considered.Config{
		Instructions: "Prefer small functions.\nPreserve public APIs.\n",
		Standards: map[string]considered.Boundary{
			"filesystem.bytes": {Max: considered.Float64(2)},
		},
		Variances: map[string]considered.Variance{},
	}
	if err := considered.SaveConfig(filepath.Join(dir, considered.ConfigName), cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		if code := run([]string{"check", "--root", dir}); code != 1 {
			t.Fatalf("check exit code = %d", code)
		}
	})
	want := "Instructions: Prefer small functions.\nPreserve public APIs.\n\nViolations"
	if !strings.Contains(out, want) {
		t.Fatalf("instructions were not printed before violations: %q", out)
	}
}

func TestRunCheckJSONRemainsMachineReadableWithInstructions(t *testing.T) {
	dir := t.TempDir()
	cfg := considered.Config{
		Instructions: "Prefer small functions.",
		Standards: map[string]considered.Boundary{
			"filesystem.bytes": {Max: considered.Float64(100)},
		},
		Variances: map[string]considered.Variance{},
	}
	if err := considered.SaveConfig(filepath.Join(dir, considered.ConfigName), cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		if code := run([]string{"check", "--root", dir, "--json"}); code != 0 {
			t.Fatalf("check exit code = %d", code)
		}
	})
	var report considered.Report
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("check output is not valid JSON: %v\n%s", err, out)
	}
}

func TestRunInstructions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, considered.ConfigName)
	cfg := considered.Config{
		Instructions: "Prefer small functions.\nPreserve public APIs.\n",
		Standards:    map[string]considered.Boundary{},
		Variances:    map[string]considered.Variance{},
	}
	if err := considered.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		if code := run([]string{"instructions", "--config", path}); code != 0 {
			t.Fatalf("instructions exit code = %d", code)
		}
	})
	if out != cfg.Instructions {
		t.Fatalf("instructions output = %q", out)
	}

	cfg.Instructions = ""
	if err := considered.SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	out = captureStdout(t, func() {
		if code := run([]string{"instructions", "--root", dir}); code != 0 {
			t.Fatalf("instructions exit code = %d", code)
		}
	})
	if out != "(No instructions provided)\n" {
		t.Fatalf("empty instructions output = %q", out)
	}
}

func TestRunVarianceAdd(t *testing.T) {
	dir := t.TempDir()
	cfg := considered.Config{
		Standards: map[string]considered.Boundary{
			"filesystem.bytes": {Max: considered.Float64(1)},
		},
		Variances: map[string]considered.Variance{},
	}
	if err := considered.SaveConfig(filepath.Join(dir, considered.ConfigName), cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	code := run([]string{
		"variance", "add",
		"--root", dir,
		"--subject", "a.txt",
		"--metric", "filesystem.bytes",
		"--kind", "debt",
		"--reason", "Temporary.",
	})
	if code != 0 {
		t.Fatalf("variance add exit code = %d", code)
	}
}

func TestRunRejectsUnknownAndConflictingOutput(t *testing.T) {
	if code := run([]string{"nope"}); code != 2 {
		t.Fatalf("unknown command exit code = %d", code)
	}
	dir := t.TempDir()
	cfg := considered.Config{
		Standards: map[string]considered.Boundary{
			"filesystem.bytes": {Max: considered.Float64(1)},
		},
		Variances: map[string]considered.Variance{},
	}
	if err := considered.SaveConfig(filepath.Join(dir, considered.ConfigName), cfg); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"check", "--root", dir, "--json", "--sarif"}); code != 2 {
		t.Fatalf("conflicting output exit code = %d", code)
	}
}

func TestRunCoversHelpTextAndErrorPaths(t *testing.T) {
	if code := run(nil); code != 2 {
		t.Fatalf("empty args exit code = %d", code)
	}
	if code := run([]string{"help"}); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	dir := t.TempDir()
	if code := run([]string{"init", "--root", dir}); code != 0 {
		t.Fatalf("init exit code = %d", code)
	}
	if code := run([]string{"init", "--root", dir}); code != 1 {
		t.Fatalf("init overwrite exit code = %d", code)
	}
	if code := run([]string{"check", "--root", dir, "--bad"}); code != 2 {
		t.Fatalf("bad check flag exit code = %d", code)
	}
	if code := run([]string{"variance"}); code != 2 {
		t.Fatalf("bad variance usage exit code = %d", code)
	}
	if got := resolveConfigPath("root", "custom.yaml"); got != "custom.yaml" {
		t.Fatalf("explicit config path changed: %q", got)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	fn()
	writer.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestUsageIncludesCommands(t *testing.T) {
	out := captureStdout(t, func() { usage(os.Stdout) })
	if !strings.Contains(out, "variance add") || !strings.Contains(out, "instructions") {
		t.Fatalf("unexpected usage: %q", out)
	}
}

func TestRunVersion(t *testing.T) {
	out := captureStdout(t, func() {
		if code := run([]string{"version"}); code != 0 {
			t.Fatalf("version exit code = %d", code)
		}
	})
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected version output, got %q", out)
	}
}

func TestRunCheckMissingConfig(t *testing.T) {
	if code := run([]string{"check", "--root", t.TempDir()}); code != 2 {
		t.Fatalf("missing-config exit code = %d", code)
	}
}

var _ = context.Background
