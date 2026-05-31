package considered

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigWarnsOnVarianceWithoutStandard(t *testing.T) {
	cfg := Config{
		Standards: map[string]Boundary{"scc.code_lines": {Max: Float64(10)}},
		Variances: map[string]Variance{
			"main.go": {
				Kind:    "debt",
				Reason:  "x",
				Metrics: map[string]VarianceMetric{"scc.complexity": {Boundary: Boundary{Max: Float64(99)}}},
			},
		},
	}
	warnings := cfg.Warnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "scc.complexity") {
		t.Fatalf("expected one warning about scc.complexity, got %#v", warnings)
	}
}

func TestConfigValidateRequiresNamespacedStandardsAndVarianceRationale(t *testing.T) {
	cfg := Config{
		Standards: map[string]Boundary{
			"code_lines": {Max: Float64(10)},
		},
		Variances: map[string]Variance{
			"main.go": {Kind: "mystery"},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation errors")
	}
	msg := err.Error()
	for _, want := range []string{"must be namespaced", "unknown kind", "must define reason"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in %q", want, msg)
		}
	}
}

func TestConfigValidateAllowsBuiltInAndCustomKinds(t *testing.T) {
	cfg := Config{
		Kinds: []string{"performance"},
		Standards: map[string]Boundary{
			"scc.code_lines": {Max: Float64(10)},
		},
		Variances: map[string]Variance{
			"a.go": {
				Kind:   "architectural",
				Reason: "Centralized for readability.",
				Metrics: map[string]VarianceMetric{
					"scc.code_lines": {Boundary: Boundary{Max: Float64(20)}},
				},
			},
			"b.go": {
				Kind:   "performance",
				Reason: "Optimized hot path.",
				Metrics: map[string]VarianceMetric{
					"scc.code_lines": {Boundary: Boundary{Max: Float64(30)}},
				},
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigName)
	cfg := DefaultConfig()
	cfg.Variances["x.go"] = Variance{
		Kind:   "generated",
		Reason: "Generated source is committed.",
		Metrics: map[string]VarianceMetric{
			"scc.code_lines": {Boundary: Boundary{Max: Float64(99)}},
		},
	}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Variances["x.go"].Kind != "generated" {
		t.Fatalf("variance was not round-tripped: %#v", loaded.Variances)
	}
}

func TestWriteDefaultConfigRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteDefaultConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDefaultConfig(dir); err == nil {
		t.Fatal("expected overwrite refusal")
	}
}
