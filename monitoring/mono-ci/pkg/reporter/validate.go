package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"monorepo/monitoring/mono-ci/pkg/runner"
)

// Config defines quality gates for validation.
type Config struct {
	MinCoverage        float64            `json:"min_coverage"`
	FailOnVetErrors    bool               `json:"fail_on_vet_errors"`
	PerModuleCoverage  map[string]float64 `json:"per_module_coverage,omitempty"`
	ExcludeFromCoverage []string           `json:"exclude_from_coverage,omitempty"`
}

// DefaultConfig returns a reasonable default configuration.
func DefaultConfig() Config {
	return Config{
		MinCoverage:     0,
		FailOnVetErrors: true,
	}
}

// LoadConfig reads a config file, falling back to defaults if not found.
func LoadConfig(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}
	return cfg
}

// Violation represents a quality gate failure.
type Violation struct {
	Module  string
	Rule    string
	Message string
}

// ValidateResults checks test results against config thresholds.
func ValidateResults(testSummary runner.Summary, vetSummary *runner.Summary, cfg Config) []Violation {
	var violations []Violation

	excludeSet := make(map[string]bool)
	for _, name := range cfg.ExcludeFromCoverage {
		excludeSet[name] = true
	}

	// Check per-module coverage
	for _, r := range testSummary.Results {
		if excludeSet[r.Module.Name] {
			continue
		}
		if r.Coverage < 0 {
			continue // no test files, skip
		}

		threshold := cfg.MinCoverage
		if perModule, ok := cfg.PerModuleCoverage[r.Module.Name]; ok {
			threshold = perModule
		}

		if threshold > 0 && r.Coverage < threshold {
			violations = append(violations, Violation{
				Module:  r.Module.Name,
				Rule:    "coverage",
				Message: fmt.Sprintf("coverage %.1f%% below threshold %.1f%%", r.Coverage, threshold),
			})
		}
	}

	// Check for test failures
	for _, r := range testSummary.Results {
		if !r.Passed {
			violations = append(violations, Violation{
				Module:  r.Module.Name,
				Rule:    "test",
				Message: fmt.Sprintf("tests failed: %s", r.Error),
			})
		}
	}

	// Check vet results
	if vetSummary != nil && cfg.FailOnVetErrors {
		for _, r := range vetSummary.Results {
			if !r.Passed {
				violations = append(violations, Violation{
					Module:  r.Module.Name,
					Rule:    "vet",
					Message: fmt.Sprintf("go vet failed: %s", r.Error),
				})
			}
		}
	}

	return violations
}

// FormatViolations produces a human-readable report of violations.
func FormatViolations(violations []Violation) string {
	if len(violations) == 0 {
		return "All quality gates passed.\n"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n%d quality gate violation(s) found:\n\n", len(violations)))

	for i, v := range violations {
		b.WriteString(fmt.Sprintf("  %d. [%s] %s: %s\n", i+1, v.Rule, v.Module, v.Message))
	}

	return b.String()
}
