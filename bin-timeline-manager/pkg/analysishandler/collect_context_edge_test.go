package analysishandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-timeline-manager/models/verdict"
)

var tOther = uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")

func Test_chaseRecord_misses(t *testing.T) {
	injected := errors.New("injected failure")

	tests := []struct {
		name    string
		refType fmactiveflow.ReferenceType
		expect  func(reqMock *requesthandler.MockRequestHandler)
	}{
		{
			name:    "transcribe lookup error",
			refType: fmactiveflow.ReferenceTypeTranscribe,
			expect: func(reqMock *requesthandler.MockRequestHandler) {
				reqMock.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), tRef).Return(nil, injected)
			},
		},
		{
			name:    "transcribe nil record",
			refType: fmactiveflow.ReferenceTypeTranscribe,
			expect: func(reqMock *requesthandler.MockRequestHandler) {
				reqMock.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), tRef).Return(nil, nil)
			},
		},
		{
			name:    "recording lookup error",
			refType: fmactiveflow.ReferenceTypeRecording,
			expect: func(reqMock *requesthandler.MockRequestHandler) {
				reqMock.EXPECT().CallV1RecordingGet(gomock.Any(), tRef).Return(nil, injected)
			},
		},
		{
			name:    "recording nil record",
			refType: fmactiveflow.ReferenceTypeRecording,
			expect: func(reqMock *requesthandler.MockRequestHandler) {
				reqMock.EXPECT().CallV1RecordingGet(gomock.Any(), tRef).Return(nil, nil)
			},
		},
		{
			name:    "recording owned by another customer",
			refType: fmactiveflow.ReferenceTypeRecording,
			expect: func(reqMock *requesthandler.MockRequestHandler) {
				rec := &cmrecording.Recording{ReferenceType: cmrecording.ReferenceTypeCall, ReferenceID: tOrig}
				rec.Identity = commonidentity.Identity{ID: tRef, CustomerID: tOther}
				reqMock.EXPECT().CallV1RecordingGet(gomock.Any(), tRef).Return(rec, nil)
			},
		},
		{
			name:    "unsupported reference type makes no lookup",
			refType: fmactiveflow.ReferenceTypeCall,
			expect:  func(reqMock *requesthandler.MockRequestHandler) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, reqMock, mc := newEnrichTestHandler(t)
			defer mc.Finish()
			tt.expect(reqMock)

			originType, originID, ok := h.chaseRecord(context.Background(), tCust, tt.refType, tRef)
			if ok || originType != "" || originID != uuid.Nil {
				t.Errorf("Wrong match. expect: miss, got: type=%q id=%s ok=%v", originType, originID, ok)
			}
		})
	}
}

func Test_chaseRecord_transcribe_hit(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	tr := &tmtranscribe.Transcribe{ReferenceType: tmtranscribe.ReferenceTypeCall, ReferenceID: tOrig}
	tr.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
	reqMock.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), tRef).Return(tr, nil)

	originType, originID, ok := h.chaseRecord(context.Background(), tCust, fmactiveflow.ReferenceTypeTranscribe, tRef)
	if !ok || originType != "call" || originID != tOrig {
		t.Errorf("Wrong match. expect: call %s, got: type=%q id=%s ok=%v", tOrig, originType, originID, ok)
	}
}

func Test_chaseOrigin_depth_limit(t *testing.T) {
	h, _, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	// no request handler call is expected: beyond the chase depth the card
	// stays header-only and keeps its origin marker.
	sc, outcome, chased := h.chaseOrigin(context.Background(), tCust, fmactiveflow.ReferenceTypeTranscribe, tRef, enrichMaxChaseDepth+1)
	if sc == nil || sc.ReferenceType != "transcribe" || sc.OriginKind != "transcription" {
		t.Errorf("Wrong match. expect: header-only transcribe card with marker, got: %+v", sc)
	}
	if outcome != nil || !chased {
		t.Errorf("Wrong match. expect: nil outcome and chased true, got: %+v %v", outcome, chased)
	}
}

func Test_chaseOrigin_at_depth_limit_still_chases(t *testing.T) {
	h, reqMock, mc := newEnrichTestHandler(t)
	defer mc.Finish()

	// at exactly the depth limit the origin is still resolved (only deeper
	// levels fall back to header-only).
	tr := &tmtranscribe.Transcribe{ReferenceType: tmtranscribe.ReferenceTypeCall, ReferenceID: tOrig}
	tr.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
	reqMock.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), tRef).Return(tr, nil)

	c := mkCall(tCust, cmcall.DirectionIncoming, cmcall.HangupReasonNormal, cmcall.HangupByRemote, cmcall.StatusHangup)
	reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{c}, nil)

	sc, outcome, chased := h.chaseOrigin(context.Background(), tCust, fmactiveflow.ReferenceTypeTranscribe, tRef, enrichMaxChaseDepth)
	if sc == nil || sc.OriginType != "call" || sc.Channel != "voice" {
		t.Errorf("Wrong match. expect: chased card from a call origin, got: %+v", sc)
	}
	if outcome == nil || outcome.Result != "completed" || !chased {
		t.Errorf("Wrong match. expect: completed outcome and chased true, got: %+v %v", outcome, chased)
	}
}

func Test_enrichCall_fallback(t *testing.T) {
	injected := errors.New("injected failure")

	t.Run("list and get both fail", func(t *testing.T) {
		h, reqMock, mc := newEnrichTestHandler(t)
		defer mc.Finish()

		reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return(nil, injected)
		reqMock.EXPECT().CallV1CallGet(gomock.Any(), tRef).Return(nil, injected)

		sc := &verdict.SessionContext{}
		if got := h.enrichCall(context.Background(), tCust, tRef, sc); got != nil {
			t.Errorf("Wrong match. expect: nil outcome, got: %+v", got)
		}
		if len(sc.Participants) != 0 {
			t.Errorf("Wrong match. expect: untouched context, got: %+v", sc)
		}
	})

	t.Run("empty list and nil get", func(t *testing.T) {
		h, reqMock, mc := newEnrichTestHandler(t)
		defer mc.Finish()

		reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{}, nil)
		reqMock.EXPECT().CallV1CallGet(gomock.Any(), tRef).Return(nil, nil)

		if got := h.enrichCall(context.Background(), tCust, tRef, &verdict.SessionContext{}); got != nil {
			t.Errorf("Wrong match. expect: nil outcome, got: %+v", got)
		}
	})

	t.Run("empty list then get succeeds", func(t *testing.T) {
		h, reqMock, mc := newEnrichTestHandler(t)
		defer mc.Finish()

		c := mkCall(tCust, cmcall.DirectionIncoming, cmcall.HangupReasonNormal, cmcall.HangupByRemote, cmcall.StatusHangup)
		reqMock.EXPECT().CallV1CallList(gomock.Any(), "", uint64(100), gomock.Any()).Return([]cmcall.Call{}, nil)
		reqMock.EXPECT().CallV1CallGet(gomock.Any(), tRef).Return(&c, nil)

		sc := &verdict.SessionContext{}
		got := h.enrichCall(context.Background(), tCust, tRef, sc)
		if got == nil || got.Result != "completed" {
			t.Fatalf("Wrong match. expect: completed outcome from the single call, got: %+v", got)
		}
		if sc.MultiLeg {
			t.Errorf("Wrong match. expect: MultiLeg false for a single call, got: true")
		}
		if sc.Direction != "inbound" {
			t.Errorf("Wrong match. expect: inbound, got: %q", sc.Direction)
		}
	})
}

func Test_enrichConversation_misses(t *testing.T) {
	injected := errors.New("injected failure")

	t.Run("lookup error", func(t *testing.T) {
		h, reqMock, mc := newEnrichTestHandler(t)
		defer mc.Finish()

		reqMock.EXPECT().ConversationV1ConversationGet(gomock.Any(), tRef).Return(nil, injected)

		if got := h.enrichConversation(context.Background(), tCust, tRef, &verdict.SessionContext{}); got != nil {
			t.Errorf("Wrong match. expect: nil outcome, got: %+v", got)
		}
	})

	t.Run("owned by another customer", func(t *testing.T) {
		h, reqMock, mc := newEnrichTestHandler(t)
		defer mc.Finish()

		cv := &cvconversation.Conversation{Type: cvconversation.TypeLine}
		cv.Identity = commonidentity.Identity{ID: tRef, CustomerID: tOther}
		reqMock.EXPECT().ConversationV1ConversationGet(gomock.Any(), tRef).Return(cv, nil)

		sc := &verdict.SessionContext{}
		if got := h.enrichConversation(context.Background(), tCust, tRef, sc); got != nil {
			t.Errorf("Wrong match. expect: nil outcome, got: %+v", got)
		}
		if sc.Channel != "" || len(sc.Participants) != 0 {
			t.Errorf("Wrong match. expect: foreign data not copied into the context, got: %+v", sc)
		}
	})

	t.Run("message list error keeps platform detail", func(t *testing.T) {
		h, reqMock, mc := newEnrichTestHandler(t)
		defer mc.Finish()

		cv := &cvconversation.Conversation{Type: cvconversation.TypeLine}
		cv.Identity = commonidentity.Identity{ID: tRef, CustomerID: tCust}
		reqMock.EXPECT().ConversationV1ConversationGet(gomock.Any(), tRef).Return(cv, nil)
		reqMock.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1000), gomock.Any()).Return(nil, injected)

		got := h.enrichConversation(context.Background(), tCust, tRef, &verdict.SessionContext{})
		if got == nil || got.Result != "in_progress" {
			t.Fatalf("Wrong match. expect: in_progress outcome, got: %+v", got)
		}
		if got.Detail["chat_platform"] != "line" {
			t.Errorf("Wrong match. expect: chat_platform line, got: %v", got.Detail)
		}
	})
}

func Test_enrichAIcall_misses(t *testing.T) {
	injected := errors.New("injected failure")

	tests := []struct {
		name   string
		result *amaicall.AIcall
		err    error
	}{
		{"lookup error", nil, injected},
		{"nil record", nil, nil},
		{
			name: "owned by another customer",
			result: func() *amaicall.AIcall {
				ac := &amaicall.AIcall{Status: amaicall.StatusTerminated}
				ac.Identity = commonidentity.Identity{ID: tRef, CustomerID: tOther}
				return ac
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, reqMock, mc := newEnrichTestHandler(t)
			defer mc.Finish()

			reqMock.EXPECT().AIV1AIcallGet(gomock.Any(), tRef).Return(tt.result, tt.err)

			sc := &verdict.SessionContext{}
			if got := h.enrichAIcall(context.Background(), tCust, tRef, sc); got != nil {
				t.Errorf("Wrong match. expect: nil outcome, got: %+v", got)
			}
			if sc.StartedAt != "" {
				t.Errorf("Wrong match. expect: untouched context, got: %+v", sc)
			}
		})
	}
}

func Test_scopedFlowName_misses(t *testing.T) {
	t.Run("lookup error", func(t *testing.T) {
		h, reqMock, mc := newEnrichTestHandler(t)
		defer mc.Finish()

		reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(nil, errors.New("injected failure"))

		if got := h.scopedFlowName(context.Background(), tCust, tFlow); got != "" {
			t.Errorf("Wrong match. expect: empty, got: %q", got)
		}
	})

	t.Run("nil flow", func(t *testing.T) {
		h, reqMock, mc := newEnrichTestHandler(t)
		defer mc.Finish()

		reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(nil, nil)

		if got := h.scopedFlowName(context.Background(), tCust, tFlow); got != "" {
			t.Errorf("Wrong match. expect: empty, got: %q", got)
		}
	})

	t.Run("own flow returns its name", func(t *testing.T) {
		h, reqMock, mc := newEnrichTestHandler(t)
		defer mc.Finish()

		reqMock.EXPECT().FlowV1FlowGet(gomock.Any(), tFlow).Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: tFlow, CustomerID: tCust}, Name: "own-flow"}, nil)

		if got := h.scopedFlowName(context.Background(), tCust, tFlow); got != "own-flow" {
			t.Errorf("Wrong match. expect: own-flow, got: %q", got)
		}
	})
}

func Test_markerFor(t *testing.T) {
	tests := []struct {
		refType fmactiveflow.ReferenceType
		expect  string
	}{
		{fmactiveflow.ReferenceTypeTranscribe, "transcription"},
		{fmactiveflow.ReferenceTypeRecording, "recording"},
		{fmactiveflow.ReferenceTypeCall, ""},
		{fmactiveflow.ReferenceTypeNone, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.refType), func(t *testing.T) {
			if got := markerFor(tt.refType); got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_parseCanonicalTS(t *testing.T) {
	tests := []struct {
		name string
		in   string

		expectOK   bool
		expectUnix int64
	}{
		{"valid", "2026-06-24T10:00:00.000000Z", true, 1782295200},
		{"empty", "", false, 0},
		{"rfc3339 without fraction", "2026-06-24T10:00:00Z", false, 0},
		{"garbage", "not a time", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCanonicalTS(tt.in)
			if ok != tt.expectOK {
				t.Fatalf("Wrong match. expect: ok=%v, got: %v", tt.expectOK, ok)
			}
			if ok && got.Unix() != tt.expectUnix {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectUnix, got.Unix())
			}
			if !ok && !got.IsZero() {
				t.Errorf("Wrong match. expect: zero time, got: %v", got)
			}
		})
	}
}

func Test_clamp0MS(t *testing.T) {
	tests := []struct {
		name   string
		in     time.Duration
		expect int
	}{
		{"negative skew is clamped", -250 * time.Millisecond, 0},
		{"zero", 0, 0},
		{"positive", 1500 * time.Millisecond, 1500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clamp0MS(tt.in); got != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expect, got)
			}
		})
	}
}

func Test_callDurationSec(t *testing.T) {
	tests := []struct {
		name        string
		progressing *time.Time
		hangup      *time.Time
		expect      int
	}{
		{"not answered", nil, ptrTime("2026-06-24T10:00:32Z"), -1},
		{"still up", ptrTime("2026-06-24T10:00:00Z"), nil, -1},
		{"normal", ptrTime("2026-06-24T10:00:00Z"), ptrTime("2026-06-24T10:00:32Z"), 32},
		{"hangup before progressing is clamped", ptrTime("2026-06-24T10:00:32Z"), ptrTime("2026-06-24T10:00:00Z"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cmcall.Call{TMProgressing: tt.progressing, TMHangup: tt.hangup}
			if got := callDurationSec(c); got != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expect, got)
			}
		})
	}
}

func Test_threadSpanSec(t *testing.T) {
	tests := []struct {
		name        string
		first, last *time.Time
		expect      int
	}{
		{"missing first", nil, ptrTime("2026-06-24T10:00:10Z"), 0},
		{"missing last", ptrTime("2026-06-24T10:00:00Z"), nil, 0},
		{"normal", ptrTime("2026-06-24T10:00:00Z"), ptrTime("2026-06-24T10:01:30Z"), 90},
		{"reversed is clamped", ptrTime("2026-06-24T10:01:30Z"), ptrTime("2026-06-24T10:00:00Z"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := threadSpanSec(tt.first, tt.last); got != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expect, got)
			}
		})
	}
}

func Test_tmBefore(t *testing.T) {
	early := ptrTime("2026-06-24T10:00:00Z")
	late := ptrTime("2026-06-24T10:00:05Z")

	tests := []struct {
		name   string
		a, b   *time.Time
		expect bool
	}{
		{"nil first is never before", nil, late, false},
		{"nil second means first is before", early, nil, true},
		{"both nil", nil, nil, false},
		{"earlier is before", early, late, true},
		{"later is not before", late, early, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tmBefore(tt.a, tt.b); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_rfc3339OrEmpty(t *testing.T) {
	if got := rfc3339OrEmpty(nil); got != "" {
		t.Errorf("Wrong match. expect: empty for nil time, got: %q", got)
	}
	if got := rfc3339OrEmpty(ptrTime("2026-06-24T10:00:00Z")); got != "2026-06-24T10:00:00Z" {
		t.Errorf("Wrong match. expect: 2026-06-24T10:00:00Z, got: %q", got)
	}
}

func Test_detailOrNil(t *testing.T) {
	if got := detailOrNil(map[string]string{}); got != nil {
		t.Errorf("Wrong match. expect: nil for an empty map, got: %v", got)
	}
	if got := detailOrNil(nil); got != nil {
		t.Errorf("Wrong match. expect: nil for a nil map, got: %v", got)
	}
	if got := detailOrNil(map[string]string{"k": "v"}); got["k"] != "v" {
		t.Errorf("Wrong match. expect: the map back, got: %v", got)
	}
}

func Test_boolStr(t *testing.T) {
	if got := boolStr(true); got != "true" {
		t.Errorf("Wrong match. expect: true, got: %q", got)
	}
	if got := boolStr(false); got != "false" {
		t.Errorf("Wrong match. expect: false, got: %q", got)
	}
}
