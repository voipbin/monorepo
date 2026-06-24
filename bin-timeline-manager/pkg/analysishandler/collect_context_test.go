package analysishandler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"monorepo/bin-timeline-manager/models/verdict"
)

// newEnrichTestHandler builds a handler with the enrichment metric wired so
// enrich()/metricEnrichmentOutcome do not nil-panic.
func newEnrichTestHandler(t *testing.T) (*analysisHandler, *requesthandler.MockRequestHandler, *gomock.Controller) {
	t.Helper()
	mc := gomock.NewController(t)
	reqMock := requesthandler.NewMockRequestHandler(mc)
	h := &analysisHandler{
		utilHandler:      utilhandler.NewUtilHandler(),
		reqHandler:       reqMock,
		models:           StageModels{Stage1: "m1", Stage2: "m2", Stage3: "m3"},
		sem:              make(chan struct{}, analysisMaxConcurrentJobs),
		metricStarted:    promAnalysisStarted,
		metricCompleted:  promAnalysisCompleted,
		metricDuration:   promAnalysisDuration,
		metricEnrichment: promAnalysisEnrichment,
	}
	return h, reqMock, mc
}

func ptrTime(s string) *time.Time {
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &tm
}

func af(refType fmactiveflow.ReferenceType, refID, flowID, customerID uuid.UUID) *fmactiveflow.Activeflow {
	a := &fmactiveflow.Activeflow{}
	a.Identity = commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID}
	a.ReferenceType = refType
	a.ReferenceID = refID
	a.FlowID = flowID
	a.Status = fmactiveflow.StatusEnded
	return a
}

var (
	tCust = uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	tRef  = uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	tFlow = uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	tOrig = uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
)

// --- channelOf / normalizeDir / mapCallResult ---

func Test_channelOf_total(t *testing.T) {
	cases := map[fmactiveflow.ReferenceType]string{
		fmactiveflow.ReferenceTypeCall:         "voice",
		fmactiveflow.ReferenceTypeConversation: "chat",
		fmactiveflow.ReferenceTypeAI:           "ai",
		fmactiveflow.ReferenceTypeAPI:          "api",
		fmactiveflow.ReferenceTypeCampaign:     "voice",
		fmactiveflow.ReferenceType("confbridge"): "voice",
		fmactiveflow.ReferenceTypeNone:         "",
		fmactiveflow.ReferenceTypeTranscribe:   "",
		fmactiveflow.ReferenceTypeRecording:    "",
	}
	for rt, want := range cases {
		if got := channelOf(rt); got != want {
			t.Errorf("channelOf(%q) = %q, want %q", rt, got, want)
		}
	}
}

func Test_normalizeDir(t *testing.T) {
	if normalizeDir("incoming") != "inbound" {
		t.Error("incoming should map to inbound")
	}
	if normalizeDir("outgoing") != "outbound" {
		t.Error("outgoing should map to outbound")
	}
	if normalizeDir("") != "" {
		t.Error("empty should stay empty")
	}
}

func Test_mapCallResult_table(t *testing.T) {
	cases := []struct {
		reason cmcall.HangupReason
		status cmcall.Status
		want   string
	}{
		{cmcall.HangupReasonNormal, cmcall.StatusHangup, "completed"},
		{cmcall.HangupReasonNoanswer, "", "no_answer"},
		{cmcall.HangupReasonDialout, "", "no_answer"},
		{cmcall.HangupReasonBusy, "", "busy"},
		{cmcall.HangupReasonFailed, "", "failed"},
		{cmcall.HangupReasonCanceled, "", "failed"},
		{cmcall.HangupReasonTimeout, "", "failed"},
		{cmcall.HangupReasonAMD, "", "failed"},
		{cmcall.HangupReasonNone, cmcall.StatusHangup, "completed"},
		{cmcall.HangupReasonNone, cmcall.StatusProgressing, "in_progress"},
		{cmcall.HangupReason("weird"), "", "unknown"},
	}
	for _, c := range cases {
		if got := mapCallResult(c.reason, c.status); got != c.want {
			t.Errorf("mapCallResult(%q,%q) = %q, want %q", c.reason, c.status, got, c.want)
		}
	}
}

// --- enrichCall ---

func mkCall(customerID uuid.UUID, dir cmcall.Direction, reason cmcall.HangupReason, by cmcall.HangupBy, status cmcall.Status) cmcall.Call {
	c := cmcall.Call{}
	c.Identity = commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID}
	c.Direction = dir
	c.HangupReason = reason
	c.HangupBy = by
	c.Status = status
	c.Source = commonaddress.Address{Target: "+1111"}
	c.Destination = commonaddress.Address{Target: "+2222"}
	c.TMProgressing = ptrTime("2026-06-24T10:00:00Z")
	c.TMHangup = ptrTime("2026-06-24T10:00:32Z")
	c.TMCreate = ptrTime("2026-06-24T09:59:58Z")
	return c
}

func Test_enrich_call_resolved_inbound_customer_ended(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	c := mkCall(tCust, cmcall.DirectionIncoming, cmcall.HangupReasonNormal, cmcall.HangupByRemote, cmcall.StatusHangup)
	reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{c}, nil)
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "support-flow"}, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeCall, tRef, tFlow, tCust))
	if got.ctx == nil || got.outcome == nil {
		t.Fatalf("expected resolved ctx+outcome, got %+v", got)
	}
	if got.ctx.Channel != "voice" || got.ctx.Direction != "inbound" || got.ctx.DirectionRaw != "incoming" {
		t.Errorf("bad ctx: %+v", got.ctx)
	}
	if got.ctx.FlowName != "support-flow" {
		t.Errorf("flow name not scoped in: %q", got.ctx.FlowName)
	}
	if len(got.ctx.Participants) != 2 || got.ctx.Participants[0].Role != "source" {
		t.Errorf("bad participants: %+v", got.ctx.Participants)
	}
	if got.outcome.Result != "completed" || got.outcome.EndedBy != "remote" {
		t.Errorf("bad outcome: %+v", got.outcome)
	}
	if got.outcome.Detail["duration_sec"] != "32" {
		t.Errorf("duration_sec wrong: %q", got.outcome.Detail["duration_sec"])
	}
}

func Test_enrich_call_customer_mismatch_no_leak(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	other := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")
	c := mkCall(other, cmcall.DirectionIncoming, cmcall.HangupReasonNormal, cmcall.HangupByRemote, cmcall.StatusHangup)
	reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{c}, nil)
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "f"}, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeCall, tRef, tFlow, tCust))
	// ctx still present (header) but outcome must be nil — foreign call body never leaks.
	if got.outcome != nil {
		t.Fatalf("expected nil outcome on customer mismatch, got %+v", got.outcome)
	}
}

func Test_enrich_call_multi_leg_flag(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	c1 := mkCall(tCust, cmcall.DirectionIncoming, cmcall.HangupReasonNormal, cmcall.HangupByRemote, cmcall.StatusHangup)
	c1.TMCreate = ptrTime("2026-06-24T09:59:58Z")
	c2 := mkCall(tCust, cmcall.DirectionOutgoing, cmcall.HangupReasonNormal, cmcall.HangupByLocal, cmcall.StatusHangup)
	c2.TMCreate = ptrTime("2026-06-24T10:00:05Z")
	reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{c2, c1}, nil)
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "f"}, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeCall, tRef, tFlow, tCust))
	if got.ctx == nil || !got.ctx.MultiLeg {
		t.Fatalf("expected MultiLeg=true, got %+v", got.ctx)
	}
	// primary must be the EARLIEST leg (c1, inbound).
	if got.ctx.Direction != "inbound" {
		t.Errorf("expected earliest leg as primary (inbound), got %q", got.ctx.Direction)
	}
}

func Test_enrich_call_unanswered_no_answer(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	c := mkCall(tCust, cmcall.DirectionOutgoing, cmcall.HangupReasonNoanswer, cmcall.HangupByRemote, cmcall.StatusHangup)
	c.TMProgressing = nil // never answered
	reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{c}, nil)
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "f"}, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeCall, tRef, tFlow, tCust))
	if got.outcome.Result != "no_answer" {
		t.Errorf("expected no_answer, got %q", got.outcome.Result)
	}
	if _, ok := got.outcome.Detail["duration_sec"]; ok {
		t.Errorf("duration_sec should be absent for unanswered call")
	}
}

// --- enrichConversation ---

func mkMsg(customerID uuid.UUID, dir cvmessage.Direction, status cvmessage.Status, tmCreate string) cvmessage.Message {
	m := cvmessage.Message{}
	m.Identity = commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: customerID}
	m.Direction = dir
	m.Status = status
	m.TMCreate = ptrTime(tmCreate)
	return m
}

func Test_enrich_conversation_unanswered_last_inbound(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	cv := &cvconversation.Conversation{
		Type: cvconversation.TypeLine,
		Self: commonaddress.Address{Target: "biz"},
		Peer: commonaddress.Address{Target: "user"},
	}
	cv.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
	cv.TMCreate = ptrTime("2026-06-24T09:00:00Z")
	reqMock.EXPECT().ConversationV1ConversationGet(gomock.Any(), tRef).Return(cv, nil)
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "chatflow"}, nil)

	// MessageList returns DESC; provider must sort ASC. Last (newest) is incoming -> unanswered.
	msgs := []cvmessage.Message{
		mkMsg(tCust, cvmessage.DirectionIncoming, cvmessage.StatusProgressing, "2026-06-24T09:05:00Z"), // newest
		mkMsg(tCust, cvmessage.DirectionOutgoing, cvmessage.StatusDone, "2026-06-24T09:02:00Z"),
		mkMsg(tCust, cvmessage.DirectionIncoming, cvmessage.StatusDone, "2026-06-24T09:00:30Z"), // oldest
	}
	reqMock.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1000), gomock.Any()).Return(msgs, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeConversation, tRef, tFlow, tCust))
	if got.ctx == nil || got.ctx.Channel != "chat" {
		t.Fatalf("bad ctx: %+v", got.ctx)
	}
	if got.ctx.Direction != "" {
		t.Errorf("thread-level direction must stay empty in P1, got %q", got.ctx.Direction)
	}
	if got.metrics != nil {
		t.Errorf("conversation must NOT get a SessionMetrics block, got %+v", got.metrics)
	}
	d := got.outcome.Detail
	if d["unanswered"] != "true" || d["last_activity_by"] != "peer" {
		t.Errorf("unanswered/last_activity_by wrong: %+v", d)
	}
	if d["turns_self"] != "1" || d["turns_peer"] != "2" {
		t.Errorf("turn counts wrong: self=%s peer=%s", d["turns_self"], d["turns_peer"])
	}
	if d["chat_platform"] != "line" {
		t.Errorf("chat_platform wrong: %q", d["chat_platform"])
	}
	if got.outcome.Result != "in_progress" {
		t.Errorf("last message progressing -> in_progress, got %q", got.outcome.Result)
	}
}

func Test_enrich_conversation_one_failed_not_thread_failed(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	cv := &cvconversation.Conversation{Type: cvconversation.TypeMessage}
	cv.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
	cv.TMCreate = ptrTime("2026-06-24T09:00:00Z")
	reqMock.EXPECT().ConversationV1ConversationGet(gomock.Any(), tRef).Return(cv, nil)
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "f"}, nil)

	// one failed in the middle, last is done -> Result completed, delivery_failures=1.
	msgs := []cvmessage.Message{
		mkMsg(tCust, cvmessage.DirectionOutgoing, cvmessage.StatusDone, "2026-06-24T09:05:00Z"),
		mkMsg(tCust, cvmessage.DirectionOutgoing, cvmessage.StatusFailed, "2026-06-24T09:02:00Z"),
		mkMsg(tCust, cvmessage.DirectionIncoming, cvmessage.StatusDone, "2026-06-24T09:00:30Z"),
	}
	reqMock.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1000), gomock.Any()).Return(msgs, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeConversation, tRef, tFlow, tCust))
	if got.outcome.Result != "completed" {
		t.Errorf("last message done -> completed, got %q", got.outcome.Result)
	}
	if got.outcome.Detail["delivery_failures"] != "1" {
		t.Errorf("delivery_failures should be 1, got %q", got.outcome.Detail["delivery_failures"])
	}
}

func Test_enrich_conversation_zero_messages(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	cv := &cvconversation.Conversation{Type: cvconversation.TypeMessage}
	cv.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
	cv.TMCreate = ptrTime("2026-06-24T09:00:00Z")
	reqMock.EXPECT().ConversationV1ConversationGet(gomock.Any(), tRef).Return(cv, nil)
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "f"}, nil)
	reqMock.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1000), gomock.Any()).Return([]cvmessage.Message{}, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeConversation, tRef, tFlow, tCust))
	if got.outcome.Result != "in_progress" {
		t.Errorf("zero messages -> in_progress, got %q", got.outcome.Result)
	}
}

// --- origin chase (transcribe/recording) ---

func Test_enrich_transcribe_chases_call_origin(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	// transcribe references a call origin.
	tr := &tmtranscribe.Transcribe{ReferenceType: tmtranscribe.ReferenceTypeCall, ReferenceID: tOrig}
	tr.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
	reqMock.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), tRef).Return(tr, nil)

	c := mkCall(tCust, cmcall.DirectionIncoming, cmcall.HangupReasonNormal, cmcall.HangupByRemote, cmcall.StatusHangup)
	reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{c}, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeTranscribe, tRef, tFlow, tCust))
	if got.ctx == nil {
		t.Fatal("expected chased ctx")
	}
	// reference_type STAYS transcribe; origin markers stamped; channel from origin.
	if got.ctx.ReferenceType != "transcribe" {
		t.Errorf("reference_type must stay transcribe, got %q", got.ctx.ReferenceType)
	}
	if got.ctx.OriginKind != "transcription" || got.ctx.OriginType != "call" {
		t.Errorf("origin markers wrong: kind=%q type=%q", got.ctx.OriginKind, got.ctx.OriginType)
	}
	if got.ctx.Channel != "voice" {
		t.Errorf("chased channel should be origin's (voice), got %q", got.ctx.Channel)
	}
	// chased card suppresses metrics/flags + FlowName.
	if got.metrics != nil {
		t.Errorf("chased card must suppress metrics, got %+v", got.metrics)
	}
	if got.ctx.AIHandled || got.ctx.HumanInvolved {
		t.Errorf("chased card must suppress AIHandled/HumanInvolved")
	}
	if got.ctx.FlowName != "" {
		t.Errorf("chased card must omit FlowName, got %q", got.ctx.FlowName)
	}
	// origin outcome borrowed.
	if got.outcome == nil || got.outcome.Result != "completed" {
		t.Errorf("expected borrowed origin outcome completed, got %+v", got.outcome)
	}
}

func Test_enrich_transcribe_origin_ownership_mismatch_header_only(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	other := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")
	tr := &tmtranscribe.Transcribe{ReferenceType: tmtranscribe.ReferenceTypeCall, ReferenceID: tOrig}
	tr.Identity = commonidentity.Identity{ID: tRef, CustomerID: other} // foreign transcribe
	reqMock.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), tRef).Return(tr, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeTranscribe, tRef, tFlow, tCust))
	// header-only + marker survives the chase miss; no call lookup happens.
	if got.ctx == nil || got.ctx.OriginKind != "transcription" {
		t.Fatalf("expected header-only with origin marker, got %+v", got.ctx)
	}
	if got.outcome != nil {
		t.Errorf("expected nil outcome on ownership mismatch, got %+v", got.outcome)
	}
}

func Test_enrich_recording_chases_call_origin(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	rec := &cmrecording.Recording{ReferenceType: cmrecording.ReferenceTypeCall, ReferenceID: tOrig}
	rec.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
	reqMock.EXPECT().CallV1RecordingGet(gomock.Any(), tRef).Return(rec, nil)

	c := mkCall(tCust, cmcall.DirectionIncoming, cmcall.HangupReasonNormal, cmcall.HangupByRemote, cmcall.StatusHangup)
	reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{c}, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeRecording, tRef, tFlow, tCust))
	if got.ctx == nil || got.ctx.ReferenceType != "recording" || got.ctx.OriginKind != "recording" {
		t.Fatalf("recording chase markers wrong: %+v", got.ctx)
	}
}

func Test_enrich_recording_origin_confbridge_header_only(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	// recording of a confbridge: chase enrichRef(confbridge) -> header-only, no call list.
	rec := &cmrecording.Recording{ReferenceType: cmrecording.ReferenceTypeConfbridge, ReferenceID: tOrig}
	rec.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
	reqMock.EXPECT().CallV1RecordingGet(gomock.Any(), tRef).Return(rec, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeRecording, tRef, tFlow, tCust))
	if got.ctx == nil || got.ctx.OriginKind != "recording" {
		t.Fatalf("expected recording marker, got %+v", got.ctx)
	}
	if got.ctx.OriginType != "confbridge" || got.ctx.Channel != "voice" {
		t.Errorf("confbridge origin: type=%q channel=%q", got.ctx.OriginType, got.ctx.Channel)
	}
	if got.outcome != nil {
		t.Errorf("confbridge origin has no P1 outcome, got %+v", got.outcome)
	}
}

// --- ai / api / none / campaign ---

func Test_enrich_ai_best_effort(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	ac := &amaicall.AIcall{Status: amaicall.StatusTerminated}
	ac.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
	ac.TMCreate = ptrTime("2026-06-24T10:00:00Z")
	reqMock.EXPECT().AIV1AIcallGet(gomock.Any(), tRef).Return(ac, nil)
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "aiflow"}, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeAI, tRef, tFlow, tCust))
	if got.ctx == nil || got.ctx.Channel != "ai" {
		t.Fatalf("bad ai ctx: %+v", got.ctx)
	}
	if got.outcome == nil || got.outcome.Result != "completed" {
		t.Errorf("terminated aicall -> completed, got %+v", got.outcome)
	}
}

func Test_enrich_api_header_only(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "apiflow"}, nil)

	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeAPI, tRef, tFlow, tCust))
	if got.ctx == nil || got.ctx.Channel != "api" {
		t.Fatalf("bad api ctx: %+v", got.ctx)
	}
	if got.outcome != nil || got.metrics != nil {
		t.Errorf("api must have no outcome/metrics: outcome=%+v metrics=%+v", got.outcome, got.metrics)
	}
	if len(got.ctx.Participants) != 0 {
		t.Errorf("api header should have no participants, got %+v", got.ctx.Participants)
	}
}

func Test_enrich_none_header_only(t *testing.T) {
	h, _, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	// flowID nil -> scopedFlowName returns "" with no RPC.
	got := h.enrich(context.Background(), &collectedInput{}, tCust, af(fmactiveflow.ReferenceTypeNone, uuid.Nil, uuid.Nil, tCust))
	if got.ctx == nil || got.ctx.Channel != "" {
		t.Fatalf("none ctx channel should be empty, got %+v", got.ctx)
	}
}

// --- scopedFlowName foreign-flow drop ---

func Test_scopedFlowName_foreign_drop(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	other := uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: other}, Name: "secret"}, nil)

	if got := h.scopedFlowName(context.Background(), tCust, tFlow); got != "" {
		t.Errorf("foreign flow name must be dropped, got %q", got)
	}
}

// --- metrics aggregation over allEvents ---

func Test_aggregateMetrics_excludes_intermediate(t *testing.T) {
	events := []*canonicalEvent{
		{EventType: "pipecatcall_initialized", Timestamp: "2026-06-24T10:00:00.000000Z"},
		{EventType: "message_bot_llm_intermediate", Timestamp: "2026-06-24T10:00:00.500000Z"}, // EXCLUDED
		{EventType: "message_bot_transcription", Timestamp: "2026-06-24T10:00:01.000000Z"},
		{EventType: "message_user_transcription", Timestamp: "2026-06-24T10:00:03.000000Z"},
		{EventType: "message_bot_transcription", Timestamp: "2026-06-24T10:00:04.000000Z"},
	}
	m := aggregateMetrics(events)
	if m == nil {
		t.Fatal("expected metrics")
	}
	if m.TurnsBot != 2 || m.TurnsUser != 1 {
		t.Errorf("turn counts wrong (intermediate must be excluded): bot=%d user=%d", m.TurnsBot, m.TurnsUser)
	}
	if m.FirstResponseMS == nil || *m.FirstResponseMS != 1000 {
		t.Errorf("first response should be 1000ms, got %v", m.FirstResponseMS)
	}
	// user@03 -> bot@04 = 1000ms.
	if m.AvgResponseMS == nil || *m.AvgResponseMS != 1000 {
		t.Errorf("avg response should be 1000ms, got %v", m.AvgResponseMS)
	}
}

func Test_aggregateMetrics_nil_when_no_turns(t *testing.T) {
	events := []*canonicalEvent{
		{EventType: "call_created", Timestamp: "2026-06-24T10:00:00.000000Z"},
	}
	if m := aggregateMetrics(events); m != nil {
		t.Errorf("expected nil metrics when no turns, got %+v", m)
	}
}

func Test_enrich_call_metrics_and_flags_from_allEvents(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	c := mkCall(tCust, cmcall.DirectionIncoming, cmcall.HangupReasonNormal, cmcall.HangupByRemote, cmcall.StatusHangup)
	reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{c}, nil)
	reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "f"}, nil)

	input := &collectedInput{allEvents: []*canonicalEvent{
		{EventType: "pipecatcall_initialized", Publisher: "pipecat-manager", Timestamp: "2026-06-24T10:00:00.000000Z"},
		{EventType: "message_bot_transcription", Publisher: "ai-manager", Timestamp: "2026-06-24T10:00:01.000000Z"},
		{EventType: "agent_status_updated", Publisher: "agent-manager", Timestamp: "2026-06-24T10:00:02.000000Z"},
	}}

	got := h.enrich(context.Background(), input, tCust, af(fmactiveflow.ReferenceTypeCall, tRef, tFlow, tCust))
	if got.metrics == nil || got.metrics.TurnsBot != 1 {
		t.Errorf("expected voice metrics from allEvents, got %+v", got.metrics)
	}
	if !got.ctx.AIHandled {
		t.Errorf("AIHandled should be true (pipecat/ai publisher present)")
	}
	if !got.ctx.HumanInvolved {
		t.Errorf("HumanInvolved should be true (agent-manager publisher present)")
	}
}

// --- serialization: absent blocks omitted, not null ---

func Test_verdict_marshal_omits_absent_v3_blocks(t *testing.T) {
	h := &analysisHandler{}
	raw := &verdict.RawVerdict{OverallStatus: verdict.OverallStatusOK}
	// empty enrichment -> all three blocks nil.
	v := h.buildFinalVerdict(raw, nil, minimalInput(), &sessionEnrichment{})
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, key := range []string{"session_context", "outcome", "metrics"} {
		if containsKey(s, key) {
			t.Errorf("absent %q must be an OMITTED key, found it in: %s", key, s)
		}
	}
}

func Test_verdict_marshal_includes_present_v3_blocks(t *testing.T) {
	h := &analysisHandler{}
	raw := &verdict.RawVerdict{OverallStatus: verdict.OverallStatusOK}
	en := &sessionEnrichment{
		ctx:     &verdict.SessionContext{ReferenceType: "call", Channel: "voice"},
		outcome: &verdict.SessionOutcome{Result: "completed", EndedBy: "remote"},
	}
	v := h.buildFinalVerdict(raw, nil, minimalInput(), en)
	b, _ := json.Marshal(v)
	s := string(b)
	if !containsKey(s, "session_context") || !containsKey(s, "outcome") {
		t.Errorf("present blocks must appear: %s", s)
	}
	if containsKey(s, "metrics") {
		t.Errorf("absent metrics must be omitted: %s", s)
	}
}

func containsKey(s, key string) bool {
	return jsonHasKey(s, `"`+key+`"`)
}

func jsonHasKey(s, quotedKey string) bool {
	return len(s) > 0 && indexOf(s, quotedKey+":") >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
