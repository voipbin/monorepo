package reporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

func TestCreateSnapshot(t *testing.T) {
	g := &analyzer.Graph{
		Services: []analyzer.Service{
			{Name: "svc-a"},
			{Name: "svc-b"},
			{Name: "svc-c"},
		},
		Dependencies: []analyzer.Dependency{
			{From: "svc-a", To: "svc-b", Type: analyzer.DepRPC, Methods: []string{"M1"}},
			{From: "svc-b", To: "svc-c", Type: analyzer.DepEvent, Methods: []string{"e1"}},
		},
	}

	snap := CreateSnapshot(g)

	if snap.Services != 3 {
		t.Errorf("services = %d, want 3", snap.Services)
	}
	if snap.Dependencies != 2 {
		t.Errorf("dependencies = %d, want 2", snap.Dependencies)
	}
	if snap.RPCCount != 1 {
		t.Errorf("rpc_count = %d, want 1", snap.RPCCount)
	}
	if snap.EventCount != 1 {
		t.Errorf("event_count = %d, want 1", snap.EventCount)
	}
	if snap.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
	if len(snap.Edges) != 2 {
		t.Errorf("edges = %d, want 2", len(snap.Edges))
	}
}

func TestSaveAndLoadSnapshot(t *testing.T) {
	g := &analyzer.Graph{
		Services: []analyzer.Service{
			{Name: "svc-a"},
			{Name: "svc-b"},
		},
		Dependencies: []analyzer.Dependency{
			{From: "svc-a", To: "svc-b", Type: analyzer.DepRPC, Methods: []string{"CallV1Get"}},
		},
	}

	snap := CreateSnapshot(g)
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")

	if err := SaveSnapshot(snap, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Services != snap.Services {
		t.Errorf("loaded services = %d, want %d", loaded.Services, snap.Services)
	}
	if loaded.Dependencies != snap.Dependencies {
		t.Errorf("loaded deps = %d, want %d", loaded.Dependencies, snap.Dependencies)
	}
	if len(loaded.Edges) != len(snap.Edges) {
		t.Errorf("loaded edges = %d, want %d", len(loaded.Edges), len(snap.Edges))
	}
}

func TestLoadSnapshot_FileNotFound(t *testing.T) {
	_, err := LoadSnapshot("/nonexistent/snapshot.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadSnapshot_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := LoadSnapshot(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDiffSnapshots_AddedEdge(t *testing.T) {
	old := &Snapshot{
		Timestamp:    "2026-01-01T00:00:00Z",
		Services:     2,
		Dependencies: 1,
		HealthScore:  100,
		Edges: []SnapshotEdge{
			{From: "a", To: "b", Type: "rpc"},
		},
	}
	new := &Snapshot{
		Timestamp:    "2026-02-01T00:00:00Z",
		Services:     3,
		Dependencies: 2,
		HealthScore:  95,
		Edges: []SnapshotEdge{
			{From: "a", To: "b", Type: "rpc"},
			{From: "a", To: "c", Type: "rpc"},
		},
	}

	diff := DiffSnapshots(old, new)

	if len(diff.AddedEdges) != 1 {
		t.Errorf("added = %d, want 1", len(diff.AddedEdges))
	}
	if len(diff.RemovedEdges) != 0 {
		t.Errorf("removed = %d, want 0", len(diff.RemovedEdges))
	}
	if diff.ServiceDelta != 1 {
		t.Errorf("service delta = %d, want 1", diff.ServiceDelta)
	}
	if diff.ScoreDelta != -5 {
		t.Errorf("score delta = %d, want -5", diff.ScoreDelta)
	}
}

func TestDiffSnapshots_RemovedEdge(t *testing.T) {
	old := &Snapshot{
		Timestamp:    "2026-01-01T00:00:00Z",
		Services:     2,
		Dependencies: 2,
		HealthScore:  80,
		Edges: []SnapshotEdge{
			{From: "a", To: "b", Type: "rpc"},
			{From: "b", To: "a", Type: "rpc"},
		},
	}
	new := &Snapshot{
		Timestamp:    "2026-02-01T00:00:00Z",
		Services:     2,
		Dependencies: 1,
		HealthScore:  100,
		Edges: []SnapshotEdge{
			{From: "a", To: "b", Type: "rpc"},
		},
	}

	diff := DiffSnapshots(old, new)

	if len(diff.AddedEdges) != 0 {
		t.Errorf("added = %d, want 0", len(diff.AddedEdges))
	}
	if len(diff.RemovedEdges) != 1 {
		t.Errorf("removed = %d, want 1", len(diff.RemovedEdges))
	}
	if diff.ScoreDelta != 20 {
		t.Errorf("score delta = %d, want 20", diff.ScoreDelta)
	}
}

func TestDiffSnapshots_NoChanges(t *testing.T) {
	snap := &Snapshot{
		Timestamp:    "2026-01-01T00:00:00Z",
		Services:     2,
		Dependencies: 1,
		HealthScore:  100,
		Edges: []SnapshotEdge{
			{From: "a", To: "b", Type: "rpc"},
		},
	}

	diff := DiffSnapshots(snap, snap)

	if len(diff.AddedEdges) != 0 || len(diff.RemovedEdges) != 0 {
		t.Error("expected no changes for identical snapshots")
	}
	if diff.ScoreDelta != 0 {
		t.Errorf("score delta = %d, want 0", diff.ScoreDelta)
	}
}

func TestFormatDiff_Added(t *testing.T) {
	diff := &SnapshotDiff{
		OldTimestamp: "2026-01-01",
		NewTimestamp: "2026-02-01",
		AddedEdges: []SnapshotEdge{
			{From: "a", To: "c", Type: "rpc"},
		},
		ScoreDelta:   -5,
		ServiceDelta: 1,
		DepDelta:     1,
	}

	output := FormatDiff(diff)
	if !strings.Contains(output, "+ [RPC] a -> c") {
		t.Error("output should show added edge")
	}
	if !strings.Contains(output, "WARNING") {
		t.Error("output should warn about regression")
	}
}

func TestFormatDiff_Removed(t *testing.T) {
	diff := &SnapshotDiff{
		OldTimestamp: "2026-01-01",
		NewTimestamp: "2026-02-01",
		RemovedEdges: []SnapshotEdge{
			{From: "b", To: "a", Type: "rpc"},
		},
		ScoreDelta: 10,
		DepDelta:   -1,
	}

	output := FormatDiff(diff)
	if !strings.Contains(output, "- [RPC] b -> a") {
		t.Error("output should show removed edge")
	}
	if !strings.Contains(output, "improved") {
		t.Error("output should note improvement")
	}
}

func TestFormatDiff_NoChanges(t *testing.T) {
	diff := &SnapshotDiff{
		OldTimestamp: "2026-01-01",
		NewTimestamp: "2026-02-01",
	}

	output := FormatDiff(diff)
	if !strings.Contains(output, "No dependency changes") {
		t.Error("output should indicate no changes")
	}
}
