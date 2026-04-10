package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

// Snapshot represents a point-in-time capture of the dependency graph.
type Snapshot struct {
	Timestamp    string           `json:"timestamp"`
	Services     int              `json:"services"`
	Dependencies int              `json:"dependencies"`
	RPCCount     int              `json:"rpc_count"`
	EventCount   int              `json:"event_count"`
	HealthScore  int              `json:"health_score"`
	Edges        []SnapshotEdge   `json:"edges"`
	Hotspots     []SnapshotHotspot `json:"hotspots,omitempty"`
	Cycles       int              `json:"cycles"`
}

// SnapshotEdge is a single dependency edge in the snapshot.
type SnapshotEdge struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Type    string   `json:"type"`
	Methods []string `json:"methods"`
}

// SnapshotHotspot records a high-coupling service at snapshot time.
type SnapshotHotspot struct {
	Name      string `json:"name"`
	Coupling  int    `json:"coupling"`
	RiskLevel string `json:"risk_level"`
}

// CreateSnapshot captures the current graph state as a serializable snapshot.
func CreateSnapshot(g *analyzer.Graph) *Snapshot {
	rpcCount := 0
	eventCount := 0
	for _, d := range g.Dependencies {
		if d.Type == analyzer.DepRPC {
			rpcCount++
		} else {
			eventCount++
		}
	}

	hotspots := analyzer.DetectHotspots(g)
	cycles := analyzer.DetectCircularDeps(g)

	critCount := 0
	highCount := 0
	directCycles := 0
	for _, h := range hotspots {
		if h.RiskLevel == "critical" {
			critCount++
		} else if h.RiskLevel == "high" {
			highCount++
		}
	}
	for _, c := range cycles {
		if len(c.Services) == 2 {
			directCycles++
		}
	}

	score := computeHealthScore(len(g.Services), critCount, highCount, len(cycles), directCycles)

	var edges []SnapshotEdge
	for _, d := range g.Dependencies {
		edges = append(edges, SnapshotEdge{
			From:    d.From,
			To:      d.To,
			Type:    string(d.Type),
			Methods: d.Methods,
		})
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})

	var snapshotHotspots []SnapshotHotspot
	for _, h := range hotspots {
		if h.RiskLevel == "critical" || h.RiskLevel == "high" {
			snapshotHotspots = append(snapshotHotspots, SnapshotHotspot{
				Name:      h.Name,
				Coupling:  h.Coupling,
				RiskLevel: h.RiskLevel,
			})
		}
	}

	return &Snapshot{
		Timestamp:    time.Now().Format(time.RFC3339),
		Services:     len(g.Services),
		Dependencies: len(g.Dependencies),
		RPCCount:     rpcCount,
		EventCount:   eventCount,
		HealthScore:  score,
		Edges:        edges,
		Hotspots:     snapshotHotspots,
		Cycles:       len(cycles),
	}
}

// SaveSnapshot writes a snapshot to a JSON file.
func SaveSnapshot(snap *Snapshot, path string) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadSnapshot reads a snapshot from a JSON file.
func LoadSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parse snapshot: %w", err)
	}
	return &snap, nil
}

// SnapshotDiff describes the changes between two snapshots.
type SnapshotDiff struct {
	OldTimestamp string
	NewTimestamp string
	AddedEdges   []SnapshotEdge
	RemovedEdges []SnapshotEdge
	ScoreDelta   int // positive = improved, negative = regressed
	ServiceDelta int
	DepDelta     int
	CycleDelta   int
}

// DiffSnapshots compares two snapshots and returns what changed.
func DiffSnapshots(old, new *Snapshot) *SnapshotDiff {
	diff := &SnapshotDiff{
		OldTimestamp: old.Timestamp,
		NewTimestamp: new.Timestamp,
		ScoreDelta:   new.HealthScore - old.HealthScore,
		ServiceDelta: new.Services - old.Services,
		DepDelta:     new.Dependencies - old.Dependencies,
		CycleDelta:   new.Cycles - old.Cycles,
	}

	oldEdges := make(map[string]SnapshotEdge)
	for _, e := range old.Edges {
		key := e.From + "|" + e.To + "|" + e.Type
		oldEdges[key] = e
	}

	newEdges := make(map[string]SnapshotEdge)
	for _, e := range new.Edges {
		key := e.From + "|" + e.To + "|" + e.Type
		newEdges[key] = e
	}

	for key, e := range newEdges {
		if _, exists := oldEdges[key]; !exists {
			diff.AddedEdges = append(diff.AddedEdges, e)
		}
	}

	for key, e := range oldEdges {
		if _, exists := newEdges[key]; !exists {
			diff.RemovedEdges = append(diff.RemovedEdges, e)
		}
	}

	sort.Slice(diff.AddedEdges, func(i, j int) bool {
		return diff.AddedEdges[i].From < diff.AddedEdges[j].From
	})
	sort.Slice(diff.RemovedEdges, func(i, j int) bool {
		return diff.RemovedEdges[i].From < diff.RemovedEdges[j].From
	})

	return diff
}

// FormatDiff returns a human-readable diff report.
func FormatDiff(d *SnapshotDiff) string {
	var sb strings.Builder

	sb.WriteString("Dependency Diff Report\n")
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")
	sb.WriteString(fmt.Sprintf("  Old: %s\n", d.OldTimestamp))
	sb.WriteString(fmt.Sprintf("  New: %s\n\n", d.NewTimestamp))

	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  Services:     %+d\n", d.ServiceDelta))
	sb.WriteString(fmt.Sprintf("  Dependencies: %+d\n", d.DepDelta))
	sb.WriteString(fmt.Sprintf("  Cycles:       %+d\n", d.CycleDelta))
	sb.WriteString(fmt.Sprintf("  Health Score: %+d\n\n", d.ScoreDelta))

	if len(d.AddedEdges) > 0 {
		sb.WriteString(fmt.Sprintf("Added Dependencies (%d):\n", len(d.AddedEdges)))
		for _, e := range d.AddedEdges {
			sb.WriteString(fmt.Sprintf("  + [%s] %s -> %s\n", strings.ToUpper(e.Type), e.From, e.To))
		}
		sb.WriteString("\n")
	}

	if len(d.RemovedEdges) > 0 {
		sb.WriteString(fmt.Sprintf("Removed Dependencies (%d):\n", len(d.RemovedEdges)))
		for _, e := range d.RemovedEdges {
			sb.WriteString(fmt.Sprintf("  - [%s] %s -> %s\n", strings.ToUpper(e.Type), e.From, e.To))
		}
		sb.WriteString("\n")
	}

	if len(d.AddedEdges) == 0 && len(d.RemovedEdges) == 0 {
		sb.WriteString("No dependency changes detected.\n")
	}

	if d.ScoreDelta < 0 {
		sb.WriteString("WARNING: Health score regressed. Review new dependencies.\n")
	} else if d.ScoreDelta > 0 {
		sb.WriteString("Health score improved.\n")
	}

	return sb.String()
}
