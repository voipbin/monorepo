package builderhandler

// flow_layout.go computes array order and canvas positions for a transcoded
// Draft (VOIP-1573 design doc §3.2 steps 6-7, §3.3's sink definition). None
// of it reads an action type name; it only reads fmaction.MetaByType.Flow
// and the actions' own next_id links.

import (
	"encoding/json"

	"monorepo/bin-ai-manager/models/flowbuilder"
	fmaction "monorepo/bin-flow-manager/models/action"
)

func actionToMap(a fmaction.Action) (map[string]any, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// isOpenEnd reports whether a is a FlowKindContinue action with no outgoing
// next_id -- the only kind of node whose "next" is array adjacency rather
// than an explicit link (design doc §3.2 step 6, §3.3).
func isOpenEnd(a fmaction.Action) bool {
	meta, ok := fmaction.MetaByType[a.Type]
	flow := fmaction.FlowKindContinue
	if ok {
		flow = meta.Flow
	}
	return flow == fmaction.FlowKindContinue && a.NextID == fmaction.IDEmpty
}

// reorderStartFirstOpenEndsLast keeps actions[0] fixed as the start node
// (flow-manager's IDStart convention) and moves every open end to the back
// of the slice, preserving relative order within each group, so array-
// adjacency fall-through only ever lands on the next open end -- never on
// an arbitrary later node the LLM happened to list first (design doc §3.2
// step 6).
func reorderStartFirstOpenEndsLast(actions []fmaction.Action) []fmaction.Action {
	if len(actions) == 0 {
		return actions
	}
	start := actions[0]
	rest := actions[1:]

	out := make([]fmaction.Action, 0, len(actions))
	out = append(out, start)

	var nonOpenEnds, openEnds []fmaction.Action
	for _, a := range rest {
		if isOpenEnd(a) {
			openEnds = append(openEnds, a)
		} else {
			nonOpenEnds = append(nonOpenEnds, a)
		}
	}
	out = append(out, nonOpenEnds...)
	out = append(out, openEnds...)
	return out
}

const (
	layoutStartX     = 0
	layoutStartY     = 100
	layoutDepthStepY = 500
	layoutColStepX   = 450
)

// successors returns, per action index, the indexes the executor can reach
// next: the next_id target, every ref:"action" target, and, for an open end
// that is not the last element, the next array element (the executor's
// array-adjacency fall-through, design doc 3.3). Layout and the unreachable
// check share it so "unreachable" and "placed in the unreachable row" agree.
func successors(actions []fmaction.Action) [][]int {
	byID := make(map[string]int, len(actions))
	for i, a := range actions {
		byID[a.ID.String()] = i
	}

	adj := make([][]int, len(actions))
	for i, a := range actions {
		if a.NextID != fmaction.IDEmpty {
			if j, ok := byID[a.NextID.String()]; ok {
				adj[i] = append(adj[i], j)
			}
		}
		for _, f := range fmaction.RefFieldsOf(a.Type) {
			if f.Kind != fmaction.RefKindAction {
				continue
			}
			for _, target := range extractActionRefTargets(a.Option, f) {
				if j, ok := byID[target]; ok {
					adj[i] = append(adj[i], j)
				}
			}
		}
		if isOpenEnd(a) && i < len(actions)-1 {
			adj[i] = append(adj[i], i+1)
		}
	}
	return adj
}

// computeLayout assigns a grid {x,y} per action: x by order-of-appearance
// within a BFS depth from the start node, y by depth. Edges are those of
// successors (so a branch's targets land one row further down, like the
// hand-built templates). Unreachable nodes go in one extra row below the
// deepest reachable one (design doc 3.2 step 7).
func computeLayout(actions []fmaction.Action) []flowbuilder.Position {
	if len(actions) == 0 {
		return nil
	}

	adj := successors(actions)

	depth := make([]int, len(actions))
	for i := range depth {
		depth[i] = -1
	}
	depth[0] = 0
	queue := []int{0}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if depth[next] == -1 {
				depth[next] = depth[cur] + 1
				queue = append(queue, next)
			}
		}
	}

	maxDepth := 0
	for _, d := range depth {
		if d > maxDepth {
			maxDepth = d
		}
	}
	unreachableDepth := maxDepth + 1

	colAtDepth := map[int]int{}
	positions := make([]flowbuilder.Position, len(actions))
	for i, d := range depth {
		if d == -1 {
			d = unreachableDepth
		}
		col := colAtDepth[d]
		colAtDepth[d] = col + 1
		positions[i] = flowbuilder.Position{
			X: layoutStartX + col*layoutColStepX,
			Y: layoutStartY + d*layoutDepthStepY,
		}
	}
	return positions
}

// extractActionRefTargets reads a ref:"action" field's resolved UUID
// string(s) out of an already-transcoded Action.Option map (the value is a
// UUID string for a scalar field, or a map[string]any of UUID strings for a
// map field).
func extractActionRefTargets(opt map[string]any, f fmaction.RefField) []string {
	if f.IsMap {
		raw, ok := opt[f.JSONName].(map[string]any)
		if !ok {
			return nil
		}
		out := make([]string, 0, len(raw))
		for _, v := range raw {
			if s, ok := v.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	if s, ok := opt[f.JSONName].(string); ok && s != "" {
		return []string{s}
	}
	return nil
}
