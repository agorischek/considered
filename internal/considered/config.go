package considered

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

var BuiltInKinds = []string{"architectural", "debt", "generated"}

var excludeCategoryPatterns = map[string][]string{
	"assets": {
		"assets/**",
		"**/assets/**",
		"branding/**",
		"**/branding/**",
		"fonts/**",
		"**/fonts/**",
		"images/**",
		"**/images/**",
		"public/**",
		"**/public/**",
		"static/**",
		"**/static/**",
		"**/*.avif",
		"**/*.bmp",
		"**/*.eot",
		"**/*.fig",
		"**/*.flac",
		"**/*.gif",
		"**/*.ico",
		"**/*.icns",
		"**/*.jpeg",
		"**/*.jpg",
		"**/*.mov",
		"**/*.mp3",
		"**/*.mp4",
		"**/*.ogg",
		"**/*.otf",
		"**/*.pdf",
		"**/*.png",
		"**/*.psd",
		"**/*.sketch",
		"**/*.svg",
		"**/*.tif",
		"**/*.tiff",
		"**/*.ttf",
		"**/*.wav",
		"**/*.webm",
		"**/*.webp",
		"**/*.woff",
		"**/*.woff2",
	},
	"dependencies": {
		"node_modules/**",
		"**/node_modules/**",
		"bower_components/**",
		"**/bower_components/**",
		"jspm_packages/**",
		"**/jspm_packages/**",
		".pnpm-store/**",
		"**/.pnpm-store/**",
		".yarn/**",
		"**/.yarn/**",
		".venv/**",
		"**/.venv/**",
		"venv/**",
		"**/venv/**",
		"env/**",
		"**/env/**",
	},
	"generated": {
		"bun.lock",
		"**/bun.lock",
		"Cargo.lock",
		"**/Cargo.lock",
		"Cartfile.resolved",
		"**/Cartfile.resolved",
		"Gemfile.lock",
		"**/Gemfile.lock",
		"Package.resolved",
		"**/Package.resolved",
		"Pipfile.lock",
		"**/Pipfile.lock",
		"Podfile.lock",
		"**/Podfile.lock",
		"composer.lock",
		"**/composer.lock",
		"conan.lock",
		"**/conan.lock",
		"go.sum",
		"**/go.sum",
		"mix.lock",
		"**/mix.lock",
		"npm-shrinkwrap.json",
		"**/npm-shrinkwrap.json",
		"package-lock.json",
		"**/package-lock.json",
		"pnpm-lock.yaml",
		"**/pnpm-lock.yaml",
		"poetry.lock",
		"**/poetry.lock",
		"pubspec.lock",
		"**/pubspec.lock",
		"uv.lock",
		"**/uv.lock",
		"yarn.lock",
		"**/yarn.lock",
		"generated/**",
		"**/generated/**",
		"gen/**",
		"**/gen/**",
		"codegen/**",
		"**/codegen/**",
		"__generated__/**",
		"**/__generated__/**",
		"**/*.generated.*",
		"**/*.gen.*",
		"**/*.pb.go",
		"**/*.pb.swift",
		"**/*.pb.ts",
		"**/*.pb.js",
		"**/*_generated.go",
		"**/*_generated.rs",
		"**/*_generated.swift",
		"**/*_generated.ts",
		"**/*_generated.tsx",
	},
	"tests": {
		// Conventional test directories used by pytest, Cargo, Gradle, Mocha,
		// Playwright, RSpec, PHPUnit, and many language communities.
		"test/**",
		"**/test/**",
		"tests/**",
		"**/tests/**",
		"Tests/**",
		"**/Tests/**",
		"spec/**",
		"**/spec/**",
		"specs/**",
		"**/specs/**",
		"__tests__/**",
		"**/__tests__/**",

		// JavaScript and TypeScript runners: Jest, Vitest, Node test runner,
		// Bun, Deno, Mocha, Cypress, and Playwright.
		"**/*.test.js",
		"**/*.test.cjs",
		"**/*.test.mjs",
		"**/*.test.jsx",
		"**/*.test.ts",
		"**/*.test.cts",
		"**/*.test.mts",
		"**/*.test.tsx",
		"**/*.spec.js",
		"**/*.spec.cjs",
		"**/*.spec.mjs",
		"**/*.spec.jsx",
		"**/*.spec.ts",
		"**/*.spec.cts",
		"**/*.spec.mts",
		"**/*.spec.tsx",
		"**/*-test.js",
		"**/*-test.cjs",
		"**/*-test.mjs",
		"**/*-test.jsx",
		"**/*-test.ts",
		"**/*-test.cts",
		"**/*-test.mts",
		"**/*-test.tsx",
		"**/*_test.js",
		"**/*_test.cjs",
		"**/*_test.mjs",
		"**/*_test.jsx",
		"**/*_test.ts",
		"**/*_test.cts",
		"**/*_test.mts",
		"**/*_test.tsx",
		"**/*_spec.js",
		"**/*_spec.jsx",
		"**/*_spec.ts",
		"**/*_spec.tsx",
		"**/test-*.js",
		"**/test-*.cjs",
		"**/test-*.mjs",
		"**/test-*.ts",
		"**/test-*.cts",
		"**/test-*.mts",
		"**/test.js",
		"**/test.cjs",
		"**/test.mjs",
		"**/test.ts",
		"**/test.cts",
		"**/test.mts",
		"cypress/e2e/**",
		"**/cypress/e2e/**",
		"**/*.cy.js",
		"**/*.cy.jsx",
		"**/*.cy.ts",
		"**/*.cy.tsx",

		// Python unittest and pytest.
		"**/test*.py",
		"**/test_*.py",
		"**/*_test.py",

		// Go.
		"**/*_test.go",

		// Rust Cargo integration tests and Java/JVM source-set conventions.
		"src/test/**",
		"**/src/test/**",
		"src/integrationTest/**",
		"**/src/integrationTest/**",
		"src/functionalTest/**",
		"**/src/functionalTest/**",

		// Maven Surefire and common JUnit/TestNG class naming conventions.
		"**/Test*.java",
		"**/*Test.java",
		"**/*Tests.java",
		"**/*TestCase.java",
		"**/*IT.java",
		"**/*ITCase.java",

		// Ruby RSpec and PHP PHPUnit.
		"**/*_spec.rb",
		"**/*Test.php",
		"**/*.phpt",
	},
	"vendored": {
		"vendor/**",
		"**/vendor/**",
		"third_party/**",
		"**/third_party/**",
		"third-party/**",
		"**/third-party/**",
		"external/**",
		"**/external/**",
		"vendored/**",
		"**/vendored/**",
	},
}

func DefaultConfig() Config {
	return Config{
		Kinds: []string{"performance", "compatibility"},
		Exclude: ExcludeConfig{
			Gitignored: true,
			Categories: []string{
				"tests",
				"vendored",
				"dependencies",
			},
			Paths: []string{},
		},
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

func (e *ExcludeConfig) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var paths []string
		if err := value.Decode(&paths); err != nil {
			return err
		}
		e.Paths = paths
		return nil
	case yaml.MappingNode:
		type raw ExcludeConfig
		var decoded raw
		if err := value.Decode(&decoded); err != nil {
			return err
		}
		*e = ExcludeConfig(decoded)
		return nil
	case yaml.ScalarNode:
		if value.Tag == "!!null" {
			return nil
		}
	}
	return fmt.Errorf("exclude must be a mapping or a list of paths")
}

// IsExcluded reports whether the subject matches configured category or path
// exclusions. Gitignore exclusions need repository context and are applied by
// FilterRecords.
func (c Config) IsExcluded(subject string) bool {
	subject = normalizeSubject(subject)
	for _, pattern := range c.Exclude.Patterns() {
		if ok, err := doublestar.Match(pattern, subject); err == nil && ok {
			return true
		}
	}
	return false
}

func (c Config) FilterRecords(root string, records []MetricRecord) []MetricRecord {
	ignored := map[string]bool{}
	if c.Exclude.Gitignored {
		ignored = gitignoredSubjects(root, records)
	}
	filtered := records[:0]
	for _, record := range records {
		subject := normalizeSubject(record.Subject)
		if ignored[subject] || c.IsExcluded(subject) {
			continue
		}
		record.Subject = subject
		filtered = append(filtered, record)
	}
	return filtered
}

func (e ExcludeConfig) Patterns() []string {
	var patterns []string
	for _, category := range e.Categories {
		patterns = append(patterns, excludeCategoryPatterns[category]...)
	}
	patterns = append(patterns, e.Paths...)
	return patterns
}

func (e ExcludeConfig) Runtime() ExcludeRuntime {
	directories := map[string]bool{".git": true}
	for _, pattern := range e.Patterns() {
		if dir, ok := directoryDenyFromPattern(pattern); ok {
			directories[dir] = true
		}
	}
	result := ExcludeRuntime{Directories: make([]string, 0, len(directories))}
	for directory := range directories {
		result.Directories = append(result.Directories, directory)
	}
	sort.Strings(result.Directories)
	return result
}

func KnownExcludeCategories() []string {
	categories := make([]string, 0, len(excludeCategoryPatterns))
	for category := range excludeCategoryPatterns {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return categories
}

func directoryDenyFromPattern(pattern string) (string, bool) {
	pattern = strings.Trim(pattern, "/")
	pattern = strings.TrimPrefix(pattern, "**/")
	if strings.HasSuffix(pattern, "/**") {
		dir := strings.TrimSuffix(pattern, "/**")
		if !strings.ContainsAny(dir, "*?[{") {
			return dir, true
		}
	}
	return "", false
}

func (c Config) Validate() error {
	var problems []string
	for _, category := range c.Exclude.Categories {
		if _, ok := excludeCategoryPatterns[category]; !ok {
			problems = append(
				problems,
				fmt.Sprintf(
					"exclude category %q is unknown; known categories are: %s",
					category,
					strings.Join(KnownExcludeCategories(), ", "),
				),
			)
		}
	}
	for _, pattern := range c.Exclude.Paths {
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

func gitignoredSubjects(root string, records []MetricRecord) map[string]bool {
	subjects := make([]string, 0, len(records))
	seen := map[string]bool{}
	for _, record := range records {
		subject := normalizeSubject(record.Subject)
		if !seen[subject] {
			seen[subject] = true
			subjects = append(subjects, subject)
		}
	}
	if len(subjects) == 0 {
		return map[string]bool{}
	}

	cmd := exec.Command("git", "-C", root, "check-ignore", "-z", "--stdin")
	cmd.Stdin = strings.NewReader(strings.Join(subjects, "\x00") + "\x00")
	output, err := cmd.Output()
	if err != nil && len(output) == 0 {
		return map[string]bool{}
	}
	ignored := map[string]bool{}
	for _, subject := range strings.Split(strings.TrimRight(string(output), "\x00"), "\x00") {
		if subject != "" {
			ignored[normalizeSubject(subject)] = true
		}
	}
	return ignored
}

func normalizeSubject(subject string) string {
	subject = filepath.ToSlash(filepath.Clean(subject))
	if subject == "." {
		return ""
	}
	return strings.TrimPrefix(subject, "./")
}
