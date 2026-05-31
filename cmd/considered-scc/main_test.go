package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunRequiresJSON(t *testing.T) {
	var stderr bytes.Buffer
	if code := run(nil, &bytes.Buffer{}, &stderr); code != 2 {
		t.Fatalf("exit code = %d", code)
	}
}

func TestRunEmitsProviderMetrics(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--root", dir, "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	var payload struct {
		Metrics []struct {
			Subject string             `json:"subject"`
			Values  map[string]float64 `json:"values"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(payload.Metrics) != 1 {
		t.Fatalf("expected one metric record, got %#v", payload)
	}
	if payload.Metrics[0].Subject != "main.go" || payload.Metrics[0].Values["scc.code_lines"] == 0 {
		t.Fatalf("unexpected payload: %#v", payload.Metrics[0])
	}
}
