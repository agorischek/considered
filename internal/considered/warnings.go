package considered

func (w WarningConfig) validationProblems() []string {
	var problems []string
	if w.PercentBelowMax != nil {
		problems = append(problems, validateWarningPercent(
			*w.PercentBelowMax,
			"warnings.percentBelowMax",
		)...)
	}
	if w.PercentAboveMin != nil {
		problems = append(problems, validateWarningPercent(
			*w.PercentAboveMin,
			"warnings.percentAboveMin",
		)...)
	}
	return problems
}

func validateWarningPercent(percent float64, field string) []string {
	if percent < 0 || percent > 100 {
		return []string{field + " must be between 0 and 100"}
	}
	return nil
}

func warningFinding(finding Finding, boundary Boundary, warnings WarningConfig) (Finding, bool) {
	if boundary.Max != nil && warnings.PercentBelowMax != nil && *boundary.Max > 0 {
		threshold := *boundary.Max * (1 - *warnings.PercentBelowMax/100)
		if finding.Actual >= threshold && finding.Actual <= *boundary.Max {
			finding.WarningBoundary = "max"
			finding.WarningPercent = warnings.PercentBelowMax
			return finding, true
		}
	}
	if boundary.Min != nil && warnings.PercentAboveMin != nil && *boundary.Min > 0 {
		threshold := *boundary.Min * (1 + *warnings.PercentAboveMin/100)
		if finding.Actual >= *boundary.Min && finding.Actual <= threshold {
			finding.WarningBoundary = "min"
			finding.WarningPercent = warnings.PercentAboveMin
			return finding, true
		}
	}
	return Finding{}, false
}
