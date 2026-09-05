package considered

import (
	"encoding/json"
	"fmt"
	"io"
)

func WriteJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func WriteText(w io.Writer, report Report) error {
	if len(report.Violations) == 0 {
		if _, err := fmt.Fprintln(w, "Violations\n\n  None"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "Violations"); err != nil {
			return err
		}
		for _, f := range report.Violations {
			if _, err := fmt.Fprintf(w, "\n  %s\n\n    %s\n\n      actual: %s\n      standard: %s\n", f.Subject, f.Metric, FormatValue(f.Actual), f.Standard.String()); err != nil {
				return err
			}
			if f.Approved != nil {
				if _, err := fmt.Fprintf(w, "      approved: %s\n", f.Approved.String()); err != nil {
					return err
				}
			}
		}
	}

	if len(report.Warnings) == 0 {
		if _, err := fmt.Fprintln(w, "\nWarnings\n\n  None"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "\nWarnings"); err != nil {
			return err
		}
		for _, f := range report.Warnings {
			if _, err := fmt.Fprintf(w, "\n  %s\n\n    %s\n\n      actual: %s\n      standard: %s\n", f.Subject, f.Metric, FormatValue(f.Actual), f.Standard.String()); err != nil {
				return err
			}
			if f.Approved != nil {
				if _, err := fmt.Fprintf(w, "      approved: %s\n", f.Approved.String()); err != nil {
					return err
				}
			}
			if f.WarningBoundary != "" && f.WarningPercent != nil {
				if _, err := fmt.Fprintf(w, "      warning: within %s%% of %s\n", FormatValue(*f.WarningPercent), f.WarningBoundary); err != nil {
					return err
				}
			}
		}
	}

	if len(report.Variances) == 0 {
		_, err := fmt.Fprintln(w, "\nVariances\n\n  None")
		return err
	}
	if _, err := fmt.Fprintln(w, "\nVariances"); err != nil {
		return err
	}
	for _, f := range report.Variances {
		reason := f.Reason
		if f.MetricReason != "" {
			reason = f.MetricReason
		}
		if _, err := fmt.Fprintf(w, "\n  %s\n\n    kind: %s\n\n    %s\n\n      actual: %s\n      approved: %s\n      reason: %s\n", f.Subject, f.Kind, f.Metric, FormatValue(f.Actual), f.Approved.String(), reason); err != nil {
			return err
		}
	}
	return nil
}

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	ShortDesc sarifMessage `json:"shortDescription"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

func WriteSARIF(w io.Writer, report Report) error {
	ruleSeen := map[string]bool{}
	rules := []sarifRule{}
	results := []sarifResult{}
	add := func(f Finding, level string) {
		if !ruleSeen[f.Metric] {
			ruleSeen[f.Metric] = true
			rules = append(rules, sarifRule{
				ID:        f.Metric,
				Name:      f.Metric,
				ShortDesc: sarifMessage{Text: fmt.Sprintf("%s standard %s", f.Metric, f.Standard.String())},
			})
		}
		msg := fmt.Sprintf("%s actual %s, standard %s", f.Metric, FormatValue(f.Actual), f.Standard.String())
		if f.Approved != nil {
			msg = fmt.Sprintf("%s, approved %s (%s)", msg, f.Approved.String(), f.Kind)
		}
		if f.WarningBoundary != "" && f.WarningPercent != nil {
			msg = fmt.Sprintf("%s, within %s%% of %s", msg, FormatValue(*f.WarningPercent), f.WarningBoundary)
		}
		results = append(results, sarifResult{
			RuleID:  f.Metric,
			Level:   level,
			Message: sarifMessage{Text: msg},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: f.Subject},
				},
			}},
		})
	}
	for _, f := range report.Violations {
		add(f, "error")
	}
	for _, f := range report.Warnings {
		add(f, "warning")
	}
	for _, f := range report.Variances {
		add(f, "note")
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "Considered",
				InformationURI: "https://github.com/quitepicky/considered",
				Rules:          rules,
			}},
			Results: results,
		}},
	})
}
