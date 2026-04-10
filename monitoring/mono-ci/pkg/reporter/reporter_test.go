package reporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"monorepo/monitoring/mono-ci/pkg/discovery"
	"monorepo/monitoring/mono-ci/pkg/runner"
)

func TestGenerateJSON(t *testing.T) {
	t.Parallel()

	s := runner.Summary{
		Results: []runner.Result{
			{
				Module:   discovery.Module{Name: "bin-api-manager"},
				Action:   "test",
				Passed:   true,
				Coverage: 85.5,
				Duration: 2 * time.Second,
			},
			{
				Module:   discovery.Module{Name: "bin-call-manager"},
				Action:   "test",
				Passed:   false,
				Coverage: -1,
				Duration: 1 * time.Second,
				Error:    "exit status 1",
			},
		},
		TotalDuration: 3 * time.Second,
		PassedCount:   1,
		FailedCount:   1,
		AvgCoverage:   85.5,
	}

	data, err := GenerateJSON(s)
	if err != nil {
		t.Fatalf("GenerateJSON failed: %v", err)
	}

	var report JSONReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if report.Action != "test" {
		t.Errorf("Action = %q, want 'test'", report.Action)
	}
	if report.TotalModules != 2 {
		t.Errorf("TotalModules = %d, want 2", report.TotalModules)
	}
	if report.Passed != 1 {
		t.Errorf("Passed = %d, want 1", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("Failed = %d, want 1", report.Failed)
	}
	if len(report.Modules) != 2 {
		t.Errorf("len(Modules) = %d, want 2", len(report.Modules))
	}
}

func TestValidateResults_CoverageThreshold(t *testing.T) {
	t.Parallel()

	s := runner.Summary{
		Results: []runner.Result{
			{Module: discovery.Module{Name: "a"}, Passed: true, Coverage: 60.0, Action: "test"},
			{Module: discovery.Module{Name: "b"}, Passed: true, Coverage: 85.0, Action: "test"},
		},
		PassedCount: 2,
	}

	cfg := Config{MinCoverage: 80.0, FailOnVetErrors: true}
	violations := ValidateResults(s, nil, cfg)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Module != "a" {
		t.Errorf("expected violation for module 'a', got %q", violations[0].Module)
	}
	if violations[0].Rule != "coverage" {
		t.Errorf("expected rule 'coverage', got %q", violations[0].Rule)
	}
}

func TestValidateResults_TestFailure(t *testing.T) {
	t.Parallel()

	s := runner.Summary{
		Results: []runner.Result{
			{Module: discovery.Module{Name: "a"}, Passed: false, Error: "exit status 1", Action: "test"},
		},
		FailedCount: 1,
	}

	cfg := DefaultConfig()
	violations := ValidateResults(s, nil, cfg)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Rule != "test" {
		t.Errorf("expected rule 'test', got %q", violations[0].Rule)
	}
}

func TestValidateResults_VetFailure(t *testing.T) {
	t.Parallel()

	testSummary := runner.Summary{
		Results: []runner.Result{
			{Module: discovery.Module{Name: "a"}, Passed: true, Coverage: 90.0, Action: "test"},
		},
		PassedCount: 1,
	}

	vetSummary := runner.Summary{
		Results: []runner.Result{
			{Module: discovery.Module{Name: "a"}, Passed: false, Error: "vet error", Action: "vet"},
		},
		FailedCount: 1,
	}

	cfg := Config{FailOnVetErrors: true}
	violations := ValidateResults(testSummary, &vetSummary, cfg)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Rule != "vet" {
		t.Errorf("expected rule 'vet', got %q", violations[0].Rule)
	}
}

func TestValidateResults_ExcludeFromCoverage(t *testing.T) {
	t.Parallel()

	s := runner.Summary{
		Results: []runner.Result{
			{Module: discovery.Module{Name: "a"}, Passed: true, Coverage: 10.0, Action: "test"},
			{Module: discovery.Module{Name: "b"}, Passed: true, Coverage: 90.0, Action: "test"},
		},
		PassedCount: 2,
	}

	cfg := Config{MinCoverage: 80.0, ExcludeFromCoverage: []string{"a"}}
	violations := ValidateResults(s, nil, cfg)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations (module 'a' excluded), got %d", len(violations))
	}
}

func TestValidateResults_PerModuleCoverage(t *testing.T) {
	t.Parallel()

	s := runner.Summary{
		Results: []runner.Result{
			{Module: discovery.Module{Name: "a"}, Passed: true, Coverage: 70.0, Action: "test"},
			{Module: discovery.Module{Name: "b"}, Passed: true, Coverage: 70.0, Action: "test"},
		},
		PassedCount: 2,
	}

	cfg := Config{
		MinCoverage:       80.0,
		PerModuleCoverage: map[string]float64{"a": 60.0},
	}
	violations := ValidateResults(s, nil, cfg)

	// Module "a" has per-module threshold of 60%, so it passes (70% > 60%)
	// Module "b" uses default threshold of 80%, so it fails (70% < 80%)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Module != "b" {
		t.Errorf("expected violation for module 'b', got %q", violations[0].Module)
	}
}

func TestFormatViolations_None(t *testing.T) {
	t.Parallel()

	output := FormatViolations(nil)
	if output != "All quality gates passed.\n" {
		t.Errorf("unexpected output: %q", output)
	}
}

func TestFormatViolations_WithViolations(t *testing.T) {
	t.Parallel()

	violations := []Violation{
		{Module: "a", Rule: "coverage", Message: "coverage 60.0% below threshold 80.0%"},
		{Module: "b", Rule: "test", Message: "tests failed"},
	}

	output := FormatViolations(violations)
	if output == "" {
		t.Error("FormatViolations returned empty string")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	t.Parallel()

	cfg := LoadConfig("/nonexistent/config.json")
	def := DefaultConfig()
	if cfg.MinCoverage != def.MinCoverage {
		t.Errorf("expected default MinCoverage, got %.1f", cfg.MinCoverage)
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	data := `{"min_coverage": 75.0, "fail_on_vet_errors": false, "exclude_from_coverage": ["bin-dbscheme-manager"]}`
	os.WriteFile(cfgPath, []byte(data), 0644)

	cfg := LoadConfig(cfgPath)
	if cfg.MinCoverage != 75.0 {
		t.Errorf("MinCoverage = %.1f, want 75.0", cfg.MinCoverage)
	}
	if cfg.FailOnVetErrors {
		t.Error("FailOnVetErrors should be false")
	}
	if len(cfg.ExcludeFromCoverage) != 1 || cfg.ExcludeFromCoverage[0] != "bin-dbscheme-manager" {
		t.Errorf("ExcludeFromCoverage = %v, want [bin-dbscheme-manager]", cfg.ExcludeFromCoverage)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	os.WriteFile(cfgPath, []byte("{invalid json"), 0644)

	cfg := LoadConfig(cfgPath)
	def := DefaultConfig()
	if cfg.MinCoverage != def.MinCoverage {
		t.Errorf("expected default on invalid JSON, got %.1f", cfg.MinCoverage)
	}
}
