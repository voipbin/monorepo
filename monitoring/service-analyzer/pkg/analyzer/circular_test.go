package analyzer

import (
	"testing"
)

func TestDetectCircularDeps_NoCycle(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		Dependencies: []Dependency{
			{From: "a", To: "b", Type: DepRPC},
			{From: "b", To: "c", Type: DepRPC},
		},
	}

	cycles := DetectCircularDeps(g)
	if len(cycles) != 0 {
		t.Errorf("expected 0 cycles, got %d: %+v", len(cycles), cycles)
	}
}

func TestDetectCircularDeps_SimpleCycle(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}},
		Dependencies: []Dependency{
			{From: "a", To: "b", Type: DepRPC},
			{From: "b", To: "a", Type: DepRPC},
		},
	}

	cycles := DetectCircularDeps(g)
	if len(cycles) == 0 {
		t.Fatal("expected at least 1 cycle")
	}

	found := false
	for _, c := range cycles {
		if len(c.Services) == 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a cycle of length 2, got: %+v", cycles)
	}
}

func TestDetectCircularDeps_IgnoresEventDeps(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}},
		Dependencies: []Dependency{
			{From: "a", To: "b", Type: DepRPC},
			{From: "b", To: "a", Type: DepEvent}, // event deps should be ignored
		},
	}

	cycles := DetectCircularDeps(g)
	if len(cycles) != 0 {
		t.Errorf("expected 0 cycles (event deps ignored), got %d", len(cycles))
	}
}

func TestDetectHotspots(t *testing.T) {
	g := &Graph{
		Services: []Service{
			{Name: "hub"},
			{Name: "leaf1"},
			{Name: "leaf2"},
			{Name: "isolated"},
		},
		Dependencies: []Dependency{
			{From: "leaf1", To: "hub", Type: DepRPC},
			{From: "leaf2", To: "hub", Type: DepRPC},
			{From: "hub", To: "leaf1", Type: DepRPC},
			{From: "hub", To: "leaf2", Type: DepRPC},
			{From: "leaf1", To: "hub", Type: DepEvent},
			{From: "leaf2", To: "hub", Type: DepEvent},
		},
	}

	hotspots := DetectHotspots(g)

	if len(hotspots) == 0 {
		t.Fatal("expected hotspots")
	}

	// hub should be the highest coupling
	if hotspots[0].Name != "hub" {
		t.Errorf("highest coupling should be 'hub', got %q", hotspots[0].Name)
	}
	if hotspots[0].Coupling != 6 {
		t.Errorf("hub coupling = %d, want 6", hotspots[0].Coupling)
	}
}

func TestNormalizeCycleKey(t *testing.T) {
	// same cycle from different starting points should produce same key
	key1 := normalizeCycleKey([]string{"a", "b", "c"})
	key2 := normalizeCycleKey([]string{"b", "c", "a"})
	key3 := normalizeCycleKey([]string{"c", "a", "b"})

	if key1 != key2 || key2 != key3 {
		t.Errorf("same cycle should produce same key: %q, %q, %q", key1, key2, key3)
	}
}
