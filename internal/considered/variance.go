package considered

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
)

type VarianceAddOptions struct {
	ConfigPath   string
	Root         string
	Subject      string
	Metric       string
	Kind         string
	Reason       string
	MetricReason string
	In           io.Reader
	Out          io.Writer
}

func AddVariance(ctx context.Context, opts VarianceAddOptions) error {
	cfg, err := LoadConfig(opts.ConfigPath)
	if err != nil {
		return err
	}
	records, err := CollectAll(ctx, opts.Root, ProvidersForStandards(cfg.Standards))
	if err != nil {
		return err
	}
	bySubject := MetricRecordsBySubject(records)
	prompt := newPrompter(opts.In, opts.Out)

	subject := strings.TrimSpace(opts.Subject)
	if subject == "" {
		subjects := sortedSubjectKeys(bySubject)
		subject, err = prompt.choice("subject", subjects)
		if err != nil {
			return err
		}
	}
	if cfg.IsExcluded(subject) {
		return fmt.Errorf("subject %q is excluded by configuration", subject)
	}
	metrics, ok := bySubject[subject]
	if !ok {
		return fmt.Errorf("subject %q was not measured", subject)
	}

	metric := strings.TrimSpace(opts.Metric)
	if metric == "" {
		metric, err = prompt.choice("metric", sortedMetricKeys(metrics))
		if err != nil {
			return err
		}
	}
	actual, ok := metrics[metric]
	if !ok {
		return fmt.Errorf("metric %q was not measured for %q", metric, subject)
	}
	standard, ok := cfg.Standards[metric]
	if !ok {
		return fmt.Errorf("metric %q does not have a standard", metric)
	}
	if standard.Allows(actual) {
		return fmt.Errorf("metric %q value %s already satisfies standard %s; no variance needed", metric, FormatValue(actual), standard.String())
	}

	kind := strings.TrimSpace(opts.Kind)
	if kind == "" {
		kind, err = prompt.text("kind")
		if err != nil {
			return err
		}
	}
	reason := strings.TrimSpace(opts.Reason)
	if reason == "" {
		reason, err = prompt.text("reason")
		if err != nil {
			return err
		}
	}

	if cfg.Variances == nil {
		cfg.Variances = map[string]Variance{}
	}
	variance, existed := cfg.Variances[subject]
	if existed && opts.Out != nil {
		if variance.Kind != "" && variance.Kind != kind {
			fmt.Fprintf(opts.Out, "warning: changing kind for %s from %q to %q\n", subject, variance.Kind, kind)
		}
		if variance.Reason != "" && variance.Reason != reason {
			fmt.Fprintf(opts.Out, "warning: replacing reason for %s\n", subject)
		}
	}
	variance.Kind = kind
	variance.Reason = reason
	if variance.Metrics == nil {
		variance.Metrics = map[string]VarianceMetric{}
	}
	variance.Metrics[metric] = VarianceMetric{
		Boundary: boundaryForActual(standard, actual),
		Reason:   strings.TrimSpace(opts.MetricReason),
	}
	cfg.Variances[subject] = variance
	if err := cfg.Validate(); err != nil {
		return err
	}
	return SaveConfig(opts.ConfigPath, cfg)
}

func boundaryForActual(standard Boundary, actual float64) Boundary {
	if standard.Max != nil && actual > *standard.Max {
		return Boundary{Max: Float64(actual)}
	}
	if standard.Min != nil && actual < *standard.Min {
		return Boundary{Min: Float64(actual)}
	}
	if standard.Max != nil {
		return Boundary{Max: Float64(actual)}
	}
	return Boundary{Min: Float64(actual)}
}

type prompter struct {
	scanner *bufio.Scanner
	out     io.Writer
}

func newPrompter(in io.Reader, out io.Writer) prompter {
	if in == nil {
		in = strings.NewReader("")
	}
	return prompter{scanner: bufio.NewScanner(in), out: out}
}

func (p prompter) choice(label string, values []string) (string, error) {
	if len(values) == 0 {
		return "", fmt.Errorf("no %s choices are available", label)
	}
	if p.out != nil {
		for i, value := range values {
			fmt.Fprintf(p.out, "%d. %s\n", i+1, value)
		}
		fmt.Fprintf(p.out, "%s: ", label)
	}
	if !p.scanner.Scan() {
		return "", fmt.Errorf("%s is required", label)
	}
	answer := strings.TrimSpace(p.scanner.Text())
	for i, value := range values {
		if answer == fmt.Sprint(i+1) || answer == value {
			return value, nil
		}
	}
	return "", fmt.Errorf("unknown %s %q", label, answer)
}

func (p prompter) text(label string) (string, error) {
	if p.out != nil {
		fmt.Fprintf(p.out, "%s: ", label)
	}
	if !p.scanner.Scan() {
		return "", fmt.Errorf("%s is required", label)
	}
	value := strings.TrimSpace(p.scanner.Text())
	if value == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return value, nil
}

func sortedSubjectKeys(m map[string]map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMetricKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
