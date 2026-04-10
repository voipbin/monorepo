package reporter

import (
	"strings"
	"testing"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

func TestGenerateFullReport(t *testing.T) {
	g := &analyzer.Graph{
		Services: []analyzer.Service{
			{Name: "svc-a"},
			{Name: "svc-b"},
			{Name: "svc-c"},
		},
		Dependencies: []analyzer.Dependency{
			{From: "svc-a", To: "svc-b", Type: analyzer.DepRPC, Methods: []string{"M1"}},
			{From: "svc-b", To: "svc-a", Type: analyzer.DepRPC, Methods: []string{"M2"}},
			{From: "svc-c", To: "svc-a", Type: analyzer.DepEvent, Methods: []string{"e1"}},
		},
	}

	report := GenerateFullReport(g)

	if !strings.Contains(report, "Architectural Health Report") {
		t.Error("report should contain title")
	}
	if !strings.Contains(report, "Services:          3") {
		t.Error("report should show service count")
	}
	if !strings.Contains(report, "CIRCULAR DEPENDENCIES") {
		t.Error("report should include circular dependencies section")
	}
	if !strings.Contains(report, "CASCADE IMPACT") {
		t.Error("report should include cascade impact section")
	}
	if !strings.Contains(report, "HEALTH SCORE") {
		t.Error("report should include health score")
	}
}

func TestComputeHealthScore(t *testing.T) {
	// perfect score: no issues
	score := computeHealthScore(10, 0, 0, 0, 0)
	if score != 100 {
		t.Errorf("perfect score = %d, want 100", score)
	}

	// one critical hotspot: -10
	score = computeHealthScore(10, 1, 0, 0, 0)
	if score != 90 {
		t.Errorf("one critical = %d, want 90", score)
	}

	// one direct cycle: -5
	score = computeHealthScore(10, 0, 0, 1, 1)
	if score != 95 {
		t.Errorf("one direct cycle = %d, want 95", score)
	}

	// never below 0
	score = computeHealthScore(10, 20, 20, 50, 50)
	if score != 0 {
		t.Errorf("floor score = %d, want 0", score)
	}
}
