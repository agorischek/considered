package considered

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func sampleReport() Report {
	return Report{
		Violations: []Finding{{
			Subject:  "a.go",
			Metric:   "scc.code_lines",
			Actual:   12,
			Standard: Boundary{Max: Float64(10)},
		}},
		Warnings: []Finding{{
			Subject:         "c.go",
			Metric:          "scc.code_lines",
			Actual:          9,
			Standard:        Boundary{Max: Float64(10)},
			WarningBoundary: "max",
			WarningPercent:  Float64(10),
		}},
		Variances: []Finding{{
			Subject:  "b.go",
			Metric:   "scc.code_lines",
			Actual:   20,
			Standard: Boundary{Max: Float64(10)},
			Approved: &Boundary{Max: Float64(20)},
			Kind:     "generated",
			Reason:   "Generated source is committed.",
		}},
	}
}

func TestWriteTextDistinguishesViolationsAndVariances(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, sampleReport()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Violations", "Warnings", "Variances", "actual: 12", "warning: within 10% of max", "approved: <= 20"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
}

func TestWriteTextEmptyReport(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteText(&buf, Report{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Count(out, "None") != 3 {
		t.Fatalf("expected empty report to show no findings: %q", out)
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sampleReport()); err != nil {
		t.Fatal(err)
	}
	var report Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.HasViolations() {
		t.Fatalf("expected violations after round trip: %#v", report)
	}
}

func TestWriteSARIFLevels(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, sampleReport()); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	text := buf.String()
	for _, want := range []string{`"level": "error"`, `"level": "warning"`, `"level": "note"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected SARIF level %s: %s", want, text)
		}
	}
}

func TestBoundaryString(t *testing.T) {
	got := (Boundary{Min: Float64(1), Max: Float64(2.5)}).String()
	if got != ">= 1 and <= 2.5" {
		t.Fatalf("unexpected boundary string: %q", got)
	}
}

func TestMarshalProviderOutputAndSortedMetrics(t *testing.T) {
	data, err := MarshalProviderOutput([]MetricRecord{{Subject: "a", Values: map[string]float64{"x.y": 1}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"metrics"`) {
		t.Fatalf("unexpected provider output: %s", string(data))
	}
	got := SortedMetrics(map[string]Boundary{"b.y": {}, "a.y": {}})
	if strings.Join(got, ",") != "a.y,b.y" {
		t.Fatalf("unexpected sort: %v", got)
	}
}
