package considered

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddVarianceCreatesApprovedBoundary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Standards: map[string]Boundary{
			"filesystem.bytes": {Max: Float64(3)},
		},
		Variances: map[string]Variance{},
	}
	configPath := filepath.Join(dir, ConfigName)
	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	err := AddVariance(context.Background(), VarianceAddOptions{
		ConfigPath: configPath,
		Root:       dir,
		Subject:    "large.txt",
		Metric:     "filesystem.bytes",
		Kind:       "debt",
		Reason:     "Temporary fixture.",
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.Variances["large.txt"].Metrics["filesystem.bytes"].Max
	if got == nil || *got != 5 {
		t.Fatalf("unexpected variance boundary: %#v", loaded.Variances)
	}
}

func TestAddVarianceRejectsValueWithinStandard(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "small.txt"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, ConfigName)
	if err := SaveConfig(configPath, Config{
		Standards: map[string]Boundary{"filesystem.bytes": {Max: Float64(100)}},
	}); err != nil {
		t.Fatal(err)
	}
	err := AddVariance(context.Background(), VarianceAddOptions{
		ConfigPath: configPath,
		Root:       dir,
		Subject:    "small.txt",
		Metric:     "filesystem.bytes",
		Kind:       "debt",
		Reason:     "unnecessary",
	})
	if err == nil || !strings.Contains(err.Error(), "already satisfies") {
		t.Fatalf("expected already-satisfies error, got %v", err)
	}
}

func TestAddVarianceRecordsMetricReason(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, ConfigName)
	if err := SaveConfig(configPath, Config{
		Standards: map[string]Boundary{"filesystem.bytes": {Max: Float64(3)}},
	}); err != nil {
		t.Fatal(err)
	}
	err := AddVariance(context.Background(), VarianceAddOptions{
		ConfigPath:   configPath,
		Root:         dir,
		Subject:      "large.txt",
		Metric:       "filesystem.bytes",
		Kind:         "debt",
		Reason:       "fixture",
		MetricReason: "specific to bytes",
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Variances["large.txt"].Metrics["filesystem.bytes"].Reason; got != "specific to bytes" {
		t.Fatalf("unexpected metric reason: %q", got)
	}
}

func TestBoundaryForActualUsesMinWhenStandardRequiresMinimum(t *testing.T) {
	boundary := boundaryForActual(Boundary{Min: Float64(80)}, 50)
	if boundary.Min == nil || *boundary.Min != 50 || boundary.Max != nil {
		t.Fatalf("unexpected boundary: %#v", boundary)
	}
}

func TestAddVariancePromptsForMissingValues(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, ConfigName)
	if err := SaveConfig(configPath, Config{
		Standards: map[string]Boundary{"filesystem.bytes": {Max: Float64(3)}},
	}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := AddVariance(context.Background(), VarianceAddOptions{
		ConfigPath: configPath,
		Root:       dir,
		In:         strings.NewReader("1\n1\ndebt\nBecause it is a fixture.\n"),
		Out:        &out,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "subject:") {
		t.Fatalf("expected prompts, got %q", out.String())
	}
}

func TestPrompterRejectsUnknownChoiceAndEmptyText(t *testing.T) {
	p := newPrompter(strings.NewReader("3\n"), nil)
	if _, err := p.choice("metric", []string{"a", "b"}); err == nil {
		t.Fatal("expected unknown choice error")
	}
	p = newPrompter(strings.NewReader("\n"), nil)
	if _, err := p.text("reason"); err == nil {
		t.Fatal("expected empty text error")
	}
}
