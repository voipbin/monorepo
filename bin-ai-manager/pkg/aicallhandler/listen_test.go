package aicallhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"
)

// Test_isListenableCallStatus pins the exact status set transcribe-manager's
// own isValidReference accepts for a call reference. Diverging in either
// direction is a real defect: a status this returns true for but
// transcribe-manager rejects means a listen-start that always fails, and a
// status it rejects that transcribe-manager would accept means listening
// silently never starts on a perfectly valid call.
func Test_isListenableCallStatus(t *testing.T) {
	tests := []struct {
		name   string
		status cmcall.Status
		expect bool
	}{
		{"dialing is listenable", cmcall.StatusDialing, true},
		{"ringing is listenable", cmcall.StatusRinging, true},
		{"progressing is listenable", cmcall.StatusProgressing, true},
		{"hangup is not", cmcall.StatusHangup, false},
		{"terminating is not", cmcall.StatusTerminating, false},
		{"canceling is not", cmcall.StatusCanceling, false},
		{"empty is not", cmcall.Status(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isListenableCallStatus(tt.status); got != tt.expect {
				t.Errorf("isListenableCallStatus(%q) mismatch. expected: %v, got: %v", tt.status, tt.expect, got)
			}
		})
	}
}

// Test_listenTranscribeIDFromMetadata pins the metadata reader's tolerance.
// Metadata round-trips through JSON, so a value written as a uuid.UUID comes
// back as a string -- and anything unexpected must read as uuid.Nil rather
// than panic, since this runs on the AIcall of every listen precondition check.
func Test_listenTranscribeIDFromMetadata(t *testing.T) {
	valid := uuid.FromStringOrNil("aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee")

	tests := []struct {
		name   string
		c      *aicall.AIcall
		expect uuid.UUID
	}{
		{"nil metadata", &aicall.AIcall{}, uuid.Nil},
		{"absent key", &aicall.AIcall{Metadata: map[string]any{"other": "x"}}, uuid.Nil},
		{"wrong type", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenTranscribeID: 42}}, uuid.Nil},
		{"unparseable string", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenTranscribeID: "not-a-uuid"}}, uuid.Nil},
		{"well-formed value", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenTranscribeID: valid.String()}}, valid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenTranscribeIDFromMetadata(tt.c); got != tt.expect {
				t.Errorf("mismatch. expected: %s, got: %s", tt.expect, got)
			}
		})
	}
}

// Test_listenOwnsTranscribeFromMetadata pins the ownership reader. Reading a
// missing or wrong-typed value as TRUE would let a non-owner stop a transcribe
// session another Case is still listening to, so the default must be false.
func Test_listenOwnsTranscribeFromMetadata(t *testing.T) {
	tests := []struct {
		name   string
		c      *aicall.AIcall
		expect bool
	}{
		{"nil metadata", &aicall.AIcall{}, false},
		{"absent key", &aicall.AIcall{Metadata: map[string]any{"other": "x"}}, false},
		{"wrong type", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenOwnsTranscribe: "true"}}, false},
		{"false", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenOwnsTranscribe: false}}, false},
		{"true", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenOwnsTranscribe: true}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenOwnsTranscribeFromMetadata(tt.c); got != tt.expect {
				t.Errorf("mismatch. expected: %v, got: %v", tt.expect, got)
			}
		})
	}
}

// Test_buildListenTurnMessages is a golden test on the exact shape of a listen
// turn's context.
//
// It asserts message COUNT and ORDER, not just presence, because the failure
// mode this guards is silent: an Insight AI missing InsightSystemPrompt still
// answers, it just answers without the platform's hallucination and
// tool-leakage guardrails -- and unsolicited output is exactly where those
// matter most.
//
// It also asserts getPipecatcallMessages is NOT called and c.PipecatcallID is
// NOT written. Both are load-bearing: reading the replay window would reintroduce
// the context-eviction problem, and writing the pipecatcall id would rotate the
// agent's own conversational turn out from under an in-flight answer.
func Test_buildListenTurnMessages(t *testing.T) {
	config.SetListenDefaultsForTest()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockMessage := messagehandler.NewMockMessageHandler(mc)
	h := &aicallHandler{messageHandler: mockMessage}
	ctx := context.Background()

	aicallID := uuid.FromStringOrNil("5f6a7b8c-9d0e-4f1a-8b2c-3d4e5f6a7b8c")
	c := &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: aicallID},
		AssistanceType: aicall.AssistanceTypeAI,
		Metadata: map[string]any{
			aicall.MetaKeyPromptSnapshots: []any{
				map[string]any{"prompt": "CUSTOMER INIT PROMPT"},
			},
		},
	}

	// Q&A rows come back newest-first and are reversed to oldest-first. Tool and
	// system rows are filtered out in-process: ApplyFields has no IN support, so
	// the role filter cannot be expressed in the query.
	//
	// Exactly ONE List call, with exactly this filter map: no getPipecatcallMessages
	// two-fetch pair may appear here (gomock fails on an unexpected call).
	mockMessage.EXPECT().List(ctx, uint64(30), "", map[message.Field]any{
		message.FieldAIcallID: aicallID,
		message.FieldDeleted:  false,
	}).Return([]*message.Message{
		{Role: message.RoleAssistant, Content: "A1"},
		{Role: message.RoleTool, Content: `{"result":"noise"}`},
		{Role: message.RoleUser, Content: "Q1"},
		{Role: message.RoleSystem, Content: "system noise"},
	}, nil).Times(1)

	// The window ALREADY CONTAINS the new lines as its tail -- intake pushes
	// every segment to both the pending buffer and the rolling window
	// (EventTMTranscriptCreated), so by the time a turn drains pending, those
	// same lines are the window's most recent entries. buildListenTranscriptBlock
	// relies on exactly that to split "already seen" from "new", so a fixture
	// whose window omitted the new line would be testing a state that cannot
	// occur in production.
	window := []string{"[CUSTOMER] hello", "[AGENT] hi there", "[CUSTOMER] I want to cancel"}
	newLines := []string{"[CUSTOMER] I want to cancel"}

	res, err := h.buildListenTurnMessages(ctx, c, window, newLines)
	if err != nil {
		t.Fatalf("buildListenTurnMessages returned an unexpected error. err: %v", err)
	}

	if len(res) != 6 {
		t.Fatalf("message count mismatch. expected: 6, got: %d (%v)", len(res), res)
	}

	if res[0]["role"] != "system" || res[0]["content"] != InsightSystemPrompt {
		t.Errorf("message 1 must be InsightSystemPrompt. got: %v", res[0])
	}
	if res[1]["role"] != "system" || res[1]["content"] != "CUSTOMER INIT PROMPT" {
		t.Errorf("message 2 must be the frozen prompt snapshot. got: %v", res[1])
	}
	if res[2]["role"] != "system" || res[2]["content"] != ListenTurnSystemPrompt {
		t.Errorf("message 3 must be ListenTurnSystemPrompt. got: %v", res[2])
	}
	if res[3]["role"] != "user" || res[3]["content"] != "Q1" {
		t.Errorf("message 4 must be the oldest Q&A row. got: %v", res[3])
	}
	if res[4]["role"] != "assistant" || res[4]["content"] != "A1" {
		t.Errorf("message 5 must be the newest Q&A row. got: %v", res[4])
	}
	if res[5]["role"] != "user" {
		t.Errorf("message 6 must be the transcript block. got: %v", res[5])
	}

	transcript, _ := res[5]["content"].(string)
	for _, want := range []string{"[CUSTOMER] hello", "[AGENT] hi there", listenTranscriptNewMarker, "[CUSTOMER] I want to cancel"} {
		if !strings.Contains(transcript, want) {
			t.Errorf("transcript block is missing %q. got:\n%s", want, transcript)
		}
	}

	// The new-lines marker must sit between the seen lines and the new ones --
	// without it the model cannot tell what it has already evaluated and will
	// re-notify about the same thing every turn.
	if strings.Index(transcript, "[AGENT] hi there") > strings.Index(transcript, listenTranscriptNewMarker) {
		t.Errorf("the new-lines marker must come after the already-seen window lines. got:\n%s", transcript)
	}
	if strings.Index(transcript, "[CUSTOMER] I want to cancel") < strings.Index(transcript, listenTranscriptNewMarker) {
		t.Errorf("the new lines must come after the marker. got:\n%s", transcript)
	}

	// The turn must never write the AIcall's bound pipecatcall id.
	if c.PipecatcallID != uuid.Nil {
		t.Errorf("buildListenTurnMessages must not touch c.PipecatcallID")
	}
}

// Test_listenPromptSnapshot pins the team-member selection and its fallback.
func Test_listenPromptSnapshot(t *testing.T) {
	memberA := uuid.FromStringOrNil("8a8a8a8a-0000-4000-8000-000000000001")
	memberB := uuid.FromStringOrNil("8a8a8a8a-0000-4000-8000-000000000002")

	snapshots := []any{
		map[string]any{"prompt": "PROMPT A", "member_id": memberA.String()},
		map[string]any{"prompt": "PROMPT B", "member_id": memberB.String()},
	}

	tests := []struct {
		name   string
		c      *aicall.AIcall
		expect string
	}{
		{"nil metadata", &aicall.AIcall{}, ""},
		{"absent key", &aicall.AIcall{Metadata: map[string]any{"other": 1}}, ""},
		{"empty slice", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyPromptSnapshots: []any{}}}, ""},
		{
			"single-AI call takes the only snapshot",
			&aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyPromptSnapshots: []any{map[string]any{"prompt": "ONLY"}}}},
			"ONLY",
		},
		{
			"team call matches CurrentMemberID",
			&aicall.AIcall{CurrentMemberID: memberB, Metadata: map[string]any{aicall.MetaKeyPromptSnapshots: snapshots}},
			"PROMPT B",
		},
		{
			// A listen turn with the wrong team member's prompt is still far
			// better than one with no customer instructions at all.
			"team call with an unmatched CurrentMemberID falls back to the first",
			&aicall.AIcall{
				CurrentMemberID: uuid.FromStringOrNil("8a8a8a8a-0000-4000-8000-00000000000f"),
				Metadata:        map[string]any{aicall.MetaKeyPromptSnapshots: snapshots},
			},
			"PROMPT A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenPromptSnapshot(tt.c); got != tt.expect {
				t.Errorf("mismatch. expected: %q, got: %q", tt.expect, got)
			}
		})
	}
}

// listenTurnHarness bundles the mocks RunListenTurn and its body need.
type listenTurnHarness struct {
	req   *requesthandler.MockRequestHandler
	db    *dbhandler.MockDBHandler
	cache *cachehandler.MockCacheHandler
	util  *utilhandler.MockUtilHandler
	msg   *messagehandler.MockMessageHandler
	h     *aicallHandler
}

func newListenTurnHarness(mc *gomock.Controller) *listenTurnHarness {
	req := requesthandler.NewMockRequestHandler(mc)
	db := dbhandler.NewMockDBHandler(mc)
	cache := cachehandler.NewMockCacheHandler(mc)
	util := utilhandler.NewMockUtilHandler(mc)
	msg := messagehandler.NewMockMessageHandler(mc)

	return &listenTurnHarness{
		req: req, db: db, cache: cache, util: util, msg: msg,
		h: &aicallHandler{
			reqHandler:     req,
			db:             db,
			cache:          cache,
			utilHandler:    util,
			messageHandler: msg,
		},
	}
}

// listeningAIcall is an AIcall that passes every RunListenTurn precondition.
func listeningAIcall() *aicall.AIcall {
	return &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: ltAIcallID, CustomerID: ltCustomerID},
		AssistanceType: aicall.AssistanceTypeAI,
		ReferenceType:  aicall.ReferenceTypeContactCase,
		Status:         aicall.StatusProgressing,
		PipecatcallID:  uuid.FromStringOrNil("99990000-0000-4000-8000-0000000000aa"),
		Metadata: map[string]any{
			aicall.MetaKeyListenTranscribeID: uuid.FromStringOrNil("99990000-0000-4000-8000-0000000000bb").String(),
		},
	}
}

// metricDelta reports how much a promListenTurnTotal (kind, label) cell moved across fn.
func metricDelta(t *testing.T, kind string, label string, fn func()) float64 {
	t.Helper()
	before := testutil.ToFloat64(promListenTurnTotal.WithLabelValues(kind, label))
	fn()
	return testutil.ToFloat64(promListenTurnTotal.WithLabelValues(kind, label)) - before
}

// Test_RunListenTurn covers every precondition and outcome.
//
// NOTE: not parallel, and its sub-tests are sequential by default -- the
// promListenTurnTotal snapshot deltas below would be corrupted by concurrent
// increments (see metrics_conversation.go's own note on this pattern).
func Test_RunListenTurn(t *testing.T) {
	turnPCID := uuid.FromStringOrNil("99990000-0000-4000-8000-0000000000cc")

	tests := []struct {
		name string

		flagEnabled bool
		status      aicall.Status
		refType     aicall.ReferenceType
		metadata    map[string]any

		expectCountIncr bool
		turnCount       int64

		expectDrain  bool
		pendingLines []string

		expectWindow bool
		registerErr  error

		expectPipecatStart  bool
		expectStopListening bool
		expectResult        string
		expectKind          string
	}{
		{
			name: "flag off stops listening entirely",
			// Not merely "clears bookkeeping": a bare state clear would leave a
			// still-running owned STT session with its handle lost, so a
			// rollback would strand a billed stream until the call ended.
			flagEnabled:         false,
			status:              aicall.StatusProgressing,
			refType:             aicall.ReferenceTypeContactCase,
			expectStopListening: true,
			expectResult:        "skipped_disabled",
		},
		{
			name:                "terminated aicall stops listening",
			flagEnabled:         true,
			status:              aicall.StatusTerminated,
			refType:             aicall.ReferenceTypeContactCase,
			expectStopListening: true,
			expectResult:        "skipped_invalid",
		},
		{
			name:                "non contact_case reference type stops listening",
			flagEnabled:         true,
			status:              aicall.StatusProgressing,
			refType:             aicall.ReferenceTypeCall,
			expectStopListening: true,
			expectResult:        "skipped_invalid",
		},
		{
			name:                "missing listen_transcribe_id metadata stops listening",
			flagEnabled:         true,
			status:              aicall.StatusProgressing,
			refType:             aicall.ReferenceTypeContactCase,
			metadata:            map[string]any{},
			expectStopListening: true,
			expectResult:        "skipped_invalid",
			expectKind:          "unknown",
		},
		{
			name:            "empty pending buffer skips without stopping",
			flagEnabled:     true,
			status:          aicall.StatusProgressing,
			refType:         aicall.ReferenceTypeContactCase,
			expectCountIncr: true,
			turnCount:       1,
			expectDrain:     true,
			pendingLines:    []string{},
			expectResult:    "skipped_empty",
		},
		{
			name:                "turn cap exceeded stops listening",
			flagEnabled:         true,
			status:              aicall.StatusProgressing,
			refType:             aicall.ReferenceTypeContactCase,
			expectCountIncr:     true,
			turnCount:           61,
			expectStopListening: true,
			expectResult:        "skipped_cap",
		},
		{
			name: "turn-id registration failure aborts before starting a pipecatcall",
			// Proceeding unregistered would make every tool call from this turn
			// resolve listenTurn=false: its rows get permanently tagged
			// OriginNone and its notify_agent call gets rejected -- the exact
			// failure the registration exists to prevent.
			flagEnabled:        true,
			status:             aicall.StatusProgressing,
			refType:            aicall.ReferenceTypeContactCase,
			expectCountIncr:    true,
			turnCount:          1,
			expectDrain:        true,
			pendingLines:       []string{"[CUSTOMER] hi"},
			expectWindow:       true,
			registerErr:        fmt.Errorf("redis unavailable"),
			expectPipecatStart: false,
			expectResult:       "skipped_register_failed",
		},
		{
			name:               "happy path runs one turn",
			flagEnabled:        true,
			status:             aicall.StatusProgressing,
			refType:            aicall.ReferenceTypeContactCase,
			expectCountIncr:    true,
			turnCount:          1,
			expectDrain:        true,
			pendingLines:       []string{"[CUSTOMER] I want to cancel"},
			expectWindow:       true,
			expectPipecatStart: true,
			expectResult:       "ran",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetListenDefaultsForTest()
			config.SetAIcallListenEnabledForTest(tt.flagEnabled)
			defer config.SetListenDefaultsForTest()

			mc := gomock.NewController(t)
			defer mc.Finish()

			m := newListenTurnHarness(mc)
			ctx := context.Background()

			c := listeningAIcall()
			c.Status = tt.status
			c.ReferenceType = tt.refType
			if tt.metadata != nil {
				c.Metadata = tt.metadata
			}
			m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)

			if tt.expectStopListening {
				// stopListening on a non-owner (listeningAIcall carries no
				// listen_owns_transcribe key) skips the transcribe stop and goes
				// straight to clearListenState.
				m.req.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), gomock.Any()).Times(0)
				m.req.EXPECT().TranscribeV1TranscribeStop(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				if listenTranscribeIDFromMetadata(c) != uuid.Nil {
					m.cache.EXPECT().ListenAIcallIDRemove(ctx, listenTranscribeIDFromMetadata(c), ltAIcallID).Return(nil)
				} else {
					m.cache.EXPECT().ListenAIcallIDRemove(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				}
				m.cache.EXPECT().ListenStateClear(ctx, ltAIcallID).Return(nil)
				m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)
			} else {
				m.cache.EXPECT().ListenStateClear(gomock.Any(), gomock.Any()).Times(0)
			}

			if tt.expectCountIncr {
				m.cache.EXPECT().ListenTurnCountIncr(ctx, ltAIcallID, listenBufferTTL()).Return(tt.turnCount, nil)
			} else {
				m.cache.EXPECT().ListenTurnCountIncr(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			if tt.expectDrain {
				m.cache.EXPECT().ListenPendingPopAll(ctx, ltAIcallID).Return(tt.pendingLines, nil)
			} else {
				m.cache.EXPECT().ListenPendingPopAll(gomock.Any(), gomock.Any()).Times(0)
			}

			if tt.expectWindow {
				m.cache.EXPECT().ListenWindowGet(ctx, ltAIcallID).Return(tt.pendingLines, nil)
				m.msg.EXPECT().List(ctx, uint64(30), "", gomock.Any()).Return(nil, nil)
				m.util.EXPECT().UUIDCreate().Return(turnPCID)
				m.cache.EXPECT().ListenTurnPipecatcallIDAdd(ctx, ltAIcallID, turnPCID, gomock.Any()).Return(tt.registerErr)
			}

			if tt.expectPipecatStart {
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx, turnPCID, ltCustomerID, gomock.Any(), pmpipecatcall.ReferenceTypeAICall, ltAIcallID,
					gomock.Any(), gomock.Any(),
					pmpipecatcall.STTTypeNone, "", pmpipecatcall.TTSTypeNone, "", "",
				).Return(&pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{ID: turnPCID},
					HostID:   "10.0.0.1",
				}, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, "10.0.0.1", turnPCID, defaultListenTurnTimeout).Return(nil)
			} else {
				m.req.EXPECT().PipecatV1PipecatcallStart(
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Times(0)
			}

			kind := tt.expectKind
			if kind == "" {
				kind = "call"
			}
			delta := metricDelta(t, kind, tt.expectResult, func() {
				m.h.RunListenTurn(ctx, ltAIcallID)
			})
			if delta != 1 {
				t.Errorf("expected exactly one %q increment. got: %v", tt.expectResult, delta)
			}
		})
	}
}

// Test_RunListenTurn_DoesNotWritePipecatcallID pins the load-bearing decision.
//
// Writing the turn's throwaway id to the AIcall row would rotate the agent's own
// conversational turn out from under an in-flight answer, bump tm_update into
// Send's cooldown, and destroy the id mismatch that is the drop signal for
// anything the turn emits. All three at once, silently.
func Test_RunListenTurn_DoesNotWritePipecatcallID(t *testing.T) {
	config.SetListenDefaultsForTest()
	config.SetAIcallListenEnabledForTest(true)
	defer config.SetListenDefaultsForTest()

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)
	ctx := context.Background()

	c := listeningAIcall()
	boundPCID := c.PipecatcallID
	turnPCID := uuid.FromStringOrNil("99990000-0000-4000-8000-0000000000dd")

	m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)
	m.cache.EXPECT().ListenTurnCountIncr(ctx, ltAIcallID, listenBufferTTL()).Return(int64(1), nil)
	m.cache.EXPECT().ListenPendingPopAll(ctx, ltAIcallID).Return([]string{"[CUSTOMER] hello"}, nil)
	m.cache.EXPECT().ListenWindowGet(ctx, ltAIcallID).Return([]string{"[CUSTOMER] hello"}, nil)
	m.msg.EXPECT().List(ctx, uint64(30), "", gomock.Any()).Return(nil, nil)
	m.util.EXPECT().UUIDCreate().Return(turnPCID)

	var registeredID uuid.UUID
	m.cache.EXPECT().ListenTurnPipecatcallIDAdd(ctx, ltAIcallID, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, pcID uuid.UUID, _ time.Duration) error {
			registeredID = pcID
			return nil
		})

	var startedID uuid.UUID
	m.req.EXPECT().PipecatV1PipecatcallStart(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).DoAndReturn(func(_ context.Context, id, _, _ uuid.UUID, _ pmpipecatcall.ReferenceType, _ uuid.UUID,
		_ pmpipecatcall.LLMType, _ []map[string]any, _ pmpipecatcall.STTType, _ string,
		_ pmpipecatcall.TTSType, _, _ string) (*pmpipecatcall.Pipecatcall, error) {
		startedID = id
		return &pmpipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: id}, HostID: "10.0.0.1"}, nil
	})
	m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	// NO AIcallUpdate / AIcallUpdateIfActive / AIcallUpdateNoTouchTMUpdate
	// expectation at all -- gomock fails the test if the turn writes the row.
	m.db.EXPECT().AIcallUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	m.db.EXPECT().AIcallUpdateIfActive(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	m.h.RunListenTurn(ctx, ltAIcallID)

	if startedID == boundPCID {
		t.Errorf("the turn must run on a throwaway id, not the AIcall's bound one. got: %s", startedID)
	}
	if startedID != registeredID {
		t.Errorf("the started id must be the one registered as a listen turn. started: %s, registered: %s", startedID, registeredID)
	}
	if c.PipecatcallID != boundPCID {
		t.Errorf("the AIcall's bound pipecatcall id must be untouched. expected: %s, got: %s", boundPCID, c.PipecatcallID)
	}
}

// Test_runListenTurnWithLines_HangupPath pins that the hangup flush can
// evaluate lines it already holds, bypassing both the drain and the debounce
// lock.
//
// This is why the turn body is a separate function at all: RunListenTurn drains
// the buffer itself and respects the lock, so the hangup path -- which has
// already drained -- would otherwise have no way in.
func Test_runListenTurnWithLines_HangupPath(t *testing.T) {
	config.SetListenDefaultsForTest()
	config.SetAIcallListenEnabledForTest(true)
	defer config.SetListenDefaultsForTest()

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)
	ctx := context.Background()

	c := listeningAIcall()
	turnPCID := uuid.FromStringOrNil("99990000-0000-4000-8000-0000000000ee")

	// The hangup path has ALREADY drained, so neither of these may be called.
	m.cache.EXPECT().ListenPendingPopAll(gomock.Any(), gomock.Any()).Times(0)
	m.cache.EXPECT().ListenTurnTryLock(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	m.cache.EXPECT().ListenWindowGet(ctx, ltAIcallID).Return([]string{"[AGENT] bye"}, nil)
	m.msg.EXPECT().List(ctx, uint64(30), "", gomock.Any()).Return(nil, nil)
	m.util.EXPECT().UUIDCreate().Return(turnPCID)
	m.cache.EXPECT().ListenTurnPipecatcallIDAdd(ctx, ltAIcallID, turnPCID, gomock.Any()).Return(nil)
	m.req.EXPECT().PipecatV1PipecatcallStart(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(&pmpipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: turnPCID}, HostID: "10.0.0.1"}, nil).Times(1)
	m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	delta := metricDelta(t, "call", "ran", func() {
		m.h.runListenTurnWithLines(ctx, c, []string{"[AGENT] bye"})
	})
	if delta != 1 {
		t.Errorf("expected exactly one 'ran' increment. got: %v", delta)
	}
}

// Test_speakerTag pins the in/out -> CUSTOMER/AGENT mapping.
//
// This is a golden test on purpose. A reversed mapping is a SILENT correctness
// failure: the AI keeps working and keeps notifying, it just attributes the
// customer's words to the agent and vice versa -- which can produce a
// confidently wrong proactive message (e.g. telling the agent the customer
// threatened to cancel when it was the agent who said it).
//
// The values below are the design's structural mapping, which the
// implementation plan's Task 0 Step 1 records as PROVISIONAL pending an
// end-to-end empirical check. If that check finds the reverse, this test and
// speakerTag both flip together.
func Test_speakerTag(t *testing.T) {
	tests := []struct {
		name      string
		direction tmtranscript.Direction
		expect    string
	}{
		{"in is the customer", tmtranscript.DirectionIn, "[CUSTOMER]"},
		{"out is the agent", tmtranscript.DirectionOut, "[AGENT]"},
		{"both is not guessed", tmtranscript.DirectionBoth, "[SPEAKER]"},
		{"unknown is not guessed", tmtranscript.Direction("weird"), "[SPEAKER]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := speakerTag(tt.direction); got != tt.expect {
				t.Errorf("speakerTag(%q) mismatch. expected: %q, got: %q", tt.direction, tt.expect, got)
			}
		})
	}
}

// Test_EventTMTranscriptCreated covers intake's drops and its fan-out.
func Test_EventTMTranscriptCreated(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-000000000001")
	aicallA := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-00000000000a")
	aicallB := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-00000000000b")

	deleted := func() *time.Time { tm := time.Now(); return &tm }()

	tests := []struct {
		name string

		transcript *tmtranscript.Transcript

		expectResolve bool
		resolvedIDs   []uuid.UUID
		resolveErr    error
		lockAcquired  []bool

		expectBuffered int
		expectLocked   int
		expectLine     string
	}{
		{
			name: "a deleted transcript is dropped before any redis call",
			// transcripthandler.dbDelete publishes transcript_created on DELETE
			// too (a known upstream bug). Without this guard a deleted line
			// replays into the LLM as freshly-spoken content.
			transcript: &tmtranscript.Transcript{
				TranscribeID: transcribeID,
				Direction:    tmtranscript.DirectionIn,
				Message:      "deleted line",
				TMDelete:     deleted,
			},
			expectResolve: false,
		},
		{
			name: "an empty message is dropped before any redis call",
			transcript: &tmtranscript.Transcript{
				TranscribeID: transcribeID,
				Direction:    tmtranscript.DirectionIn,
				Message:      "   \n\t ",
			},
			expectResolve: false,
		},
		{
			name: "an unknown transcribe id is dropped after one redis call and no DB call",
			// This is 99.9% of platform events. It must cost one SMEMBERS and
			// nothing else -- no DB query, no RPC.
			transcript: &tmtranscript.Transcript{
				TranscribeID: transcribeID,
				Direction:    tmtranscript.DirectionIn,
				Message:      "hello",
			},
			expectResolve: true,
			resolvedIDs:   nil,
		},
		{
			name: "a resolver error is dropped, not propagated",
			transcript: &tmtranscript.Transcript{
				TranscribeID: transcribeID,
				Direction:    tmtranscript.DirectionIn,
				Message:      "hello",
			},
			expectResolve: true,
			resolveErr:    fmt.Errorf("redis unavailable"),
		},
		{
			name: "two AIcalls sharing one transcribe each get the segment buffered",
			// Two Cases open on one call. A single-valued resolver key would let
			// the second listener silently steal the first's mapping; the set
			// makes both independent.
			transcript: &tmtranscript.Transcript{
				TranscribeID: transcribeID,
				Direction:    tmtranscript.DirectionOut,
				Message:      "  we can help with that  ",
			},
			expectResolve:  true,
			resolvedIDs:    []uuid.UUID{aicallA, aicallB},
			lockAcquired:   []bool{true, true},
			expectBuffered: 2,
			expectLine:     "[AGENT] we can help with that",
		},
		{
			name: "a buffered-but-locked segment runs no turn",
			transcript: &tmtranscript.Transcript{
				TranscribeID: transcribeID,
				Direction:    tmtranscript.DirectionIn,
				Message:      "I want to cancel",
			},
			expectResolve:  true,
			resolvedIDs:    []uuid.UUID{aicallA},
			lockAcquired:   []bool{false},
			expectBuffered: 1,
			expectLocked:   1,
			expectLine:     "[CUSTOMER] I want to cancel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetListenDefaultsForTest()
			defer config.SetListenDefaultsForTest()

			mc := gomock.NewController(t)
			defer mc.Finish()

			m := newListenTurnHarness(mc)
			ctx := context.Background()

			if !tt.expectResolve {
				// NO redis call at all -- the drop must happen before it.
				m.cache.EXPECT().ListenAIcallIDsGet(gomock.Any(), gomock.Any()).Times(0)
			} else {
				m.cache.EXPECT().ListenAIcallIDsGet(ctx, transcribeID).Return(tt.resolvedIDs, tt.resolveErr).Times(1)
			}

			// Intake itself is NO-DB, NO-RPC by construction. The only DB read
			// that can occur is inside a DETACHED turn goroutine a won debounce
			// spawns; failing it there keeps that goroutine on RunListenTurn's
			// own short-circuit so this test stays about intake.
			m.db.EXPECT().AIcallGet(gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("not under test here")).AnyTimes()

			for i, id := range tt.resolvedIDs {
				m.cache.EXPECT().ListenPendingPush(ctx, id, tt.expectLine, listenBufferTTL()).Return(nil)
				m.cache.EXPECT().ListenWindowPush(ctx, id, tt.expectLine, 40, listenBufferTTL()).Return(nil)
				m.cache.EXPECT().ListenTurnTryLock(ctx, id, 20*time.Second).Return(tt.lockAcquired[i], nil)
			}

			// Winning the debounce spawns a detached turn; those goroutines run
			// against a fresh handler-less context and are allowed to make any
			// number of calls or none, so they are not asserted here (the turn
			// body has its own tests).
			m.cache.EXPECT().ListenTurnCountIncr(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil).AnyTimes()
			m.cache.EXPECT().ListenPendingPopAll(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

			bufferedDelta := testutil.ToFloat64(promListenSegmentTotal.WithLabelValues("buffered"))
			lockedDelta := testutil.ToFloat64(promListenTurnTotal.WithLabelValues("call", "skipped_locked"))

			m.h.EventTMTranscriptCreated(ctx, tt.transcript)

			gotBuffered := testutil.ToFloat64(promListenSegmentTotal.WithLabelValues("buffered")) - bufferedDelta
			if int(gotBuffered) != tt.expectBuffered {
				t.Errorf("buffered count mismatch. expected: %d, got: %v", tt.expectBuffered, gotBuffered)
			}
			gotLocked := testutil.ToFloat64(promListenTurnTotal.WithLabelValues("call", "skipped_locked")) - lockedDelta
			if int(gotLocked) != tt.expectLocked {
				t.Errorf("skipped_locked count mismatch. expected: %d, got: %v", tt.expectLocked, gotLocked)
			}

			// Let any spawned turn goroutine settle before the controller's
			// Finish, so its calls land against the AnyTimes expectations above
			// rather than after the controller closes.
			time.Sleep(50 * time.Millisecond)
		})
	}
}

// Test_stopListening_NeverTerminatesTheAIcall is a regression test for a naming
// hazard, not a hypothetical.
//
// "Stop listening" and "terminate the AIcall" are one word apart and worlds
// apart in effect: ProcessTerminate ends the AIcall itself (status terminated +
// activeflow service stop), which would kill the agent's entire Insight Q&A
// session. Every stop path must leave the panel working normally.
func Test_stopListening_NeverTerminatesTheAIcall(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("bbbb0000-0000-4000-8000-000000000001")
	hostID := uuid.FromStringOrNil("bbbb0000-0000-4000-8000-0000000000ff")

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)
	ctx := context.Background()

	c := listeningAIcall()
	c.Metadata = map[string]any{
		aicall.MetaKeyListenTranscribeID:   transcribeID.String(),
		aicall.MetaKeyListenOwnsTranscribe: true,
		"prompt_snapshots":                 []any{},
	}

	// NO FlowV1ActiveflowServiceStop and NO status update -- gomock fails if
	// either is called.
	m.req.EXPECT().FlowV1ActiveflowServiceStop(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	m.db.EXPECT().AIcallUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// The two calls it SHOULD make, in order: the owned-transcribe stop, then
	// clearListenState.
	get := m.req.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(&tmtranscribe.Transcribe{
		Identity: commonidentity.Identity{ID: transcribeID},
		HostID:   hostID,
		Status:   tmtranscribe.StatusProgressing,
	}, nil)
	stop := m.req.EXPECT().TranscribeV1TranscribeStop(ctx, hostID, transcribeID).Return(nil, nil)
	rem := m.cache.EXPECT().ListenAIcallIDRemove(ctx, transcribeID, ltAIcallID).Return(nil)
	clear := m.cache.EXPECT().ListenStateClear(ctx, ltAIcallID).Return(nil)

	var wroteFields map[aicall.Field]any
	update := m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[aicall.Field]any) error {
			wroteFields = fields
			return nil
		})
	gomock.InOrder(get, stop, rem, clear, update)

	m.h.stopListening(ctx, c)

	metadata, ok := wroteFields[aicall.FieldMetadata].(map[string]any)
	if !ok {
		t.Fatalf("the clear must carry a metadata map. got: %v", wroteFields[aicall.FieldMetadata])
	}
	if _, present := metadata[aicall.MetaKeyListenTranscribeID]; present {
		t.Errorf("listen_transcribe_id must be removed from metadata")
	}
	if _, present := metadata[aicall.MetaKeyListenOwnsTranscribe]; present {
		t.Errorf("listen_owns_transcribe must be removed from metadata")
	}
	if _, present := metadata["prompt_snapshots"]; !present {
		t.Errorf("every other metadata key must survive the clear")
	}
	if got := wroteFields[aicall.FieldListenCallID]; got != uuid.Nil {
		t.Errorf("listen_call_id must be cleared to uuid.Nil. got: %v", got)
	}
}

// Test_stopListening_NonOwnerNeverStopsTheTranscribe pins the ownership guard:
// a non-owner must never touch a session another listening Case still depends
// on.
func Test_stopListening_NonOwnerNeverStopsTheTranscribe(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("bbbb0000-0000-4000-8000-000000000002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)
	ctx := context.Background()

	c := listeningAIcall()
	c.Metadata = map[string]any{
		aicall.MetaKeyListenTranscribeID:   transcribeID.String(),
		aicall.MetaKeyListenOwnsTranscribe: false,
	}

	m.req.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), gomock.Any()).Times(0)
	m.req.EXPECT().TranscribeV1TranscribeStop(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// But this AIcall's OWN membership and state still go away.
	m.cache.EXPECT().ListenAIcallIDRemove(ctx, transcribeID, ltAIcallID).Return(nil)
	m.cache.EXPECT().ListenStateClear(ctx, ltAIcallID).Return(nil)
	m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)

	m.h.stopListening(ctx, c)
}

// Test_clearListenState_StepOrder pins that the resolver-set removal happens
// BEFORE the metadata clear.
//
// The SREM needs the transcribe id, and the metadata clear destroys it. Doing
// them in the other order leaves a stale (transcribe_id, aicall_id) pairing that
// intake can still match, feeding segments to an AIcall that has stopped
// listening.
func Test_clearListenState_StepOrder(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("bbbb0000-0000-4000-8000-000000000003")

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)
	ctx := context.Background()

	c := listeningAIcall()
	c.Metadata = map[string]any{aicall.MetaKeyListenTranscribeID: transcribeID.String()}

	rem := m.cache.EXPECT().ListenAIcallIDRemove(ctx, transcribeID, ltAIcallID).Return(nil)
	clear := m.cache.EXPECT().ListenStateClear(ctx, ltAIcallID).Return(nil)
	update := m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)
	gomock.InOrder(rem, clear, update)

	m.h.clearListenState(ctx, c)
}

// Test_stopListenByCallID_ClearsEveryMatch pins the plural lookup.
//
// Two Cases open on one call each get their own AIcall (the active-reference
// unique key is per Case, not per customer), and BOTH must be cleared when the
// call hangs up. Clearing only the first leaves the second listening to a
// transcribe session that has stopped producing.
func Test_stopListenByCallID_ClearsEveryMatch(t *testing.T) {
	aicallA := uuid.FromStringOrNil("cccc0000-0000-4000-8000-00000000000a")
	aicallB := uuid.FromStringOrNil("cccc0000-0000-4000-8000-00000000000b")

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)
	ctx := context.Background()

	rows := []*aicall.AIcall{
		{Identity: commonidentity.Identity{ID: aicallA}, ReferenceType: aicall.ReferenceTypeContactCase},
		{Identity: commonidentity.Identity{ID: aicallB}, ReferenceType: aicall.ReferenceTypeContactCase},
	}

	m.db.EXPECT().AIcallList(ctx, uint64(10), "", map[aicall.Field]any{
		aicall.FieldReferenceType: aicall.ReferenceTypeContactCase,
		aicall.FieldListenCallID:  ltCallID,
		aicall.FieldDeleted:       false,
	}).Return(rows, nil)

	// Empty buffers -> no flush turn, but BOTH get cleared.
	for _, id := range []uuid.UUID{aicallA, aicallB} {
		m.cache.EXPECT().ListenPendingPopAll(ctx, id).Return(nil, nil)
		m.cache.EXPECT().ListenStateClear(ctx, id).Return(nil)
		m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, id, gomock.Any()).Return(nil)
	}

	m.h.stopListenByCallID(ctx, ltCallID)
}

// Test_stopListenByCallID_FinalFlush pins that the last words of a call are
// still evaluated.
//
// The debounce means the final lines before a hangup sit unevaluated in the
// pending buffer. This is the only chance to read them, and it deliberately
// bypasses the debounce lock -- there is no "next segment" coming.
func Test_stopListenByCallID_FinalFlush(t *testing.T) {
	turnPCID := uuid.FromStringOrNil("cccc0000-0000-4000-8000-0000000000cc")

	config.SetListenDefaultsForTest()
	defer config.SetListenDefaultsForTest()

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)
	ctx := context.Background()

	// Keeps listeningAIcall's default listen_transcribe_id metadata: this row
	// is a genuine call-kind listen session, and clearListenState below reads
	// that same id to remove the AIcall's resolver membership.
	row := listeningAIcall()

	m.db.EXPECT().AIcallList(ctx, uint64(10), "", gomock.Any()).Return([]*aicall.AIcall{row}, nil)

	// A non-empty buffer reaches runListenTurnWithLines with exactly those
	// lines -- and NEVER through the debounce lock.
	m.cache.EXPECT().ListenPendingPopAll(ctx, ltAIcallID).Return([]string{"[CUSTOMER] one last thing"}, nil)
	m.cache.EXPECT().ListenTurnTryLock(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	m.cache.EXPECT().ListenWindowGet(ctx, ltAIcallID).Return([]string{"[CUSTOMER] one last thing"}, nil)
	m.msg.EXPECT().List(ctx, uint64(30), "", gomock.Any()).Return(nil, nil)
	m.util.EXPECT().UUIDCreate().Return(turnPCID)
	m.cache.EXPECT().ListenTurnPipecatcallIDAdd(ctx, ltAIcallID, turnPCID, gomock.Any()).Return(nil)
	m.req.EXPECT().PipecatV1PipecatcallStart(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(&pmpipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: turnPCID}, HostID: "10.0.0.1"}, nil)
	m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	// Then the stop itself. listeningAIcall carries no listen_owns_transcribe
	// key, so stopListening skips the transcribe stop and clearListenState
	// only removes this AIcall's resolver membership.
	m.cache.EXPECT().ListenAIcallIDRemove(ctx, listenTranscribeIDFromMetadata(row), ltAIcallID).Return(nil)
	m.cache.EXPECT().ListenStateClear(ctx, ltAIcallID).Return(nil)
	m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)

	delta := metricDelta(t, "call", "ran", func() {
		m.h.stopListenByCallID(ctx, ltCallID)
	})
	if delta != 1 {
		t.Errorf("the final flush must run exactly one turn. got: %v", delta)
	}
}

// Test_stopListenByCallID_NilCallID is the direct regression test for review
// round 1's security MEDIUM-2.
//
// callID arrives from an unvalidated call_hangup event field, and the migration
// declares listen_call_id NOT NULL DEFAULT 0x00... -- so uuid.Nil is the value
// EVERY non-listening contact_case AIcall carries, across every tenant. A zero
// id must therefore never reach the query at all.
func Test_stopListenByCallID_NilCallID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)
	ctx := context.Background()

	// Zero calls of any kind: no list, no drain, no clear, no stop.
	m.db.EXPECT().AIcallList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	m.cache.EXPECT().ListenPendingPopAll(gomock.Any(), gomock.Any()).Times(0)
	m.cache.EXPECT().ListenStateClear(gomock.Any(), gomock.Any()).Times(0)
	m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	m.h.stopListenByCallID(ctx, uuid.Nil)
}

// Test_EventCMCallHangup_NilCallID pins the same guard one layer up, at the
// event boundary itself -- neither the reference lookup nor the listen sweep
// may run for a zero id.
func Test_EventCMCallHangup_NilCallID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)

	m.db.EXPECT().AIcallGetByReferenceID(gomock.Any(), gomock.Any()).Times(0)
	m.db.EXPECT().AIcallList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	m.h.EventCMCallHangup(context.Background(), &cmcall.Call{})
}

// Test_stopListenByCallID_Paginates is the direct regression test for review
// round 1's MEDIUM-2 (code review).
//
// The lookup was a single fixed page of 10 with no continuation, so an
// eleventh listening AIcall was silently never cleaned up: its resolver
// membership survived, its listen_call_id survived, and if it owned the
// transcribe session that session was never stopped.
func Test_stopListenByCallID_Paginates(t *testing.T) {
	ctx := context.Background()

	// A full first page forces a second fetch; the short second page proves
	// exhaustion and ends the sweep.
	firstPage := []*aicall.AIcall{}
	for i := 0; i < listenStopPageSize; i++ {
		tm := time.Date(2026, 9, 4, 12, 0, i, 0, time.UTC)
		firstPage = append(firstPage, &aicall.AIcall{
			Identity:      commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
			ReferenceType: aicall.ReferenceTypeContactCase,
			TMCreate:      &tm,
		})
	}
	lastTM := firstPage[len(firstPage)-1].TMCreate
	expectedToken := lastTM.UTC().Format(utilhandler.ISO8601Layout)

	overflow := &aicall.AIcall{
		Identity:      commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		ReferenceType: aicall.ReferenceTypeContactCase,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)

	filters := map[aicall.Field]any{
		aicall.FieldReferenceType: aicall.ReferenceTypeContactCase,
		aicall.FieldListenCallID:  ltCallID,
		aicall.FieldDeleted:       false,
	}

	gomock.InOrder(
		m.db.EXPECT().AIcallList(ctx, uint64(listenStopPageSize), "", filters).Return(firstPage, nil),
		// The continuation token is the last row's tm_create, in the SAME
		// fixed-precision layout AIcallList's own default token uses.
		m.db.EXPECT().AIcallList(ctx, uint64(listenStopPageSize), expectedToken, filters).
			Return([]*aicall.AIcall{overflow}, nil),
	)

	// Every row on BOTH pages is torn down -- the eleventh included.
	for _, row := range append(append([]*aicall.AIcall{}, firstPage...), overflow) {
		m.cache.EXPECT().ListenPendingPopAll(ctx, row.ID).Return(nil, nil)
		m.cache.EXPECT().ListenStateClear(ctx, row.ID).Return(nil)
		m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, row.ID, gomock.Any()).Return(nil)
	}

	m.h.stopListenByCallID(ctx, ltCallID)
}

// Test_stopListenByCallID_PageBudget pins that the sweep terminates rather
// than paging forever, and that hitting the budget is not silent.
func Test_stopListenByCallID_PageBudget(t *testing.T) {
	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newListenTurnHarness(mc)

	// Always a full page: the source never proves exhaustion.
	m.db.EXPECT().AIcallList(ctx, uint64(listenStopPageSize), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, size uint64, _ string, _ map[aicall.Field]any) ([]*aicall.AIcall, error) {
			rows := []*aicall.AIcall{}
			for i := uint64(0); i < size; i++ {
				tm := time.Date(2026, 9, 4, 12, 0, int(i), 0, time.UTC)
				rows = append(rows, &aicall.AIcall{
					Identity:      commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
					ReferenceType: aicall.ReferenceTypeContactCase,
					TMCreate:      &tm,
				})
			}
			return rows, nil
		}).Times(listenStopMaxPages)

	m.cache.EXPECT().ListenPendingPopAll(ctx, gomock.Any()).Return(nil, nil).AnyTimes()
	m.cache.EXPECT().ListenStateClear(ctx, gomock.Any()).Return(nil).AnyTimes()
	m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// The assertion that matters: it returns. Times(listenStopMaxPages) above
	// is what proves it stopped at the budget rather than looping.
	m.h.stopListenByCallID(ctx, ltCallID)
}
