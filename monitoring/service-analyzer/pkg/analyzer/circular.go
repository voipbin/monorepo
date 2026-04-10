package analyzer

import "sort"

// Cycle represents a circular dependency chain.
type Cycle struct {
	Services []string // ordered list forming the cycle, last→first completes the loop
}

// DetectCircularDeps finds all circular dependency cycles in the graph.
// Only considers RPC dependencies (event deps are loosely coupled by design).
func DetectCircularDeps(g *Graph) []Cycle {
	adj := make(map[string][]string)
	for _, dep := range g.Dependencies {
		if dep.Type == DepRPC {
			adj[dep.From] = appendUniq(adj[dep.From], dep.To)
		}
	}

	var cycles []Cycle
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		visited[node] = true
		stack[node] = true
		path = append(path, node)

		for _, neighbor := range adj[node] {
			if stack[neighbor] {
				// found cycle - extract it
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cyclePath := make([]string, len(path)-cycleStart)
					copy(cyclePath, path[cycleStart:])
					cycles = append(cycles, Cycle{Services: cyclePath})
				}
			} else if !visited[neighbor] {
				dfs(neighbor, path)
			}
		}

		stack[node] = false
	}

	serviceNames := make([]string, 0, len(adj))
	for name := range adj {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	for _, name := range serviceNames {
		if !visited[name] {
			dfs(name, nil)
		}
	}

	// deduplicate cycles (same cycle can be found from different starting nodes)
	return deduplicateCycles(cycles)
}

// Hotspot identifies services with high coupling risk.
type Hotspot struct {
	Name       string
	FanIn      int
	FanOut     int
	Coupling   int // FanIn + FanOut
	RiskLevel  string
}

// DetectHotspots finds services with high coupling that are architectural risk points.
func DetectHotspots(g *Graph) []Hotspot {
	fanIn := make(map[string]int)
	fanOut := make(map[string]int)

	for _, dep := range g.Dependencies {
		fanOut[dep.From]++
		fanIn[dep.To]++
	}

	var hotspots []Hotspot
	for _, svc := range g.Services {
		in := fanIn[svc.Name]
		out := fanOut[svc.Name]
		coupling := in + out
		if coupling == 0 {
			continue
		}

		risk := "low"
		if coupling >= 20 {
			risk = "critical"
		} else if coupling >= 10 {
			risk = "high"
		} else if coupling >= 5 {
			risk = "medium"
		}

		hotspots = append(hotspots, Hotspot{
			Name:      svc.Name,
			FanIn:     in,
			FanOut:    out,
			Coupling:  coupling,
			RiskLevel: risk,
		})
	}

	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Coupling > hotspots[j].Coupling
	})
	return hotspots
}

func deduplicateCycles(cycles []Cycle) []Cycle {
	seen := make(map[string]bool)
	var unique []Cycle

	for _, c := range cycles {
		key := normalizeCycleKey(c.Services)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, c)
		}
	}
	return unique
}

// normalizeCycleKey creates a canonical key for a cycle regardless of starting point.
func normalizeCycleKey(services []string) string {
	if len(services) == 0 {
		return ""
	}

	// find the lexicographically smallest element and rotate
	minIdx := 0
	for i, s := range services {
		if s < services[minIdx] {
			minIdx = i
		}
	}

	rotated := make([]string, len(services))
	for i := range services {
		rotated[i] = services[(i+minIdx)%len(services)]
	}

	key := ""
	for _, s := range rotated {
		key += s + "|"
	}
	return key
}

func appendUniq(slice []string, val string) []string {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}
