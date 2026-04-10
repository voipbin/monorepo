package reporter

import (
	"encoding/json"

	"monorepo/monitoring/mono-ci/pkg/runner"
)

// JSONReport is the CI-friendly output format.
type JSONReport struct {
	Action        string         `json:"action"`
	TotalModules  int            `json:"total_modules"`
	Passed        int            `json:"passed"`
	Failed        int            `json:"failed"`
	AvgCoverage   float64        `json:"avg_coverage,omitempty"`
	DurationMs    int64          `json:"duration_ms"`
	Modules       []ModuleReport `json:"modules"`
}

// ModuleReport is the per-module result in JSON format.
type ModuleReport struct {
	Name      string  `json:"name"`
	Passed    bool    `json:"passed"`
	Coverage  float64 `json:"coverage,omitempty"`
	Duration  string  `json:"duration"`
	Error     string  `json:"error,omitempty"`
	TestCount int     `json:"test_count,omitempty"`
	FailCount int     `json:"fail_count,omitempty"`
}

// GenerateJSON converts a runner.Summary into a JSON-serialized report.
func GenerateJSON(s runner.Summary) ([]byte, error) {
	report := JSONReport{
		TotalModules: len(s.Results),
		Passed:       s.PassedCount,
		Failed:       s.FailedCount,
		AvgCoverage:  s.AvgCoverage,
		DurationMs:   s.TotalDuration.Milliseconds(),
	}

	if len(s.Results) > 0 {
		report.Action = s.Results[0].Action
	}

	for _, r := range s.Results {
		mr := ModuleReport{
			Name:      r.Module.Name,
			Passed:    r.Passed,
			Duration:  r.Duration.String(),
			Error:     r.Error,
			TestCount: r.TestCount,
			FailCount: r.FailCount,
		}
		if r.Coverage >= 0 {
			mr.Coverage = r.Coverage
		}
		report.Modules = append(report.Modules, mr)
	}

	return json.MarshalIndent(report, "", "  ")
}
