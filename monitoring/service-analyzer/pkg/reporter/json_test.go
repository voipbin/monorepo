package reporter

import (
	"encoding/json"
	"testing"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

func TestGenerateJSON(t *testing.T) {
	g := &analyzer.Graph{
		Services: []analyzer.Service{
			{Name: "svc-a"},
			{Name: "svc-b"},
		},
		Dependencies: []analyzer.Dependency{
			{From: "svc-a", To: "svc-b", Type: analyzer.DepRPC, Methods: []string{"M1"}},
			{From: "svc-a", To: "svc-b", Type: analyzer.DepEvent, Methods: []string{"evt_1"}},
		},
	}

	data, err := GenerateJSON(g)
	if err != nil {
		t.Fatal(err)
	}

	var jg JSONGraph
	if err := json.Unmarshal(data, &jg); err != nil {
		t.Fatal(err)
	}

	if jg.TotalServices != 2 {
		t.Errorf("TotalServices = %d, want 2", jg.TotalServices)
	}
	if jg.TotalDependencies != 2 {
		t.Errorf("TotalDependencies = %d, want 2", jg.TotalDependencies)
	}
	if jg.RPCCount != 1 {
		t.Errorf("RPCCount = %d, want 1", jg.RPCCount)
	}
	if jg.EventCount != 1 {
		t.Errorf("EventCount = %d, want 1", jg.EventCount)
	}
}

func TestGenerateImpactJSON(t *testing.T) {
	result := &ImpactResult{
		Service:       "test-svc",
		TotalAffected: 2,
		DirectCallers: []string{"caller"},
		DirectEvents:  []string{"subscriber"},
		CascadeImpact: []string{"caller", "subscriber"},
	}

	data, err := GenerateImpactJSON(result)
	if err != nil {
		t.Fatal(err)
	}

	var ji JSONImpact
	if err := json.Unmarshal(data, &ji); err != nil {
		t.Fatal(err)
	}

	if ji.Service != "test-svc" {
		t.Errorf("Service = %q, want 'test-svc'", ji.Service)
	}
	if ji.TotalAffected != 2 {
		t.Errorf("TotalAffected = %d, want 2", ji.TotalAffected)
	}
}
