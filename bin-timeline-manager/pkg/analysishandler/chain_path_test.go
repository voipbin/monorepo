package analysishandler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	amanalysis "monorepo/bin-ai-manager/models/analysis"

	"go.uber.org/mock/gomock"

	"monorepo/bin-timeline-manager/models/verdict"
)

// stagedInput is a collectedInput that forces the staged (3-stage) path:
// >= analysisStageThresholdEvents canonical events.
func stagedInput() *collectedInput {
	events := make([]*canonicalEvent, analysisStageThresholdEvents)
	for i := range events {
		events[i] = &canonicalEvent{Index: i, EventType: "call_event", Summary: "e"}
	}
	return &collectedInput{
		events:    events,
		inventory: []resourceCount{{Type: "call", Count: 1}},
	}
}

// gatewayResp is a small helper to build a gateway response with a JSON body.
func gatewayResp(t *testing.T, body string) *amanalysis.Response {
	t.Helper()
	return &amanalysis.Response{
		Result:       json.RawMessage(body),
		Model:        "m",
		FinishReason: "stop",
	}
}

// Test_runStaged_carries_stage2_interactions_end_to_end proves the staged path
// actually wires the Stage 2 interactions into the returned slice AND that
// Stage 3 runs on stage3VerdictSchema (no interactions). This pins the whole
// VOIP-1200 carry glue, not just buildFinalVerdict copying a slice it was given:
// if runStaged returned nil instead of s2.Interactions, this fails.
func Test_runStaged_carries_stage2_interactions_end_to_end(t *testing.T) {
	h, reqMock, _, mc := newStartTestHandler(t)
	defer mc.Finish()

	stage1Body := `{"resources_used":[{"type":"call","count":1}],"event_outline":[]}`
	stage2Body := `{"interactions":[{"resource_type":"call","summary":"answered, billing question"}],"overall_narrative":"normal call"}`
	stage3Body := `{"overall_status":"ok","resources_used":[{"type":"call","count":1}],"narrative":"ok","issues":[]}`

	// Capture the schema each stage was invoked with so we can assert Stage 3
	// uses stage3VerdictSchema (NOT verdictSchema).
	var sawSchemas []string
	reqMock.EXPECT().
		AIV1ServiceTypeAnalysisRun(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *amanalysis.Request, _ int) (*amanalysis.Response, error) {
			sawSchemas = append(sawSchemas, string(req.Schema))
			switch req.SchemaName {
			case "timeline_stage1":
				return gatewayResp(t, stage1Body), nil
			case "timeline_stage2":
				return gatewayResp(t, stage2Body), nil
			case "timeline_verdict":
				return gatewayResp(t, stage3Body), nil
			default:
				t.Fatalf("unexpected schema name: %s", req.SchemaName)
				return nil, nil
			}
		}).Times(3)

	_, _, interactions, err := h.runStaged(context.Background(), stagedInput(), nil)
	if err != nil {
		t.Fatalf("runStaged failed: %v", err)
	}

	// The Stage 2 interactions must be carried out of runStaged.
	if len(interactions) != 1 || interactions[0].ResourceType != "call" || interactions[0].Summary == "" {
		t.Fatalf("Stage 2 interactions not carried: %+v", interactions)
	}

	// Stage 3 (the 3rd call) must use stage3VerdictSchema, which has NO
	// interactions. Guard against a regression that points it back at verdictSchema.
	stage3Schema := sawSchemas[2]
	if strings.Contains(stage3Schema, `"interactions"`) {
		t.Fatalf("Stage 3 must use stage3VerdictSchema (no interactions), got: %s", stage3Schema)
	}
}

// Test_runCombined_emits_interactions_end_to_end proves the single-call path
// uses verdictSchema (WITH interactions) and that raw.Interactions flows into
// the final persisted verdict via runChain's `if !staged` source selection.
// If `interactions = raw.Interactions` were dropped, the final verdict would
// lose them and this fails.
func Test_runCombined_emits_interactions_end_to_end(t *testing.T) {
	h, reqMock, _, mc := newStartTestHandler(t)
	defer mc.Finish()

	combinedBody := `{"overall_status":"ok","resources_used":[{"type":"call","count":1}],"interactions":[{"resource_type":"call","summary":"short call answered"}],"narrative":"ok","issues":[]}`

	var sawSchema string
	reqMock.EXPECT().
		AIV1ServiceTypeAnalysisRun(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *amanalysis.Request, _ int) (*amanalysis.Response, error) {
			sawSchema = string(req.Schema)
			return gatewayResp(t, combinedBody), nil
		}).Times(1)

	rawVerdict, _, err := h.runCombined(context.Background(), minimalInput(), nil)
	if err != nil {
		t.Fatalf("runCombined failed: %v", err)
	}

	// The combined call must use verdictSchema (WITH interactions).
	if !strings.Contains(sawSchema, `"interactions"`) {
		t.Fatalf("combined path must use verdictSchema (with interactions), got: %s", sawSchema)
	}

	// Mirror runChain's single-call source selection and assert the interaction
	// survives into the final verdict.
	raw, err := verdict.ValidateRaw(rawVerdict, len(minimalInput().events))
	if err != nil {
		t.Fatalf("ValidateRaw failed: %v", err)
	}
	final := h.buildFinalVerdict(raw, raw.Interactions, minimalInput(), nil)
	if len(final.Interactions) != 1 || final.Interactions[0].Summary == "" {
		t.Fatalf("combined-path interaction did not flow into final verdict: %+v", final.Interactions)
	}
}
