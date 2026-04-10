package analyzer

import "sort"

// Path represents a dependency chain from one service to another.
type Path struct {
	Services []string       // ordered list of services in the path
	Types    []DependencyType // dependency type for each hop (len = len(Services)-1)
}

// FindAllPaths finds all dependency paths from source to target using DFS.
// maxDepth limits the search depth to avoid combinatorial explosion.
func FindAllPaths(g *Graph, source, target string, maxDepth int) []Path {
	if maxDepth <= 0 {
		maxDepth = 10
	}

	adj := buildAdjacency(g)

	var paths []Path
	visited := make(map[string]bool)
	visited[source] = true

	var dfs func(current string, pathSvcs []string, pathTypes []DependencyType)
	dfs = func(current string, pathSvcs []string, pathTypes []DependencyType) {
		if current == target {
			p := Path{
				Services: make([]string, len(pathSvcs)),
				Types:    make([]DependencyType, len(pathTypes)),
			}
			copy(p.Services, pathSvcs)
			copy(p.Types, pathTypes)
			paths = append(paths, p)
			return
		}

		if len(pathSvcs) > maxDepth {
			return
		}

		for _, edge := range adj[current] {
			if !visited[edge.to] {
				visited[edge.to] = true
				dfs(edge.to, append(pathSvcs, edge.to), append(pathTypes, edge.depType))
				visited[edge.to] = false
			}
		}
	}

	dfs(source, []string{source}, nil)

	// sort by path length (shortest first)
	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i].Services) < len(paths[j].Services)
	})

	return paths
}

// FindShortestPath returns the shortest dependency path from source to target using BFS.
// Returns nil if no path exists.
func FindShortestPath(g *Graph, source, target string) *Path {
	adj := buildAdjacency(g)

	type bfsNode struct {
		service  string
		path     []string
		types    []DependencyType
	}

	visited := make(map[string]bool)
	visited[source] = true
	queue := []bfsNode{{service: source, path: []string{source}}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range adj[current.service] {
			if edge.to == target {
				return &Path{
					Services: append(current.path, target),
					Types:    append(current.types, edge.depType),
				}
			}
			if !visited[edge.to] {
				visited[edge.to] = true
				newPath := make([]string, len(current.path))
				copy(newPath, current.path)
				newTypes := make([]DependencyType, len(current.types))
				copy(newTypes, current.types)
				queue = append(queue, bfsNode{
					service: edge.to,
					path:    append(newPath, edge.to),
					types:   append(newTypes, edge.depType),
				})
			}
		}
	}

	return nil
}

type adjEdge struct {
	to      string
	depType DependencyType
}

func buildAdjacency(g *Graph) map[string][]adjEdge {
	adj := make(map[string][]adjEdge)
	for _, dep := range g.Dependencies {
		adj[dep.From] = append(adj[dep.From], adjEdge{to: dep.To, depType: dep.Type})
	}
	return adj
}
