package analyzer

import (
	"testing"
)

func TestFindShortestPath_Direct(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}},
		Dependencies: []Dependency{
			{From: "a", To: "b", Type: DepRPC},
		},
	}

	p := FindShortestPath(g, "a", "b")
	if p == nil {
		t.Fatal("expected path, got nil")
	}
	if len(p.Services) != 2 {
		t.Errorf("path length = %d, want 2", len(p.Services))
	}
	if p.Services[0] != "a" || p.Services[1] != "b" {
		t.Errorf("path = %v, want [a, b]", p.Services)
	}
	if len(p.Types) != 1 || p.Types[0] != DepRPC {
		t.Errorf("types = %v, want [rpc]", p.Types)
	}
}

func TestFindShortestPath_Transitive(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		Dependencies: []Dependency{
			{From: "a", To: "b", Type: DepRPC},
			{From: "b", To: "c", Type: DepEvent},
		},
	}

	p := FindShortestPath(g, "a", "c")
	if p == nil {
		t.Fatal("expected path, got nil")
	}
	if len(p.Services) != 3 {
		t.Errorf("path length = %d, want 3", len(p.Services))
	}
	if p.Types[0] != DepRPC || p.Types[1] != DepEvent {
		t.Errorf("types = %v, want [rpc, event]", p.Types)
	}
}

func TestFindShortestPath_NoPath(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		Dependencies: []Dependency{
			{From: "a", To: "b", Type: DepRPC},
			// no path from a to c
		},
	}

	p := FindShortestPath(g, "a", "c")
	if p != nil {
		t.Errorf("expected nil path, got %v", p.Services)
	}
}

func TestFindShortestPath_ChoosesShorter(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
		Dependencies: []Dependency{
			{From: "a", To: "d", Type: DepRPC},   // direct: a->d
			{From: "a", To: "b", Type: DepRPC},   // long: a->b->c->d
			{From: "b", To: "c", Type: DepRPC},
			{From: "c", To: "d", Type: DepRPC},
		},
	}

	p := FindShortestPath(g, "a", "d")
	if p == nil {
		t.Fatal("expected path, got nil")
	}
	if len(p.Services) != 2 {
		t.Errorf("shortest path length = %d, want 2 (direct a->d)", len(p.Services))
	}
}

func TestFindAllPaths_MultiplePaths(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
		Dependencies: []Dependency{
			{From: "a", To: "d", Type: DepRPC},   // path 1: a->d
			{From: "a", To: "b", Type: DepRPC},   // path 2: a->b->d
			{From: "b", To: "d", Type: DepRPC},
			{From: "a", To: "c", Type: DepEvent},  // path 3: a->c->d
			{From: "c", To: "d", Type: DepRPC},
		},
	}

	paths := FindAllPaths(g, "a", "d", 5)
	if len(paths) != 3 {
		t.Errorf("expected 3 paths, got %d", len(paths))
		for i, p := range paths {
			t.Logf("path %d: %v", i, p.Services)
		}
	}

	// shortest first
	if len(paths) > 0 && len(paths[0].Services) != 2 {
		t.Errorf("first path should be shortest (2 hops), got %d", len(paths[0].Services))
	}
}

func TestFindAllPaths_NoPath(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}},
		Dependencies: []Dependency{
			{From: "b", To: "a", Type: DepRPC}, // reverse direction only
		},
	}

	paths := FindAllPaths(g, "a", "b", 5)
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

func TestFindAllPaths_MaxDepth(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}, {Name: "e"}},
		Dependencies: []Dependency{
			{From: "a", To: "b", Type: DepRPC},
			{From: "b", To: "c", Type: DepRPC},
			{From: "c", To: "d", Type: DepRPC},
			{From: "d", To: "e", Type: DepRPC},
		},
	}

	// maxDepth=2 should not find a->b->c->d->e (depth 4)
	paths := FindAllPaths(g, "a", "e", 2)
	if len(paths) != 0 {
		t.Errorf("expected 0 paths with maxDepth=2, got %d", len(paths))
	}

	// maxDepth=5 should find it
	paths = FindAllPaths(g, "a", "e", 5)
	if len(paths) != 1 {
		t.Errorf("expected 1 path with maxDepth=5, got %d", len(paths))
	}
}

func TestFindAllPaths_NoCycle(t *testing.T) {
	// ensure cycles don't cause infinite loops
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		Dependencies: []Dependency{
			{From: "a", To: "b", Type: DepRPC},
			{From: "b", To: "a", Type: DepRPC}, // cycle
			{From: "b", To: "c", Type: DepRPC},
		},
	}

	paths := FindAllPaths(g, "a", "c", 10)
	if len(paths) != 1 {
		t.Errorf("expected 1 path (a->b->c), got %d", len(paths))
	}
}

func TestFindAllPaths_DefaultMaxDepth(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}, {Name: "b"}},
		Dependencies: []Dependency{
			{From: "a", To: "b", Type: DepRPC},
		},
	}

	// maxDepth=0 should use default (10)
	paths := FindAllPaths(g, "a", "b", 0)
	if len(paths) != 1 {
		t.Errorf("expected 1 path with default maxDepth, got %d", len(paths))
	}
}

func TestFindShortestPath_SameNode(t *testing.T) {
	g := &Graph{
		Services: []Service{{Name: "a"}},
		Dependencies: []Dependency{},
	}

	// source == target, no self-loop
	p := FindShortestPath(g, "a", "a")
	if p != nil {
		t.Errorf("expected nil for same node with no self-loop, got %v", p.Services)
	}
}
