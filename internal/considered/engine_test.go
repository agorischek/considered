package considered

import "testing"
import (
	"context"
	"os"
	"path/filepath"
)

func TestEvaluateReportsViolationsAndVariances(t *testing.T) {
	cfg := Config{
		Standards: map[string]Boundary{
			"scc.code_lines": {Max: Float64(100)},
			"scc.complexity": {Max: Float64(10)},
		},
		Variances: map[string]Variance{
			"parser.go": {
				Kind:   "architectural",
				Reason: "Parser rules are intentionally centralized.",
				Metrics: map[string]VarianceMetric{
					"scc.code_lines": {Boundary: Boundary{Max: Float64(200)}},
				},
			},
		},
	}
	report := Evaluate(cfg, []MetricRecord{
		{Subject: "parser.go", Provider: "scc", Values: map[string]float64{"scc.code_lines": 150, "scc.complexity": 12}},
		{Subject: "small.go", Provider: "scc", Values: map[string]float64{"scc.code_lines": 10}},
	})
	if len(report.Variances) != 1 {
		t.Fatalf("expected 1 variance, got %#v", report.Variances)
	}
	if report.Variances[0].Kind != "architectural" {
		t.Fatalf("variance metadata missing: %#v", report.Variances[0])
	}
	if len(report.Violations) != 1 || report.Violations[0].Metric != "scc.complexity" {
		t.Fatalf("expected complexity violation, got %#v", report.Violations)
	}
}

func TestEvaluateReportsExceededVarianceAsViolation(t *testing.T) {
	cfg := Config{
		Standards: map[string]Boundary{"scc.code_lines": {Max: Float64(100)}},
		Variances: map[string]Variance{
			"parser.go": {
				Kind:   "architectural",
				Reason: "Centralized.",
				Metrics: map[string]VarianceMetric{
					"scc.code_lines": {Boundary: Boundary{Max: Float64(120)}},
				},
			},
		},
	}
	report := Evaluate(cfg, []MetricRecord{
		{Subject: "parser.go", Values: map[string]float64{"scc.code_lines": 121}},
	})
	if len(report.Violations) != 1 || !report.Violations[0].VarianceExceeded {
		t.Fatalf("expected exceeded variance violation, got %#v", report.Violations)
	}
}

func TestEvaluateReportsWarningsNearStandardsAndVariances(t *testing.T) {
	cfg := Config{
		Standards: map[string]Boundary{
			"filesystem.bytes": {Min: Float64(100)},
			"scc.code_lines":   {Max: Float64(100)},
		},
		WarningThresholds: WarningConfig{
			PercentBelowMax: Float64(10),
			PercentAboveMin: Float64(10),
		},
		Variances: map[string]Variance{
			"generated.go": {
				Kind:   "generated",
				Reason: "Generated source is committed.",
				Metrics: map[string]VarianceMetric{
					"scc.code_lines": {Boundary: Boundary{Max: Float64(200)}},
				},
			},
		},
	}
	report := Evaluate(cfg, []MetricRecord{
		{Subject: "near-max.go", Values: map[string]float64{"scc.code_lines": 91}},
		{Subject: "near-min.bin", Values: map[string]float64{"filesystem.bytes": 105}},
		{Subject: "safe.go", Values: map[string]float64{"scc.code_lines": 80}},
		{Subject: "over.go", Values: map[string]float64{"scc.code_lines": 101}},
		{Subject: "generated.go", Values: map[string]float64{"scc.code_lines": 190}},
	})
	if len(report.Warnings) != 3 {
		t.Fatalf("expected 3 warnings, got %#v", report.Warnings)
	}
	if len(report.Variances) != 1 || report.Variances[0].Subject != "generated.go" {
		t.Fatalf("expected generated variance, got %#v", report.Variances)
	}
	if len(report.Violations) != 1 || report.Violations[0].Subject != "over.go" {
		t.Fatalf("expected over.go violation, got %#v", report.Violations)
	}
	if report.Warnings[0].Subject != "generated.go" || report.Warnings[0].WarningBoundary != "max" {
		t.Fatalf("expected generated max warning first after sort, got %#v", report.Warnings)
	}
	if report.Warnings[1].WarningBoundary != "max" || report.Warnings[2].WarningBoundary != "min" {
		t.Fatalf("unexpected warning boundaries: %#v", report.Warnings)
	}
}

func TestEvaluateSkipsExcludedSubjects(t *testing.T) {
	cfg := Config{
		Exclude:   ExcludeConfig{Categories: []string{"tests"}},
		Standards: map[string]Boundary{"scc.complexity": {Max: Float64(10)}},
	}
	report := Evaluate(cfg, []MetricRecord{
		{Subject: "internal/foo.go", Values: map[string]float64{"scc.complexity": 20}},
		{Subject: "internal/foo_test.go", Values: map[string]float64{"scc.complexity": 99}},
	})
	if len(report.Violations) != 1 || report.Violations[0].Subject != "internal/foo.go" {
		t.Fatalf("excluded test file should not be evaluated: %#v", report.Violations)
	}
}

func TestMetricRecordsBySubjectMergesProviders(t *testing.T) {
	got := MetricRecordsBySubject([]MetricRecord{
		{Subject: "a.go", Values: map[string]float64{"scc.code_lines": 1}},
		{Subject: "a.go", Values: map[string]float64{"filesystem.bytes": 2}},
	})
	if got["a.go"]["scc.code_lines"] != 1 || got["a.go"]["filesystem.bytes"] != 2 {
		t.Fatalf("metrics were not merged: %#v", got)
	}
}

func TestCheckCollectsAndEvaluates(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abcd"), 0644); err != nil {
		t.Fatal(err)
	}
	report, err := Check(context.Background(), dir, Config{
		Standards: map[string]Boundary{"filesystem.bytes": {Max: Float64(3)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Violations) != 1 || report.Violations[0].Subject != "a.txt" {
		t.Fatalf("unexpected report: %#v", report)
	}
}
