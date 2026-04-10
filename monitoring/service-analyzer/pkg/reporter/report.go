package reporter

import (
	"fmt"
	"strings"
	"time"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

// GenerateFullReport produces a complete architectural health report.
func GenerateFullReport(g *analyzer.Graph) string {
	var sb strings.Builder

	sb.WriteString("VoIPbin Monorepo - Architectural Health Report\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(strings.Repeat("=", 70) + "\n\n")

	// 1. Overview
	rpcCount := 0
	eventCount := 0
	for _, d := range g.Dependencies {
		if d.Type == analyzer.DepRPC {
			rpcCount++
		} else {
			eventCount++
		}
	}

	sb.WriteString("1. OVERVIEW\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("  Services:          %d\n", len(g.Services)))
	sb.WriteString(fmt.Sprintf("  Total deps:        %d\n", len(g.Dependencies)))
	sb.WriteString(fmt.Sprintf("  RPC deps:          %d\n", rpcCount))
	sb.WriteString(fmt.Sprintf("  Event deps:        %d\n", eventCount))
	sb.WriteString("\n")

	// 2. Hotspots
	hotspots := analyzer.DetectHotspots(g)
	sb.WriteString("2. ARCHITECTURAL HOTSPOTS\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")

	critCount := 0
	highCount := 0
	for _, h := range hotspots {
		if h.RiskLevel == "critical" {
			critCount++
		} else if h.RiskLevel == "high" {
			highCount++
		}
	}

	sb.WriteString(fmt.Sprintf("  CRITICAL risk:     %d services\n", critCount))
	sb.WriteString(fmt.Sprintf("  HIGH risk:         %d services\n", highCount))
	sb.WriteString("\n")

	if critCount > 0 {
		sb.WriteString("  Critical services (coupling >= 20):\n")
		for _, h := range hotspots {
			if h.RiskLevel == "critical" {
				sb.WriteString(fmt.Sprintf("    %-22s  in:%d  out:%d  total:%d\n",
					h.Name, h.FanIn, h.FanOut, h.Coupling))
			}
		}
		sb.WriteString("\n")
	}

	// 3. Circular Dependencies
	cycles := analyzer.DetectCircularDeps(g)
	sb.WriteString("3. CIRCULAR DEPENDENCIES\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("  Circular RPC chains:  %d\n\n", len(cycles)))

	// show only short cycles (length 2 = direct mutual dependency)
	directCycles := 0
	for _, c := range cycles {
		if len(c.Services) == 2 {
			directCycles++
			sb.WriteString(fmt.Sprintf("    %s <-> %s\n", c.Services[0], c.Services[1]))
		}
	}
	if directCycles > 0 {
		sb.WriteString(fmt.Sprintf("\n  Direct mutual dependencies: %d\n", directCycles))
	}
	sb.WriteString("\n")

	// 4. Top Impact Services
	sb.WriteString("4. CASCADE IMPACT (Top 5)\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")

	type impactEntry struct {
		name    string
		total   int
	}
	var impacts []impactEntry
	for _, svc := range g.Services {
		result := AnalyzeImpact(g, svc.Name)
		if result.TotalAffected > 0 {
			impacts = append(impacts, impactEntry{svc.Name, result.TotalAffected})
		}
	}

	// sort by impact descending
	for i := 0; i < len(impacts); i++ {
		for j := i + 1; j < len(impacts); j++ {
			if impacts[j].total > impacts[i].total {
				impacts[i], impacts[j] = impacts[j], impacts[i]
			}
		}
	}

	limit := 5
	if len(impacts) < limit {
		limit = len(impacts)
	}
	for i := 0; i < limit; i++ {
		sb.WriteString(fmt.Sprintf("  %-25s  affects %d/%d services\n",
			impacts[i].name, impacts[i].total, len(g.Services)))
	}
	sb.WriteString("\n")

	// 5. Isolated Services
	sb.WriteString("5. ISOLATED SERVICES (no dependencies)\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")

	metrics := analyzer.ComputeMetrics(g)
	isolated := 0
	for _, m := range metrics {
		if m.RPCFanIn == 0 && m.RPCFanOut == 0 && m.EventPublishers == 0 && m.EventConsumers == 0 {
			sb.WriteString(fmt.Sprintf("  - %s\n", m.Name))
			isolated++
		}
	}
	if isolated == 0 {
		sb.WriteString("  (none)\n")
	}
	sb.WriteString("\n")

	// 6. Health Score
	sb.WriteString("6. HEALTH SCORE\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")

	score := computeHealthScore(len(g.Services), critCount, highCount, len(cycles), directCycles)
	sb.WriteString(fmt.Sprintf("  Score: %d/100\n\n", score))

	if critCount > 0 {
		sb.WriteString("  Recommendations:\n")
		sb.WriteString("  - Review CRITICAL coupling services for potential decomposition\n")
	}
	if directCycles > 0 {
		sb.WriteString("  - Break direct mutual RPC cycles with async events or shared services\n")
	}
	if score >= 80 {
		sb.WriteString("  Architecture is in good shape.\n")
	} else if score >= 60 {
		sb.WriteString("  Architecture has moderate coupling concerns.\n")
	} else {
		sb.WriteString("  Architecture has significant coupling issues to address.\n")
	}

	return sb.String()
}

// ComputeHealthScorePublic exposes the health score computation for use by other packages.
func ComputeHealthScorePublic(totalServices, criticalCount, highCount, totalCycles, directCycles int) int {
	return computeHealthScore(totalServices, criticalCount, highCount, totalCycles, directCycles)
}

// computeHealthScore produces a 0-100 score based on architectural metrics.
func computeHealthScore(totalServices, criticalCount, highCount, totalCycles, directCycles int) int {
	score := 100

	// penalty for critical hotspots: -10 each
	score -= criticalCount * 10

	// penalty for high hotspots: -3 each
	score -= highCount * 3

	// penalty for direct mutual cycles: -5 each
	score -= directCycles * 5

	// penalty for long cycles: -1 each (beyond direct ones)
	longCycles := totalCycles - directCycles
	if longCycles > 0 {
		score -= longCycles
	}

	if score < 0 {
		score = 0
	}
	return score
}
