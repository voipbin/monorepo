package reporter

import (
	"encoding/json"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

// JSONGraph is the JSON-serializable representation of the full dependency graph.
type JSONGraph struct {
	TotalServices     int              `json:"total_services"`
	TotalDependencies int              `json:"total_dependencies"`
	RPCCount          int              `json:"rpc_count"`
	EventCount        int              `json:"event_count"`
	Services          []JSONService    `json:"services"`
	Dependencies      []JSONDependency `json:"dependencies"`
}

// JSONService is a JSON-serializable service entry.
type JSONService struct {
	Name string `json:"name"`
}

// JSONDependency is a JSON-serializable dependency edge.
type JSONDependency struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Type    string   `json:"type"`
	Methods []string `json:"methods"`
}

// GenerateJSON produces a JSON representation of the dependency graph.
func GenerateJSON(g *analyzer.Graph) ([]byte, error) {
	jg := JSONGraph{
		TotalServices:     len(g.Services),
		TotalDependencies: len(g.Dependencies),
	}

	for _, svc := range g.Services {
		jg.Services = append(jg.Services, JSONService{Name: svc.Name})
	}

	for _, dep := range g.Dependencies {
		jd := JSONDependency{
			From:    dep.From,
			To:      dep.To,
			Type:    string(dep.Type),
			Methods: dep.Methods,
		}
		jg.Dependencies = append(jg.Dependencies, jd)
		if dep.Type == analyzer.DepRPC {
			jg.RPCCount++
		} else {
			jg.EventCount++
		}
	}

	return json.MarshalIndent(jg, "", "  ")
}

// JSONImpact is the JSON-serializable representation of an impact result.
type JSONImpact struct {
	Service       string   `json:"service"`
	TotalAffected int      `json:"total_affected"`
	DirectCallers []string `json:"direct_callers"`
	DirectEvents  []string `json:"direct_event_subscribers"`
	CascadeImpact []string `json:"cascade_impact"`
}

// GenerateImpactJSON produces a JSON representation of an impact analysis.
func GenerateImpactJSON(r *ImpactResult) ([]byte, error) {
	ji := JSONImpact{
		Service:       r.Service,
		TotalAffected: r.TotalAffected,
		DirectCallers: r.DirectCallers,
		DirectEvents:  r.DirectEvents,
		CascadeImpact: r.CascadeImpact,
	}
	return json.MarshalIndent(ji, "", "  ")
}
