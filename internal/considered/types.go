package considered

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

const ConfigName = ".considered.yaml"

type Boundary struct {
	Min *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max *float64 `json:"max,omitempty" yaml:"max,omitempty"`
}

type Config struct {
	Kinds     []string            `json:"kinds,omitempty" yaml:"kinds,omitempty"`
	Exclude   ExcludeConfig       `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	Standards map[string]Boundary `json:"standards,omitempty" yaml:"standards,omitempty"`
	Variances map[string]Variance `json:"variances,omitempty" yaml:"variances,omitempty"`
}

type ExcludeConfig struct {
	Gitignored bool     `json:"gitignored,omitempty" yaml:"gitignored,omitempty"`
	Categories []string `json:"categories,omitempty" yaml:"categories,omitempty"`
	Paths      []string `json:"paths,omitempty" yaml:"paths,omitempty"`
}

type Variance struct {
	Kind    string                    `json:"kind" yaml:"kind"`
	Reason  string                    `json:"reason" yaml:"reason"`
	Metrics map[string]VarianceMetric `json:"metrics,omitempty" yaml:"metrics,omitempty"`
}

type VarianceMetric struct {
	Boundary `json:",inline" yaml:",inline"`
	Reason   string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type MetricRecord struct {
	Subject  string             `json:"subject" yaml:"subject"`
	Provider string             `json:"provider,omitempty" yaml:"provider,omitempty"`
	Values   map[string]float64 `json:"values" yaml:"values"`
}

type ProviderOutput struct {
	Metrics []MetricRecord `json:"metrics"`
}

type Finding struct {
	Subject          string    `json:"subject"`
	Metric           string    `json:"metric"`
	Actual           float64   `json:"actual"`
	Standard         Boundary  `json:"standard"`
	Approved         *Boundary `json:"approved,omitempty"`
	Kind             string    `json:"kind,omitempty"`
	Reason           string    `json:"reason,omitempty"`
	MetricReason     string    `json:"metric_reason,omitempty"`
	Provider         string    `json:"provider,omitempty"`
	VarianceExceeded bool      `json:"variance_exceeded,omitempty"`
}

type Report struct {
	Violations []Finding `json:"violations"`
	Variances  []Finding `json:"variances"`
}

func (r Report) HasViolations() bool {
	return len(r.Violations) > 0
}

func (b Boundary) String() string {
	parts := []string{}
	if b.Min != nil {
		parts = append(parts, fmt.Sprintf(">= %s", formatNumber(*b.Min)))
	}
	if b.Max != nil {
		parts = append(parts, fmt.Sprintf("<= %s", formatNumber(*b.Max)))
	}
	if len(parts) == 0 {
		return "(unbounded)"
	}
	return strings.Join(parts, " and ")
}

func (b Boundary) Allows(v float64) bool {
	if b.Min != nil && v < *b.Min {
		return false
	}
	if b.Max != nil && v > *b.Max {
		return false
	}
	return true
}

func (b Boundary) Empty() bool {
	return b.Min == nil && b.Max == nil
}

func formatNumber(v float64) string {
	if math.Trunc(v) == v {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%g", v)
}

func FormatValue(v float64) string {
	return formatNumber(v)
}

func Prefix(metric string) string {
	before, _, ok := strings.Cut(metric, ".")
	if !ok {
		return ""
	}
	return before
}

func SortedMetrics(m map[string]Boundary) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func MarshalProviderOutput(records []MetricRecord) ([]byte, error) {
	return json.MarshalIndent(ProviderOutput{Metrics: records}, "", "  ")
}
