package considered

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

var BuiltInKinds = []string{"architectural", "debt", "generated"}

func DefaultConfig() Config {
	return Config{
		Kinds: []string{"performance", "compatibility"},
		Standards: map[string]Boundary{
			"scc.code_lines":          {Max: Float64(500)},
			"scc.complexity":          {Max: Float64(50)},
			"filesystem.bytes":        {Max: Float64(50000)},
			"filesystem.longest_line": {Max: Float64(140)},
		},
		Variances: map[string]Variance{},
	}
}

func Float64(v float64) *float64 {
	return &v
}

func LoadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Standards == nil {
		cfg.Standards = map[string]Boundary{}
	}
	if cfg.Variances == nil {
		cfg.Variances = map[string]Variance{}
	}
	return cfg, cfg.Validate()
}

func SaveConfig(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func WriteDefaultConfig(root string) (string, error) {
	path := filepath.Join(root, ConfigName)
	if _, err := os.Stat(path); err == nil {
		return path, fmt.Errorf("%s already exists", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return path, err
	}
	return path, SaveConfig(path, DefaultConfig())
}

// IsExcluded reports whether the subject matches any exclude glob and so
// should be left unmeasured. Patterns use doublestar semantics, so "**"
// crosses directory boundaries (e.g. "**/*_test.go").
func (c Config) IsExcluded(subject string) bool {
	for _, pattern := range c.Exclude {
		if ok, err := doublestar.Match(pattern, subject); err == nil && ok {
			return true
		}
	}
	return false
}

func (c Config) Validate() error {
	var problems []string
	for _, pattern := range c.Exclude {
		if !doublestar.ValidatePattern(pattern) {
			problems = append(problems, fmt.Sprintf("exclude pattern %q is not a valid glob", pattern))
		}
	}
	for metric, boundary := range c.Standards {
		if Prefix(metric) == "" {
			problems = append(problems, fmt.Sprintf("standard %q must be namespaced", metric))
		}
		if boundary.Empty() {
			problems = append(problems, fmt.Sprintf("standard %q must define min or max", metric))
		}
		if boundary.Min != nil && boundary.Max != nil && *boundary.Min > *boundary.Max {
			problems = append(problems, fmt.Sprintf("standard %q min cannot exceed max", metric))
		}
	}

	kinds := map[string]bool{}
	for _, kind := range BuiltInKinds {
		kinds[kind] = true
	}
	for _, kind := range c.Kinds {
		kinds[kind] = true
	}

	for subject, variance := range c.Variances {
		if strings.TrimSpace(subject) == "" {
			problems = append(problems, "variance subject cannot be empty")
		}
		if strings.TrimSpace(variance.Kind) == "" {
			problems = append(problems, fmt.Sprintf("variance %q must define kind", subject))
		} else if !kinds[variance.Kind] {
			problems = append(problems, fmt.Sprintf("variance %q has unknown kind %q", subject, variance.Kind))
		}
		if strings.TrimSpace(variance.Reason) == "" {
			problems = append(problems, fmt.Sprintf("variance %q must define reason", subject))
		}
		for metric, override := range variance.Metrics {
			if Prefix(metric) == "" {
				problems = append(problems, fmt.Sprintf("variance %q metric %q must be namespaced", subject, metric))
			}
			if override.Boundary.Empty() {
				problems = append(problems, fmt.Sprintf("variance %q metric %q must define min or max", subject, metric))
			}
			if override.Min != nil && override.Max != nil && *override.Min > *override.Max {
				problems = append(problems, fmt.Sprintf("variance %q metric %q min cannot exceed max", subject, metric))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

// Warnings reports non-fatal configuration issues — things that are valid but
// almost certainly mistakes, such as a variance overriding a metric that has
// no standard (and so can never apply).
func (c Config) Warnings() []string {
	var warnings []string
	for subject, variance := range c.Variances {
		if c.IsExcluded(subject) {
			warnings = append(warnings, fmt.Sprintf("variance %q is also excluded; it will never apply", subject))
		}
		for metric := range variance.Metrics {
			if _, ok := c.Standards[metric]; !ok {
				warnings = append(warnings, fmt.Sprintf("variance %q overrides metric %q which has no standard; it will never apply", subject, metric))
			}
		}
	}
	sort.Strings(warnings)
	return warnings
}
