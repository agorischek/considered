package considered

import (
	"os"
	"os/exec"
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
		Exclude: ExcludeConfig{Categories: []string{"typo"}},
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
	for _, want := range []string{"must be namespaced", "unknown kind", "must define reason", "exclude category"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in %q", want, msg)
		}
	}
}

func TestConfigValidateRejectsBadExcludePathGlob(t *testing.T) {
	cfg := Config{
		Exclude:   ExcludeConfig{Paths: []string{"["}},
		Standards: map[string]Boundary{"filesystem.bytes": {Max: Float64(1)}},
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "exclude pattern") {
		t.Fatalf("expected exclude pattern error, got %v", err)
	}
}

func TestConfigValidateRejectsWarningPercentOutsideRange(t *testing.T) {
	cfg := Config{
		Standards: map[string]Boundary{"filesystem.bytes": {Max: Float64(1)}},
		WarningThresholds: WarningConfig{
			PercentBelowMax: Float64(101),
			PercentAboveMin: Float64(-1),
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected warning threshold validation errors")
	}
	for _, want := range []string{"warnings.percentBelowMax", "warnings.percentAboveMin"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
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
	if loaded.WarningThresholds.PercentBelowMax == nil || *loaded.WarningThresholds.PercentBelowMax != 10 {
		t.Fatalf("warning threshold was not round-tripped: %#v", loaded.WarningThresholds)
	}
}

func TestLoadConfigAcceptsLegacyExcludePathList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigName)
	if err := os.WriteFile(path, []byte("exclude:\n  - src/generated/**\nstandards:\n  filesystem.bytes:\n    max: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Exclude.Paths) != 1 || cfg.Exclude.Paths[0] != "src/generated/**" {
		t.Fatalf("legacy exclude list was not loaded as paths: %#v", cfg.Exclude)
	}
}

func TestExcludeCategoriesAndPaths(t *testing.T) {
	cfg := Config{Exclude: ExcludeConfig{
		Categories: []string{"assets", "documentation", "generated", "tests", "vendored", "dependencies"},
		Paths:      []string{"src/generated/**"},
	}}
	excluded := []string{
		"branding/studio-primer.svg",
		"bun.lock",
		"Cargo.lock",
		"CHANGELOG.md",
		"CODE_OF_CONDUCT.md",
		"CONTRIBUTING.md",
		"LICENSE",
		"NOTICE.txt",
		"README.md",
		"SECURITY.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		"docs/guide.md",
		"manual/reference.rst",
		"package-lock.json",
		"pnpm-lock.yaml",
		"public/favicon.ico",
		"src/assets/logo.png",
		"src/__generated__/schema.ts",
		"src/api/client.generated.ts",
		"src/api/service.pb.go",
		"ExampleTest.php",
		"Tests/AppTests.swift",
		"cypress/e2e/login.cy.ts",
		"example_test.go",
		"package.test.mjs",
		"src/FooTest.java",
		"src/test/kotlin/FooTest.kt",
		"src/foo_test.go",
		"test_math.py",
		"tests/thing.go",
		"third_party/lib/main.go",
		"node_modules/pkg/index.js",
		"src/generated/schema.go",
	}
	for _, subject := range excluded {
		if !cfg.IsExcluded(subject) {
			t.Fatalf("expected %s to be excluded", subject)
		}
	}
	if cfg.IsExcluded("src/main.go") {
		t.Fatal("src/main.go should not be excluded")
	}
}

func TestFilterRecordsHonorsGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored.txt\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "init")
	records := []MetricRecord{
		{Subject: "ignored.txt", Values: map[string]float64{"filesystem.bytes": 1}},
		{Subject: "kept.txt", Values: map[string]float64{"filesystem.bytes": 1}},
	}
	filtered := (Config{Exclude: ExcludeConfig{Gitignored: true}}).FilterRecords(dir, records)
	if len(filtered) != 1 || filtered[0].Subject != "kept.txt" {
		t.Fatalf("unexpected filtered records: %#v", filtered)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
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
