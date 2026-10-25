package analysishandler

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	amanalysis "monorepo/bin-ai-manager/models/analysis"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/models/verdict"
)

var (
	chainRunCustomerID = uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	chainRunAFID       = uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
)

func Test_useSingleCall(t *testing.T) {
	tests := []struct {
		name        string
		events      int
		transcripts []string
		expectRes   bool
	}{
		{"49 events, no transcript", 49, nil, true},
		{"50 events, no transcript", 50, nil, false},
		{"3999 runes in one transcript", 2, []string{strings.Repeat("a", 3999)}, true},
		{"4000 runes in one transcript", 2, []string{strings.Repeat("a", 4000)}, false},
		{"3999 runes split in two", 2, []string{strings.Repeat("a", 1999), strings.Repeat("b", 2000)}, true},
		{"4000 runes split in two", 2, []string{strings.Repeat("a", 2000), strings.Repeat("b", 2000)}, false},
		{"3999 multi-byte runes exceed 4000 bytes", 2, []string{strings.Repeat("한", 3999)}, true},
		{"4000 multi-byte runes", 2, []string{strings.Repeat("한", 4000)}, false},
	}

	h := &analysisHandler{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &collectedInput{
				events:      make([]*canonicalEvent, tt.events),
				transcripts: tt.transcripts,
			}
			if res := h.useSingleCall(input); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_callGateway(t *testing.T) {
	tests := []struct {
		name         string
		response     *amanalysis.Response
		responseErr  error
		expectErrMsg string
	}{
		{
			name:         "rpc error",
			responseErr:  errors.New("rpc down"),
			expectErrMsg: "gateway call failed",
		},
		{
			name:         "nil response",
			expectErrMsg: "gateway returned nil response",
		},
		{
			name:         "truncated flag with stop reason",
			response:     &amanalysis.Response{Result: json.RawMessage(`{}`), Truncated: true, FinishReason: "stop"},
			expectErrMsg: "gateway output truncated",
		},
		{
			name:         "length reason without truncated flag",
			response:     &amanalysis.Response{Result: json.RawMessage(`{}`), Truncated: false, FinishReason: "length"},
			expectErrMsg: "gateway output truncated (finish_reason=length)",
		},
		{
			name:         "nil result",
			response:     &amanalysis.Response{FinishReason: "stop"},
			expectErrMsg: "gateway returned empty result",
		},
		{
			name:         "empty result",
			response:     &amanalysis.Response{Result: json.RawMessage{}, FinishReason: "stop"},
			expectErrMsg: "gateway returned empty result",
		},
		{
			name:     "ok",
			response: &amanalysis.Response{Result: json.RawMessage(`{"a":1}`), Model: "model-x", FinishReason: "stop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, reqMock, _, _, _, mc := newRunTestHandler(t)
			defer mc.Finish()

			var sawReq *amanalysis.Request
			reqMock.EXPECT().AIV1ServiceTypeAnalysisRun(gomock.Any(), gomock.Any(), 120000).
				DoAndReturn(func(_ context.Context, req *amanalysis.Request, _ int) (*amanalysis.Response, error) {
					sawReq = req
					return tt.response, tt.responseErr
				}).Times(1)

			res, err := h.callGateway(context.Background(), "prompt-x", json.RawMessage(`{"d":1}`), json.RawMessage(`{"type":"object"}`), "schema-x", "model-y")

			if sawReq == nil || sawReq.Prompt != "prompt-x" || string(sawReq.Data) != `{"d":1}` || string(sawReq.Schema) != `{"type":"object"}` || sawReq.SchemaName != "schema-x" || sawReq.Model != "model-y" {
				t.Errorf("Wrong match. expect: the request fields passed through, got: %+v", sawReq)
			}

			if tt.expectErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErrMsg) {
					t.Errorf("Wrong match. expect: error containing %q, got: %v", tt.expectErrMsg, err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil response, got: %+v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.response {
				t.Errorf("Wrong match. expect: the gateway response, got: %+v", res)
			}
		})
	}
}

func Test_runStaged_failures(t *testing.T) {
	const stage1Body = `{"resources_used":[],"event_outline":[]}`
	const stage2Body = `{"interactions":[],"overall_narrative":"n"}`

	tests := []struct {
		name         string
		failStage    string
		stage2Body   string
		expectCalls  []string
		expectErrMsg string
	}{
		{
			name:         "stage1 fails",
			failStage:    "timeline_stage1",
			stage2Body:   stage2Body,
			expectCalls:  []string{"m1:timeline_stage1"},
			expectErrMsg: "runStaged: stage1",
		},
		{
			name:         "stage2 fails",
			failStage:    "timeline_stage2",
			stage2Body:   stage2Body,
			expectCalls:  []string{"m1:timeline_stage1", "m2:timeline_stage2"},
			expectErrMsg: "runStaged: stage2:",
		},
		{
			name:         "stage2 content is valid JSON but not an object",
			stage2Body:   `[]`,
			expectCalls:  []string{"m1:timeline_stage1", "m2:timeline_stage2"},
			expectErrMsg: "could not parse stage2 content",
		},
		{
			name:         "stage3 fails",
			failStage:    "timeline_verdict",
			stage2Body:   stage2Body,
			expectCalls:  []string{"m1:timeline_stage1", "m2:timeline_stage2", "m3:timeline_verdict"},
			expectErrMsg: "runStaged: stage3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, reqMock, _, _, _, mc := newRunTestHandler(t)
			defer mc.Finish()

			var calls []string
			reqMock.EXPECT().AIV1ServiceTypeAnalysisRun(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, req *amanalysis.Request, _ int) (*amanalysis.Response, error) {
					calls = append(calls, req.Model+":"+req.SchemaName)
					if req.SchemaName == tt.failStage {
						return nil, errors.New("gateway unavailable")
					}
					switch req.SchemaName {
					case "timeline_stage1":
						return gatewayResp(t, stage1Body), nil
					case "timeline_stage2":
						return gatewayResp(t, tt.stage2Body), nil
					}
					return gatewayResp(t, `{}`), nil
				}).Times(len(tt.expectCalls))

			_, _, _, err := h.runStaged(context.Background(), stagedInput(), nil)
			if err == nil || !strings.Contains(err.Error(), tt.expectErrMsg) {
				t.Errorf("Wrong match. expect: error containing %q, got: %v", tt.expectErrMsg, err)
			}
			if !reflect.DeepEqual(calls, tt.expectCalls) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCalls, calls)
			}
		})
	}
}

func Test_runChain_single_call(t *testing.T) {
	h, reqMock, _, evMock, _, mc := newRunTestHandler(t)
	defer mc.Finish()

	evMock.EXPECT().AggregatedList(gomock.Any(), chainRunAFID, "", 500).Return(&event.AggregatedEventListResponse{
		Result: []*event.Event{collectEvt(0, "call_created"), collectEvt(1, "call_hangup")},
	}, nil)
	reqMock.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), map[tmtranscribe.Field]any{
		tmtranscribe.FieldCustomerID:   chainRunCustomerID,
		tmtranscribe.FieldActiveflowID: chainRunAFID,
		tmtranscribe.FieldDeleted:      false,
	}).Return([]tmtranscribe.Transcribe{}, nil)

	var sawModel, sawSchemaName string
	reqMock.EXPECT().AIV1ServiceTypeAnalysisRun(gomock.Any(), gomock.Any(), 120000).
		DoAndReturn(func(_ context.Context, req *amanalysis.Request, _ int) (*amanalysis.Response, error) {
			sawModel = req.Model
			sawSchemaName = req.SchemaName
			body := `{"overall_status":"ok","resources_used":[{"type":"llm-made-up","count":99}],` +
				`"interactions":[{"resource_type":"call","summary":"short call"}],"narrative":"fine","issues":[]}`
			return &amanalysis.Response{Result: json.RawMessage(body), Model: "model-c", FinishReason: "stop"}, nil
		}).Times(1)

	out, model, err := h.runChain(context.Background(), chainRunCustomerID, chainRunAFID, endedActiveflow(chainRunAFID, chainRunCustomerID))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if model != "model-c" {
		t.Errorf("Wrong match. expect: model-c, got: %s", model)
	}
	if sawModel != "m3" || sawSchemaName != "timeline_verdict" {
		t.Errorf("Wrong match. expect: the stage3 model with the full verdict schema, got: %s/%s", sawModel, sawSchemaName)
	}

	var got verdict.Verdict
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("could not parse the verdict. err: %v", err)
	}
	// resources_used is the Go inventory, interactions come from the combined emission.
	expectResources := []verdict.ResourceUsed{{Type: "call-manager", Count: 2}}
	if !reflect.DeepEqual(got.ResourcesUsed, expectResources) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectResources, got.ResourcesUsed)
	}
	expectInteractions := []verdict.Interaction{{ResourceType: "call", Summary: "short call"}}
	if !reflect.DeepEqual(got.Interactions, expectInteractions) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectInteractions, got.Interactions)
	}
	if got.OverallStatus != verdict.OverallStatusOK || got.Narrative != "fine" || got.Version != 3 {
		t.Errorf("Wrong match. expect: ok/fine/3, got: %s/%s/%d", got.OverallStatus, got.Narrative, got.Version)
	}
}

func Test_runChain_staged(t *testing.T) {
	h, reqMock, _, evMock, _, mc := newRunTestHandler(t)
	defer mc.Finish()

	// every event differs in timestamp and data so the dedup key keeps all 50.
	evMock.EXPECT().AggregatedList(gomock.Any(), chainRunAFID, "", 500).Return(&event.AggregatedEventListResponse{
		Result: collectEvts(0, 50, "call_event"),
	}, nil)
	reqMock.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), map[tmtranscribe.Field]any{
		tmtranscribe.FieldCustomerID:   chainRunCustomerID,
		tmtranscribe.FieldActiveflowID: chainRunAFID,
		tmtranscribe.FieldDeleted:      false,
	}).Return([]tmtranscribe.Transcribe{}, nil)

	var calls []string
	reqMock.EXPECT().AIV1ServiceTypeAnalysisRun(gomock.Any(), gomock.Any(), 120000).
		DoAndReturn(func(_ context.Context, req *amanalysis.Request, _ int) (*amanalysis.Response, error) {
			calls = append(calls, req.Model+":"+req.SchemaName)
			switch req.SchemaName {
			case "timeline_stage1":
				return &amanalysis.Response{Result: json.RawMessage(`{"resources_used":[],"event_outline":[]}`), Model: "model-1", FinishReason: "stop"}, nil
			case "timeline_stage2":
				return &amanalysis.Response{Result: json.RawMessage(`{"interactions":[{"resource_type":"call","summary":"from stage two"}],"overall_narrative":"n"}`), Model: "model-2", FinishReason: "stop"}, nil
			}
			return &amanalysis.Response{Result: json.RawMessage(`{"overall_status":"ok","resources_used":[{"type":"llm-made-up","count":7}],"narrative":"staged","issues":[]}`), Model: "model-3", FinishReason: "stop"}, nil
		}).Times(3)

	out, model, err := h.runChain(context.Background(), chainRunCustomerID, chainRunAFID, endedActiveflow(chainRunAFID, chainRunCustomerID))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectCalls := []string{"m1:timeline_stage1", "m2:timeline_stage2", "m3:timeline_verdict"}
	if !reflect.DeepEqual(calls, expectCalls) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectCalls, calls)
	}
	if model != "model-3" {
		t.Errorf("Wrong match. expect: the stage3 model, got: %s", model)
	}

	var got verdict.Verdict
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("could not parse the verdict. err: %v", err)
	}
	// the interactions come from stage 2 (stage 3 emitted none), resources from the Go inventory.
	expectInteractions := []verdict.Interaction{{ResourceType: "call", Summary: "from stage two"}}
	if !reflect.DeepEqual(got.Interactions, expectInteractions) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectInteractions, got.Interactions)
	}
	expectResources := []verdict.ResourceUsed{{Type: "call-manager", Count: 50}}
	if !reflect.DeepEqual(got.ResourcesUsed, expectResources) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectResources, got.ResourcesUsed)
	}
}

func Test_runChain_collect_error(t *testing.T) {
	h, _, _, evMock, _, mc := newRunTestHandler(t)
	defer mc.Finish()

	boom := errors.New("backend failure")
	evMock.EXPECT().AggregatedList(gomock.Any(), chainRunAFID, "", 500).Return(nil, boom)

	out, model, err := h.runChain(context.Background(), chainRunCustomerID, chainRunAFID, endedActiveflow(chainRunAFID, chainRunCustomerID))
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "could not collect input") {
		t.Errorf("Wrong match. expect: wrapped collect error, got: %v", err)
	}
	if out != nil || model != "" {
		t.Errorf("Wrong match. expect: no output, got: %q/%q", out, model)
	}
}

func Test_runChain_failures(t *testing.T) {
	tests := []struct {
		name         string
		gatewayBody  string
		gatewayErr   error
		expectErrMsg string
	}{
		{
			name:         "gateway error",
			gatewayErr:   errors.New("gateway unavailable"),
			expectErrMsg: "runCombined: gateway call failed",
		},
		{
			name:         "unknown overall status",
			gatewayBody:  `{"overall_status":"bogus","resources_used":[],"narrative":"n","issues":[]}`,
			expectErrMsg: "raw verdict validation failed",
		},
		{
			name:         "evidence index out of range",
			gatewayBody:  `{"overall_status":"ok","resources_used":[],"narrative":"n","issues":[{"severity":"info","area":"call","summary":"s","evidence_index":[99]}]}`,
			expectErrMsg: "out of range",
		},
		{
			name:         "result is not an object",
			gatewayBody:  `"x"`,
			expectErrMsg: "raw verdict validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, reqMock, _, evMock, _, mc := newRunTestHandler(t)
			defer mc.Finish()

			evMock.EXPECT().AggregatedList(gomock.Any(), chainRunAFID, "", 500).Return(&event.AggregatedEventListResponse{
				Result: []*event.Event{collectEvt(0, "call_created"), collectEvt(1, "call_hangup")},
			}, nil)
			reqMock.EXPECT().TranscribeV1TranscribeList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]tmtranscribe.Transcribe{}, nil)

			var sawModel string
			reqMock.EXPECT().AIV1ServiceTypeAnalysisRun(gomock.Any(), gomock.Any(), 120000).
				DoAndReturn(func(_ context.Context, req *amanalysis.Request, _ int) (*amanalysis.Response, error) {
					sawModel = req.Model
					if tt.gatewayErr != nil {
						return nil, tt.gatewayErr
					}
					return &amanalysis.Response{Result: json.RawMessage(tt.gatewayBody), Model: "m", FinishReason: "stop"}, nil
				}).Times(1)

			out, _, err := h.runChain(context.Background(), chainRunCustomerID, chainRunAFID, endedActiveflow(chainRunAFID, chainRunCustomerID))
			if err == nil || !strings.Contains(err.Error(), tt.expectErrMsg) {
				t.Errorf("Wrong match. expect: error containing %q, got: %v", tt.expectErrMsg, err)
			}
			if out != nil {
				t.Errorf("Wrong match. expect: no output, got: %q", out)
			}
			if sawModel != "m3" {
				t.Errorf("Wrong match. expect: the combined call to use the stage3 model, got: %s", sawModel)
			}
		})
	}
}

func Test_addContextBlock(t *testing.T) {
	sc := &verdict.SessionContext{ReferenceType: "call"}
	outcome := &verdict.SessionOutcome{Result: "completed"}
	metrics := &verdict.SessionMetrics{TurnsUser: 3}

	tests := []struct {
		name         string
		enrichment   *sessionEnrichment
		expectKeys   []string
		expectValues map[string]string
	}{
		{"nil enrichment", nil, nil, nil},
		{"nothing resolved", &sessionEnrichment{}, nil, nil},
		{"context only", &sessionEnrichment{ctx: sc}, []string{"session_context"}, nil},
		{"outcome only", &sessionEnrichment{outcome: outcome}, []string{"outcome"}, nil},
		{"metrics only", &sessionEnrichment{metrics: metrics}, []string{"metrics"}, nil},
		{
			"all blocks",
			&sessionEnrichment{ctx: sc, outcome: outcome, metrics: metrics},
			[]string{"metrics", "outcome", "session_context"},
			map[string]string{
				"outcome":         `{"result":"completed"}`,
				"metrics":         `{"turns_user":3,"turns_bot":0}`,
				"session_context": `{"reference_type":"call","channel":"","multi_leg":false,"ai_handled":false,"human_involved":false}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]any{"event_count": 1}
			addContextBlock(payload, tt.enrichment)

			b, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("could not marshal the payload. err: %v", err)
			}
			var doc map[string]json.RawMessage
			if err := json.Unmarshal(b, &doc); err != nil {
				t.Fatalf("could not parse the payload. err: %v", err)
			}

			rawBlock, ok := doc["context"]
			if tt.expectKeys == nil {
				if ok {
					t.Errorf("Wrong match. expect: no context key, got: %s", rawBlock)
				}
				return
			}
			if !ok {
				t.Fatalf("Wrong match. expect: a context key, got: none")
			}
			var block map[string]json.RawMessage
			if err := json.Unmarshal(rawBlock, &block); err != nil {
				t.Fatalf("could not parse the context block. err: %v", err)
			}
			keys := []string{}
			for k := range block {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			if !reflect.DeepEqual(keys, tt.expectKeys) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectKeys, keys)
			}
			for k, expect := range tt.expectValues {
				var expectDoc, gotDoc any
				if err := json.Unmarshal([]byte(expect), &expectDoc); err != nil {
					t.Fatalf("could not parse the expectation for %s. err: %v", k, err)
				}
				if err := json.Unmarshal(block[k], &gotDoc); err != nil {
					t.Fatalf("could not parse the block %s. err: %v", k, err)
				}
				if !reflect.DeepEqual(expectDoc, gotDoc) {
					t.Errorf("Wrong match. expect: %s=%s, got: %s", k, expect, block[k])
				}
			}
		})
	}
}

func Test_buildFinalVerdict_issues_and_enrichment(t *testing.T) {
	h := &analysisHandler{}
	input := &collectedInput{
		events: []*canonicalEvent{
			{Index: 0, EventType: "call_created", Timestamp: "2026-01-02T03:00:00.000000Z", ResourceID: "r0"},
			{Index: 1, EventType: "call_hangup", Timestamp: "2026-01-02T03:00:01.000000Z", ResourceID: "r1"},
		},
		inventory:    []resourceCount{{Type: "call-manager", Count: 2}},
		inputReduced: true,
	}
	raw := &verdict.RawVerdict{
		OverallStatus: verdict.OverallStatusError,
		Narrative:     "n",
		Issues: []verdict.RawIssue{
			{Severity: verdict.SeverityError, Area: "call", Summary: "s", EvidenceIndex: []int{1, 0}},
			{Severity: verdict.SeverityInfo, Area: "flow", Summary: "t", EvidenceIndex: nil},
		},
	}
	enrichment := &sessionEnrichment{
		ctx:     &verdict.SessionContext{ReferenceType: "call"},
		outcome: &verdict.SessionOutcome{Result: "completed"},
		metrics: &verdict.SessionMetrics{TurnsUser: 3},
	}

	got := h.buildFinalVerdict(raw, nil, input, enrichment)

	expectIssues := []verdict.Issue{
		{Severity: verdict.SeverityError, Area: "call", Summary: "s", Evidence: []verdict.Evidence{
			{EvidenceIndex: 1, EventType: "call_hangup", Timestamp: "2026-01-02T03:00:01.000000Z", ResourceID: "r1"},
			{EvidenceIndex: 0, EventType: "call_created", Timestamp: "2026-01-02T03:00:00.000000Z", ResourceID: "r0"},
		}},
		{Severity: verdict.SeverityInfo, Area: "flow", Summary: "t", Evidence: []verdict.Evidence{}},
	}
	if !reflect.DeepEqual(got.Issues, expectIssues) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expectIssues, got.Issues)
	}
	if !got.InputReduced {
		t.Errorf("Wrong match. expect: input_reduced carried, got: false")
	}
	if got.SessionContext == nil || got.SessionContext.ReferenceType != "call" || got.Outcome == nil || got.Outcome.Result != "completed" || got.Metrics == nil || got.Metrics.TurnsUser != 3 {
		t.Errorf("Wrong match. expect: the enrichment blocks attached, got: %+v/%+v/%+v", got.SessionContext, got.Outcome, got.Metrics)
	}

	// without enrichment the three blocks are omitted from the wire shape.
	plain := h.buildFinalVerdict(raw, nil, input, nil)
	b, err := json.Marshal(plain)
	if err != nil {
		t.Fatalf("could not marshal the verdict. err: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("could not parse the verdict. err: %v", err)
	}
	for _, key := range []string{"session_context", "outcome", "metrics"} {
		if _, ok := doc[key]; ok {
			t.Errorf("Wrong match. expect: %s omitted, got: present", key)
		}
	}
}

func Test_buildCombinedData_error_signals(t *testing.T) {
	h := &analysisHandler{}
	input := &collectedInput{
		events: []*canonicalEvent{
			{Index: 0, EventType: "call_created", Publisher: "call-manager", Timestamp: "t0", Summary: "up"},
			{Index: 1, EventType: "call_hangup", Publisher: "call-manager", Timestamp: "t1", Summary: "down"},
		},
		inventory:    []resourceCount{{Type: "call-manager", Count: 2}},
		errorSignals: []*canonicalEvent{{Index: 1, EventType: "call_hangup", Publisher: "call-manager", Timestamp: "t1", Summary: "down"}},
		transcripts:  []string{"[in] hi"},
	}
	enrichment := &sessionEnrichment{outcome: &verdict.SessionOutcome{Result: "completed"}}

	decode := func(b json.RawMessage) map[string]json.RawMessage {
		t.Helper()
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatalf("could not parse the payload. err: %v", err)
		}
		return doc
	}
	asString := func(raw json.RawMessage) string {
		t.Helper()
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatalf("could not parse a string field. err: %v", err)
		}
		return s
	}

	combined := decode(h.buildCombinedData(input, enrichment))
	if got := asString(combined["error_signals"]); got != "1\tt1\tcall_hangup\tdown\n" {
		t.Errorf("Wrong match. expect: the single hangup signal line, got: %q", got)
	}
	if got := asString(combined["events"]); got != "0\tt0\tcall_created\tcall-manager\tup\n1\tt1\tcall_hangup\tcall-manager\tdown\n" {
		t.Errorf("Wrong match. expect: the two indexed event lines, got: %q", got)
	}
	if got := asString(combined["inventory"]); got != "call-manager\t2\n" {
		t.Errorf("Wrong match. expect: the inventory line, got: %q", got)
	}
	if _, ok := combined["context"]; !ok {
		t.Errorf("Wrong match. expect: a context block, got: none")
	}
}

func Test_buildStage3Data_error_signals(t *testing.T) {
	h := &analysisHandler{}
	input := &collectedInput{
		events: []*canonicalEvent{
			{Index: 0, EventType: "call_created", Publisher: "call-manager", Timestamp: "t0", Summary: "up"},
			{Index: 1, EventType: "call_hangup", Publisher: "call-manager", Timestamp: "t1", Summary: "down"},
		},
		inventory:    []resourceCount{{Type: "call-manager", Count: 2}},
		errorSignals: []*canonicalEvent{{Index: 1, EventType: "call_hangup", Publisher: "call-manager", Timestamp: "t1", Summary: "down"}},
		transcripts:  []string{"[in] hi"},
	}
	decode := func(b json.RawMessage) map[string]json.RawMessage {
		t.Helper()
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatalf("could not parse the payload. err: %v", err)
		}
		return doc
	}
	asString := func(raw json.RawMessage) string {
		t.Helper()
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatalf("could not parse a string field. err: %v", err)
		}
		return s
	}

	stage3 := decode(h.buildStage3Data(json.RawMessage(`{"s":1}`), json.RawMessage(`{"s":2}`), input, nil))
	if got := asString(stage3["error_signals"]); got != "1\tt1\tcall_hangup\tdown\n" {
		t.Errorf("Wrong match. expect: the single hangup signal line, got: %q", got)
	}
	if _, ok := stage3["context"]; ok {
		t.Errorf("Wrong match. expect: no context block without enrichment, got: present")
	}
}
