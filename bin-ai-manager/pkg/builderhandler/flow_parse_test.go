package builderhandler

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"monorepo/bin-ai-manager/models/flowbuilder"
	fmaction "monorepo/bin-flow-manager/models/action"
)

func Test_FlowParse(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantErr   error
		wantMsg   string
		wantNodes []string // labels, nil means no graph
		wantWarn  []string
		wantAssum []string
	}{
		{
			name:    "message only",
			raw:     `{"message":"hi"}`,
			wantMsg: "hi",
		},
		{
			name:      "code fence around the object",
			raw:       "```json\n{\"message\":\"hi\",\"draft\":{\"nodes\":[{\"label\":\"a\",\"type\":\"hangup\"}]},\"assumptions\":[\"x\"]}\n```",
			wantMsg:   "hi",
			wantNodes: []string{"a"},
			wantAssum: []string{"x"},
		},
		{
			name:    "no object at all",
			raw:     "sorry",
			wantErr: ErrInvalidResponse,
		},
		{
			name:    "empty message is not usable",
			raw:     `{"message":""}`,
			wantErr: ErrInvalidResponse,
		},
		{
			name:     "draft that is not an object is discarded",
			raw:      `{"message":"hi","draft":"nope"}`,
			wantMsg:  "hi",
			wantWarn: []string{WarnDraftDiscarded},
		},
		{
			name:     "draft without nodes is discarded",
			raw:      `{"message":"hi","draft":{"nodes":[]}}`,
			wantMsg:  "hi",
			wantWarn: []string{WarnDraftDiscarded},
		},
		{
			name:      "nodes without a label or type are skipped",
			raw:       `{"message":"hi","draft":{"nodes":[{"type":"stop"},{"label":"x"},{"label":"ok","type":"stop"},"junk"]}}`,
			wantMsg:   "hi",
			wantNodes: []string{"ok"},
			wantAssum: []string{},
		},
		{
			name:      "a wrong typed option is dropped and reported, the node stays",
			raw:       `{"message":"hi","draft":{"nodes":[{"label":"a","type":"talk","option":"text"}]}}`,
			wantMsg:   "hi",
			wantNodes: []string{"a"},
			wantWarn:  []string{flowbuilder.WarningInvalidOption + ": a"},
			wantAssum: []string{},
		},
		{
			name:      "null next means no next",
			raw:       `{"message":"hi","draft":{"nodes":[{"label":"a","type":"stop","next":null}]}}`,
			wantMsg:   "hi",
			wantNodes: []string{"a"},
			wantAssum: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FlowParse(tt.raw)
			if err != tt.wantErr {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.wantErr, err)
			}
			if err != nil {
				return
			}
			if got.Message != tt.wantMsg {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.wantMsg, got.Message)
			}
			var labels []string
			if got.Graph != nil {
				for _, n := range got.Graph.Nodes {
					labels = append(labels, n.Label)
				}
			}
			if !reflect.DeepEqual(labels, tt.wantNodes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.wantNodes, labels)
			}
			if !reflect.DeepEqual(got.Warnings, tt.wantWarn) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.wantWarn, got.Warnings)
			}
			if tt.wantAssum != nil && !reflect.DeepEqual(got.Assumptions, tt.wantAssum) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.wantAssum, got.Assumptions)
			}
		})
	}
}

func Test_FlowParse_tooManyNodesDiscardsTheDraft(t *testing.T) {
	var nodes []string
	for i := 0; i <= flowbuilder.MaxFlowNodes; i++ {
		nodes = append(nodes, `{"label":"n`+strings.Repeat("x", i%5)+string(rune('a'+i%26))+`","type":"stop"}`)
	}
	raw := `{"message":"hi","draft":{"nodes":[` + strings.Join(nodes, ",") + `]}}`

	got, err := FlowParse(raw)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got.Graph != nil || !containsExact(got.Warnings, WarnDraftDiscarded) {
		t.Errorf("Wrong match. expect: discarded draft, got: %+v", got)
	}
}

func Test_FlowParse_nodeWithLongLabelIsSkipped(t *testing.T) {
	long := strings.Repeat("a", flowbuilder.MaxLabelRunes+1)
	raw := `{"message":"hi","draft":{"nodes":[{"label":"` + long + `","type":"stop"},{"label":"ok","type":"stop"}]}}`

	got, err := FlowParse(raw)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got.Graph == nil || len(got.Graph.Nodes) != 1 || got.Graph.Nodes[0].Label != "ok" {
		t.Errorf("Wrong match. expect: only ok, got: %+v", got.Graph)
	}
}

func Test_FlowResponseSchema_enumIsTheAllowedSet(t *testing.T) {
	schema := FlowResponseSchema([]fmaction.Type{fmaction.TypeTalk, fmaction.TypeHangup})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		t.Fatalf("Wrong match. expect: valid JSON, got: %v", err)
	}
	enum := parsed["properties"].(map[string]any)["draft"].(map[string]any)["properties"].(map[string]any)["nodes"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["type"].(map[string]any)["enum"].([]any)
	if !reflect.DeepEqual(enum, []any{"hangup", "talk"}) {
		t.Errorf("Wrong match. expect: [hangup talk], got: %v", enum)
	}
}

func Test_ReconstructGraph(t *testing.T) {
	const (
		idA = "6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"
		idB = "841c5fa2-f0c2-11ee-834f-53b2b00ec88d"
		idC = "11111111-2222-3333-4444-555555555555"
	)
	tests := []struct {
		name  string
		draft *flowbuilder.Draft
		check func(t *testing.T, g *flowbuilder.SymbolicGraph)
	}{
		{
			name:  "nil draft",
			draft: nil,
			check: func(t *testing.T, g *flowbuilder.SymbolicGraph) {
				if g != nil {
					t.Errorf("Wrong match. expect: nil, got: %+v", g)
				}
			},
		},
		{
			name: "ids become labels, resources are hidden, jump and terminate never carry next",
			draft: &flowbuilder.Draft{
				Actions: []map[string]any{
					{"id": idA, "type": "queue_join", "option": map[string]any{"queue_id": idC}, "next_id": idB},
					{"id": idB, "type": "branch", "option": map[string]any{
						"variable":          "v",
						"default_target_id": idA,
						"target_ids":        map[string]any{"1": idA, "2": "99999999-9999-9999-9999-999999999999"},
					}, "next_id": idA},
				},
				Labels: map[string]string{idA: "q", idB: "menu"},
			},
			check: func(t *testing.T, g *flowbuilder.SymbolicGraph) {
				if len(g.Nodes) != 2 {
					t.Fatalf("Wrong match. expect: 2 nodes, got: %+v", g.Nodes)
				}
				q, menu := g.Nodes[0], g.Nodes[1]
				if _, ok := q.Option["queue_id"]; ok {
					t.Errorf("Wrong match. expect: queue_id hidden, got: %v", q.Option)
				}
				if q.Next == nil || *q.Next != "menu" {
					t.Errorf("Wrong match. expect: next menu, got: %v", q.Next)
				}
				if menu.Next != nil {
					t.Errorf("Wrong match. expect: no next on a jump node, got: %v", *menu.Next)
				}
				if menu.Option["default_target_id"] != "q" {
					t.Errorf("Wrong match. expect: default_target_id q, got: %v", menu.Option["default_target_id"])
				}
				targets := menu.Option["target_ids"].(map[string]any)
				if !reflect.DeepEqual(targets, map[string]any{"1": "q"}) {
					t.Errorf("Wrong match. expect: only the known target, got: %v", targets)
				}
			},
		},
		{
			name: "client labels that are duplicated, auto-styled or too long are re-assigned",
			draft: &flowbuilder.Draft{
				Actions: []map[string]any{
					{"id": idA, "type": "stop"},
					{"id": idB, "type": "stop"},
					{"id": idC, "type": "stop"},
				},
				Labels: map[string]string{idA: "same", idB: "same", idC: "n1"},
			},
			check: func(t *testing.T, g *flowbuilder.SymbolicGraph) {
				seen := map[string]bool{}
				for _, n := range g.Nodes {
					if seen[n.Label] {
						t.Errorf("Wrong match. expect: unique labels, got: %+v", g.Nodes)
					}
					seen[n.Label] = true
				}
				if g.Nodes[0].Label != "same" {
					t.Errorf("Wrong match. expect: first same kept, got: %s", g.Nodes[0].Label)
				}
				if g.Nodes[2].Label == "n1" && g.Nodes[1].Label == "n1" {
					t.Errorf("Wrong match. expect: no collision on n1")
				}
			},
		},
		{
			name: "duplicate ids, empty ids and undecodable actions are dropped",
			draft: &flowbuilder.Draft{
				Actions: []map[string]any{
					{"id": idA, "type": "stop"},
					{"id": idA, "type": "hangup"},
					{"id": "00000000-0000-0000-0000-000000000000", "type": "stop"},
					{"id": idB, "type": 5},
				},
			},
			check: func(t *testing.T, g *flowbuilder.SymbolicGraph) {
				if len(g.Nodes) != 1 || g.Nodes[0].Type != "stop" {
					t.Errorf("Wrong match. expect: one stop node, got: %+v", g.Nodes)
				}
			},
		},
		{
			name: "a self reference and a cycle are kept as is (the executor bounds loops, design OQ9) and do not hang",
			draft: &flowbuilder.Draft{
				Actions: []map[string]any{
					{"id": idA, "type": "answer", "next_id": idA},
					{"id": idB, "type": "talk", "next_id": idC},
					{"id": idC, "type": "talk", "next_id": idB},
				},
			},
			check: func(t *testing.T, g *flowbuilder.SymbolicGraph) {
				if len(g.Nodes) != 3 || g.Nodes[0].Next == nil || *g.Nodes[0].Next != g.Nodes[0].Label {
					t.Errorf("Wrong match. expect: 3 nodes, the first pointing at itself, got: %+v", g.Nodes)
				}
			},
		},
		{
			name: "next_id pointing outside the draft is dropped",
			draft: &flowbuilder.Draft{
				Actions: []map[string]any{{"id": idA, "type": "answer", "next_id": "99999999-9999-9999-9999-999999999999"}},
			},
			check: func(t *testing.T, g *flowbuilder.SymbolicGraph) {
				if g.Nodes[0].Next != nil {
					t.Errorf("Wrong match. expect: no next, got: %v", *g.Nodes[0].Next)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, ReconstructGraph(tt.draft))
		})
	}
}

// Round trip: what the server sends as a draft is what it reads back, so a
// draft that is returned unchanged yields the same graph on the next turn
// (design doc 3.2 step 1, section 5).
func Test_ReconstructGraph_roundTripsAnAssembledDraft(t *testing.T) {
	graph := flowbuilder.SymbolicGraph{Nodes: []flowbuilder.SymbolicNode{
		{Label: "greet", Type: "talk", Option: map[string]any{"text": "hi"}, Next: strp("menu")},
		{Label: "menu", Type: "branch", Option: map[string]any{"variable": "v", "target_ids": map[string]any{"1": "greet"}, "default_target_id": "bye"}},
		{Label: "bye", Type: "hangup"},
	}}
	draft, warnings := AssembleFlowDraft(graph, allAllowed(fmaction.TypeTalk, fmaction.TypeBranch, fmaction.TypeHangup))
	if draft == nil {
		t.Fatalf("Wrong match. expect: draft, warnings: %v", warnings)
	}

	back := ReconstructGraph(draft)
	if back == nil || len(back.Nodes) != 3 {
		t.Fatalf("Wrong match. expect: 3 nodes, got: %+v", back)
	}

	// Compare as JSON: the graph is the contract with the model.
	byLabel := map[string]flowbuilder.SymbolicNode{}
	for _, n := range back.Nodes {
		byLabel[n.Label] = n
	}
	for _, want := range graph.Nodes {
		got, ok := byLabel[want.Label]
		if !ok {
			t.Errorf("Wrong match. expect: label %s kept", want.Label)
			continue
		}
		if got.Type != want.Type {
			t.Errorf("Wrong match. expect: %s, got: %s", want.Type, got.Type)
		}
		wantJSON, _ := json.Marshal(want)
		gotJSON, _ := json.Marshal(got)
		if string(wantJSON) != string(gotJSON) {
			t.Errorf("Wrong match.\nexpect: %s\ngot: %s", wantJSON, gotJSON)
		}
	}
}

func Test_FlowCatalog_isDerivedFromMetadataAndCatalog(t *testing.T) {
	got := FlowCatalog([]fmaction.Type{fmaction.TypeBranch, fmaction.TypeHangup, fmaction.TypeMessageSend, fmaction.TypeQueueJoin, fmaction.TypeAITask})

	for _, want := range []string{
		"- type branch (flow: jump; media: any)",
		"- type hangup (flow: terminate; media: call only)",
		"- type message_send (flow: continue; media: any; sensitive:",
		"- type ai_task (flow: continue; media: non-call only",
		"label fields (value is another node's label): default_target_id, target_ids (object: key to label)",
		"resource fields (always null, the user picks the resource later): queue_id",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Wrong match. expect the catalog to contain %q, got:\n%s", want, got)
		}
	}
}

// The prompt teaches two complete outputs. If a type, an option key or a
// reference stops being valid (flow-manager changed), this fails instead of
// the prompt silently teaching a stale shape.
func Test_FlowSystemPrompt_fewShotExamplesStayValid(t *testing.T) {
	for name, raw := range map[string]string{"A": flowExampleA, "B": flowExampleB} {
		t.Run(name, func(t *testing.T) {
			parsed, err := FlowParse(raw)
			if err != nil || parsed.Graph == nil {
				t.Fatalf("Wrong match. expect: a parsed draft, got: %v %+v", err, parsed)
			}
			allowed := FlowAllowedTypes(allTypeStrings())
			draft, warnings := AssembleFlowDraft(*parsed.Graph, allowedSet(allowed))
			if draft == nil {
				t.Fatalf("Wrong match. expect: a draft, got warnings: %v", warnings)
			}
			for _, w := range warnings {
				// select_resource is the example's intended output for a queue.
				if strings.HasPrefix(w, flowbuilder.WarningSelectResource) {
					continue
				}
				t.Errorf("Wrong match. expect: no warning, got: %s", w)
			}
		})
	}
}

func Test_FlowSystemPrompt_containsTheContractParts(t *testing.T) {
	got := FlowSystemPrompt([]fmaction.Type{fmaction.TypeTalk})

	for _, want := range []string{
		"Session facts", "checkpoint", "draft_exists", "user_turns",
		"지금까지의 정보로 초안을 만들어 주세요",
		"Please create the draft with the information so far.",
		"Never ask for or include secrets",
		"# Available actions",
		"- type talk",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Wrong match. expect the prompt to contain %q", want)
		}
	}
	if strings.Contains(got, "- type hangup") {
		t.Errorf("Wrong match. expect: only the allowed types in the catalog")
	}
}

func allTypeStrings() []string {
	out := make([]string, 0, len(fmaction.TypeListAll))
	for _, t := range fmaction.TypeListAll {
		out = append(out, string(t))
	}
	return out
}
