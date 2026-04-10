package reporter

import (
	"strings"
	"testing"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

func TestAnalyzeImpact(t *testing.T) {
	g := &analyzer.Graph{
		Services: []analyzer.Service{
			{Name: "svc-a"},
			{Name: "svc-b"},
			{Name: "svc-c"},
			{Name: "svc-d"},
		},
		Dependencies: []analyzer.Dependency{
			{From: "svc-b", To: "svc-a", Type: analyzer.DepRPC, Methods: []string{"M1"}},
			{From: "svc-c", To: "svc-a", Type: analyzer.DepEvent, Methods: []string{"evt_1"}},
			{From: "svc-d", To: "svc-b", Type: analyzer.DepRPC, Methods: []string{"M2"}},
		},
	}

	result := AnalyzeImpact(g, "svc-a")

	if result.Service != "svc-a" {
		t.Errorf("Service = %q, want 'svc-a'", result.Service)
	}

	if len(result.DirectCallers) != 1 || result.DirectCallers[0] != "svc-b" {
		t.Errorf("DirectCallers = %v, want [svc-b]", result.DirectCallers)
	}

	if len(result.DirectEvents) != 1 || result.DirectEvents[0] != "svc-c" {
		t.Errorf("DirectEvents = %v, want [svc-c]", result.DirectEvents)
	}

	// cascade: svc-b depends on svc-a, svc-d depends on svc-b → svc-d is transitively affected
	if result.TotalAffected != 3 {
		t.Errorf("TotalAffected = %d, want 3 (svc-b, svc-c, svc-d)", result.TotalAffected)
	}
}

func TestAnalyzeImpactNoDependent(t *testing.T) {
	g := &analyzer.Graph{
		Services: []analyzer.Service{
			{Name: "leaf"},
			{Name: "core"},
		},
		Dependencies: []analyzer.Dependency{
			{From: "leaf", To: "core", Type: analyzer.DepRPC, Methods: []string{"M1"}},
		},
	}

	result := AnalyzeImpact(g, "leaf")
	if result.TotalAffected != 0 {
		t.Errorf("TotalAffected = %d, want 0 (leaf has no dependents)", result.TotalAffected)
	}
}

func TestFormatImpact(t *testing.T) {
	result := &ImpactResult{
		Service:       "test-svc",
		DirectCallers: []string{"caller-1", "caller-2"},
		DirectEvents:  []string{"subscriber-1"},
		CascadeImpact: []string{"caller-1", "caller-2", "subscriber-1"},
		TotalAffected: 3,
	}

	output := FormatImpact(result)
	if !strings.Contains(output, "test-svc") {
		t.Error("output should contain service name")
	}
	if !strings.Contains(output, "caller-1") {
		t.Error("output should contain direct callers")
	}
	if !strings.Contains(output, "subscriber-1") {
		t.Error("output should contain event subscribers")
	}
	if !strings.Contains(output, "Total affected services: 3") {
		t.Error("output should contain total count")
	}
}

func TestFormatMetrics(t *testing.T) {
	metrics := []analyzer.ServiceMetrics{
		{Name: "svc-a", RPCFanOut: 5, RPCFanIn: 3, EventPublishers: 2, EventConsumers: 1},
		{Name: "svc-b", RPCFanOut: 0, RPCFanIn: 0, EventPublishers: 0, EventConsumers: 0},
	}

	output := FormatMetrics(metrics)
	if !strings.Contains(output, "svc-a") {
		t.Error("output should contain svc-a")
	}
	// svc-b has all zeros, should be filtered out
	if strings.Contains(output, "svc-b") {
		t.Error("svc-b with all zeros should be filtered out")
	}
}
