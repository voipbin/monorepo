package analysishandler

import (
	"encoding/json"
	"testing"

	"monorepo/bin-timeline-manager/models/verdict"
)

// minimalInput is the smallest collectedInput buildFinalVerdict needs for an
// ok verdict (no issues -> no evidence resolution, inventory drives resources).
func minimalInput() *collectedInput {
	return &collectedInput{
		events:    []*canonicalEvent{},
		inventory: []resourceCount{{Type: "call", Count: 1}},
	}
}

// Test_buildFinalVerdict_interactions_carried verifies the staged path's Stage 2
// interactions are carried into the final verdict (the VOIP-1200 fix: they used
// to be discarded).
func Test_buildFinalVerdict_interactions_carried(t *testing.T) {
	h := &analysisHandler{}
	raw := &verdict.RawVerdict{OverallStatus: verdict.OverallStatusOK}
	carried := []verdict.Interaction{
		{ResourceType: "call", Summary: "inbound call answered, customer asked for billing"},
	}

	got := h.buildFinalVerdict(raw, carried, minimalInput(), nil)

	if len(got.Interactions) != 1 {
		t.Fatalf("expected 1 interaction carried, got %d", len(got.Interactions))
	}
	if got.Interactions[0].ResourceType != "call" || got.Interactions[0].Summary == "" {
		t.Fatalf("interaction not carried verbatim: %+v", got.Interactions[0])
	}
	if verdict.CurrentVersion != 3 {
		t.Fatalf("expected CurrentVersion to be 3, got %d", verdict.CurrentVersion)
	}
	if got.Version != verdict.CurrentVersion {
		t.Fatalf("expected version %d, got %d", verdict.CurrentVersion, got.Version)
	}
}

// Test_buildFinalVerdict_empty_interactions_marshal_is_array is the R2 HIGH pin:
// when interactions is nil (quiet activeflow), the persisted verdict MUST
// serialize "interactions":[] and NOT "interactions":null. A len()==0 assertion
// would pass for both nil and empty, so this asserts the MARSHALED wire shape.
func Test_buildFinalVerdict_empty_interactions_marshal_is_array(t *testing.T) {
	h := &analysisHandler{}
	raw := &verdict.RawVerdict{OverallStatus: verdict.OverallStatusOK}

	// nil interactions argument == the staged quiet-call case.
	got := h.buildFinalVerdict(raw, nil, minimalInput(), nil)

	// Go-level: must be non-nil empty, not nil.
	if got.Interactions == nil {
		t.Fatal("Interactions must be normalized to a non-nil empty slice, got nil")
	}

	// Wire-level: marshaled JSON must contain "interactions":[] not :null.
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	raw2, ok := doc["interactions"]
	if !ok {
		t.Fatal("interactions key missing from marshaled verdict")
	}
	if string(raw2) != "[]" {
		t.Fatalf("expected interactions to marshal as [], got %s", string(raw2))
	}
}

// Test_buildFinalVerdict_empty_interactions_arg_marshal_is_array covers the same
// guarantee when the caller passes an already-empty (non-nil) slice.
func Test_buildFinalVerdict_empty_interactions_arg_marshal_is_array(t *testing.T) {
	h := &analysisHandler{}
	raw := &verdict.RawVerdict{OverallStatus: verdict.OverallStatusOK}

	got := h.buildFinalVerdict(raw, []verdict.Interaction{}, minimalInput(), nil)

	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if string(doc["interactions"]) != "[]" {
		t.Fatalf("expected interactions to marshal as [], got %s", string(doc["interactions"]))
	}
}

// Test_stage2Result_parse verifies the Stage 2 content is parsed from its OBJECT
// shape ({interactions, overall_narrative}), not a bare array. Unmarshaling the
// object into []Interaction would fail; the wrapper struct extracts interactions
// and tolerates overall_narrative.
func Test_stage2Result_parse(t *testing.T) {
	stage2JSON := `{
		"interactions": [
			{"resource_type": "call", "summary": "answered"},
			{"resource_type": "transcribe", "summary": "transcribed 2 utterances"}
		],
		"overall_narrative": "a normal inbound call"
	}`

	var s2 stage2Result
	if err := json.Unmarshal([]byte(stage2JSON), &s2); err != nil {
		t.Fatalf("stage2Result parse failed: %v", err)
	}
	if len(s2.Interactions) != 2 {
		t.Fatalf("expected 2 interactions parsed, got %d", len(s2.Interactions))
	}
	if s2.Interactions[1].ResourceType != "transcribe" {
		t.Fatalf("interaction not parsed correctly: %+v", s2.Interactions[1])
	}
	if s2.OverallNarrative == "" {
		t.Fatal("overall_narrative should be parsed (tolerated), got empty")
	}
}

// Test_stage2Result_parse_rejects_bare_array confirms why the wrapper struct is
// required: the Stage 2 payload is an object, so a bare-array parse target fails.
func Test_stage2Result_parse_rejects_bare_array(t *testing.T) {
	stage2JSON := `{"interactions":[],"overall_narrative":""}`
	var bare []verdict.Interaction
	if err := json.Unmarshal([]byte(stage2JSON), &bare); err == nil {
		t.Fatal("expected object->bare-array unmarshal to fail, but it succeeded")
	}
}

// Test_verdict_schema_split is the schema-split guard: the staged Stage 3 schema
// must NOT contain interactions; the combined/single-call schema MUST.
func Test_verdict_schema_split(t *testing.T) {
	type schema struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}

	var s3 schema
	if err := json.Unmarshal(stage3VerdictSchema, &s3); err != nil {
		t.Fatalf("stage3VerdictSchema not valid JSON: %v", err)
	}
	if _, ok := s3.Properties["interactions"]; ok {
		t.Fatal("stage3VerdictSchema must NOT contain interactions")
	}
	if containsString(s3.Required, "interactions") {
		t.Fatal("stage3VerdictSchema required must NOT list interactions")
	}

	var vs schema
	if err := json.Unmarshal(verdictSchema, &vs); err != nil {
		t.Fatalf("verdictSchema not valid JSON: %v", err)
	}
	if _, ok := vs.Properties["interactions"]; !ok {
		t.Fatal("verdictSchema MUST contain interactions property")
	}
	if !containsString(vs.Required, "interactions") {
		t.Fatal("verdictSchema required MUST list interactions")
	}

	// The derived schema must preserve all of stage3's shared properties.
	for k := range s3.Properties {
		if _, ok := vs.Properties[k]; !ok {
			t.Fatalf("verdictSchema dropped shared property %q from stage3VerdictSchema", k)
		}
	}
}

func containsString(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}
