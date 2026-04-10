package reporter

import (
	"fmt"
	"sort"
	"strings"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

// ImpactResult describes the cascade impact when a service goes down.
type ImpactResult struct {
	Service        string
	DirectCallers  []string // services that directly call this one via RPC
	DirectEvents   []string // services that subscribe to this service's events
	CascadeImpact  []string // full transitive closure of affected services
	TotalAffected  int
}

// AnalyzeImpact calculates what happens if the given service goes down.
func AnalyzeImpact(g *analyzer.Graph, serviceName string) *ImpactResult {
	result := &ImpactResult{Service: serviceName}

	// build reverse adjacency: who depends on serviceName?
	for _, dep := range g.Dependencies {
		if dep.To == serviceName {
			if dep.Type == analyzer.DepRPC {
				result.DirectCallers = appendUnique(result.DirectCallers, dep.From)
			} else {
				result.DirectEvents = appendUnique(result.DirectEvents, dep.From)
			}
		}
	}

	// BFS for transitive cascade
	reverseAdj := make(map[string][]string)
	for _, dep := range g.Dependencies {
		reverseAdj[dep.To] = appendUnique(reverseAdj[dep.To], dep.From)
	}

	visited := make(map[string]bool)
	visited[serviceName] = true
	queue := []string{serviceName}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, dependent := range reverseAdj[current] {
			if !visited[dependent] {
				visited[dependent] = true
				queue = append(queue, dependent)
				result.CascadeImpact = appendUnique(result.CascadeImpact, dependent)
			}
		}
	}

	sort.Strings(result.DirectCallers)
	sort.Strings(result.DirectEvents)
	sort.Strings(result.CascadeImpact)
	result.TotalAffected = len(result.CascadeImpact)
	return result
}

// FormatImpact returns a human-readable impact report.
func FormatImpact(r *ImpactResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Impact Analysis: %s\n", r.Service))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	sb.WriteString(fmt.Sprintf("Total affected services: %d\n\n", r.TotalAffected))

	if len(r.DirectCallers) > 0 {
		sb.WriteString("Direct RPC callers (will fail immediately):\n")
		for _, c := range r.DirectCallers {
			sb.WriteString(fmt.Sprintf("  - %s\n", c))
		}
		sb.WriteString("\n")
	}

	if len(r.DirectEvents) > 0 {
		sb.WriteString("Event subscribers (will miss events):\n")
		for _, e := range r.DirectEvents {
			sb.WriteString(fmt.Sprintf("  - %s\n", e))
		}
		sb.WriteString("\n")
	}

	if len(r.CascadeImpact) > 0 {
		sb.WriteString("Full cascade (transitive impact):\n")
		for _, c := range r.CascadeImpact {
			sb.WriteString(fmt.Sprintf("  - %s\n", c))
		}
	}

	return sb.String()
}

// FormatMetrics returns a formatted table of service metrics.
func FormatMetrics(metrics []analyzer.ServiceMetrics) string {
	var sb strings.Builder
	sb.WriteString("Service Dependency Metrics\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	sb.WriteString(fmt.Sprintf("%-25s %8s %8s %8s %8s\n",
		"Service", "RPC-Out", "RPC-In", "Evt-Pub", "Evt-Sub"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, m := range metrics {
		if m.RPCFanIn == 0 && m.RPCFanOut == 0 && m.EventPublishers == 0 && m.EventConsumers == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("%-25s %8d %8d %8d %8d\n",
			m.Name, m.RPCFanOut, m.RPCFanIn, m.EventPublishers, m.EventConsumers))
	}

	sb.WriteString("\n")
	sb.WriteString("RPC-Out: services this one calls | RPC-In: services calling this one\n")
	sb.WriteString("Evt-Pub: event types published    | Evt-Sub: event types subscribed\n")

	return sb.String()
}

func appendUnique(slice []string, val string) []string {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}
