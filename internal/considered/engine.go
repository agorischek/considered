package considered

import (
	"context"
	"sort"
)

func Check(ctx context.Context, root string, cfg Config) (Report, error) {
	providers := ProvidersWithExcludes(ProvidersForStandards(cfg.Standards), cfg.Exclude.Runtime())
	records, err := CollectAll(ctx, root, providers)
	if err != nil {
		return Report{}, err
	}
	records = cfg.FilterRecords(root, records)
	return Evaluate(cfg, records), nil
}

func Evaluate(cfg Config, records []MetricRecord) Report {
	report := Report{
		Violations: []Finding{},
		Warnings:   []Finding{},
		Variances:  []Finding{},
	}
	for _, record := range records {
		if cfg.IsExcluded(record.Subject) {
			continue
		}
		for metric, actual := range record.Values {
			standard, ok := cfg.Standards[metric]
			if !ok {
				continue
			}
			finding := Finding{
				Subject:  record.Subject,
				Metric:   metric,
				Actual:   actual,
				Standard: standard,
				Provider: record.Provider,
			}
			if standard.Allows(actual) {
				if warning, ok := warningFinding(finding, standard, cfg.WarningThresholds); ok {
					report.Warnings = append(report.Warnings, warning)
				}
				continue
			}
			if variance, ok := cfg.Variances[record.Subject]; ok {
				if override, ok := variance.Metrics[metric]; ok {
					approved := override.Boundary
					finding.Approved = &approved
					finding.Kind = variance.Kind
					finding.Reason = variance.Reason
					finding.MetricReason = override.Reason
					if approved.Allows(actual) {
						report.Variances = append(report.Variances, finding)
						if warning, ok := warningFinding(finding, approved, cfg.WarningThresholds); ok {
							report.Warnings = append(report.Warnings, warning)
						}
						continue
					}
					finding.VarianceExceeded = true
				}
			}
			report.Violations = append(report.Violations, finding)
		}
	}
	sortFindings(report.Violations)
	sortFindings(report.Warnings)
	sortFindings(report.Variances)
	return report
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Subject == findings[j].Subject {
			return findings[i].Metric < findings[j].Metric
		}
		return findings[i].Subject < findings[j].Subject
	})
}

func MetricRecordsBySubject(records []MetricRecord) map[string]map[string]float64 {
	result := map[string]map[string]float64{}
	for _, record := range records {
		if result[record.Subject] == nil {
			result[record.Subject] = map[string]float64{}
		}
		for key, value := range record.Values {
			result[record.Subject][key] = value
		}
	}
	return result
}
