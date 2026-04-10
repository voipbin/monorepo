package runner

import (
	"testing"
	"time"

	"monorepo/monitoring/mono-ci/pkg/discovery"
)

func TestExtractCoverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   string
		expected float64
	}{
		{
			name:     "single package",
			output:   "ok  \tpkg/handler\t0.5s\tcoverage: 85.3% of statements",
			expected: 85.3,
		},
		{
			name: "multiple packages",
			output: `ok  	pkg/handler	0.5s	coverage: 80.0% of statements
ok  	pkg/service	0.3s	coverage: 90.0% of statements`,
			expected: 85.0,
		},
		{
			name:     "no test files",
			output:   "?   \tpkg/types\t[no test files]",
			expected: -1,
		},
		{
			name:     "no coverage info",
			output:   "some random output",
			expected: -1,
		},
		{
			name:     "empty output",
			output:   "",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractCoverage(tt.output)
			if got != tt.expected {
				t.Errorf("extractCoverage() = %.1f, want %.1f", got, tt.expected)
			}
		})
	}
}

func TestCountTests(t *testing.T) {
	t.Parallel()

	output := `ok  	pkg/handler	0.5s
ok  	pkg/service	0.3s
FAIL	pkg/broken	0.1s
--- SKIP: TestSkipped (0.00s)`

	total, failed, skipped := countTests(output)
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
}

func TestBuildSummary(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Module: discovery.Module{Name: "a"}, Passed: true, Coverage: 80.0, Action: "test"},
		{Module: discovery.Module{Name: "b"}, Passed: true, Coverage: 90.0, Action: "test"},
		{Module: discovery.Module{Name: "c"}, Passed: false, Coverage: -1, Action: "test"},
	}

	s := buildSummary(results, 5*time.Second)

	if s.PassedCount != 2 {
		t.Errorf("PassedCount = %d, want 2", s.PassedCount)
	}
	if s.FailedCount != 1 {
		t.Errorf("FailedCount = %d, want 1", s.FailedCount)
	}
	if s.AvgCoverage != 85.0 {
		t.Errorf("AvgCoverage = %.1f, want 85.0", s.AvgCoverage)
	}
}

func TestFormatSummary(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Module: discovery.Module{Name: "bin-api-manager"}, Passed: true, Coverage: 85.0, Duration: 2 * time.Second, Action: "test"},
		{Module: discovery.Module{Name: "bin-call-manager"}, Passed: false, Coverage: -1, Duration: 1 * time.Second, Error: "exit status 1", Action: "test"},
	}

	s := buildSummary(results, 3*time.Second)
	output := FormatSummary(s)

	if output == "" {
		t.Error("FormatSummary returned empty string")
	}

	// Should contain key elements
	if !containsAll(output, "TEST Summary", "Passed: 1", "Failed: 1", "FAIL", "bin-call-manager", "bin-api-manager", "85.0%") {
		t.Errorf("missing expected content in summary:\n%s", output)
	}
}

func TestFormatSummary_AllPassed(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Module: discovery.Module{Name: "a"}, Passed: true, Coverage: 90.0, Duration: 1 * time.Second, Action: "vet"},
		{Module: discovery.Module{Name: "b"}, Passed: true, Coverage: -1, Duration: 500 * time.Millisecond, Action: "vet"},
	}

	s := buildSummary(results, 2*time.Second)
	output := FormatSummary(s)

	if !containsAll(output, "VET Summary", "Passed: 2", "Failed: 0") {
		t.Errorf("unexpected output:\n%s", output)
	}
}

func TestBuildSummary_NoCoverage(t *testing.T) {
	t.Parallel()

	results := []Result{
		{Module: discovery.Module{Name: "a"}, Passed: true, Coverage: -1, Action: "vet"},
	}

	s := buildSummary(results, 1*time.Second)
	if s.AvgCoverage != 0 {
		t.Errorf("AvgCoverage should be 0 when no coverage data, got %.1f", s.AvgCoverage)
	}
}

func TestCountTests_Empty(t *testing.T) {
	t.Parallel()

	total, failed, skipped := countTests("")
	if total != 0 || failed != 0 || skipped != 0 {
		t.Errorf("expected all zeros for empty output, got total=%d failed=%d skipped=%d", total, failed, skipped)
	}
}

func TestCountTests_OnlySkips(t *testing.T) {
	t.Parallel()

	output := `--- SKIP: TestA (0.00s)
--- SKIP: TestB (0.00s)`

	_, _, skipped := countTests(output)
	if skipped != 2 {
		t.Errorf("skipped = %d, want 2", skipped)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
