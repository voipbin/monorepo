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

// metricDelta reports how much a promListenTurnTotal label moved across fn.
func metricDelta(t *testing.T, label string, fn func()) float64 {
	t.Helper()
	before := testutil.ToFloat64(promListenTurnTotal.WithLabelValues(label))
	fn()
	return testutil.ToFloat64(promListenTurnTotal.WithLabelValues(label)) - before
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

		expectPipecatStart bool
		expectResult       string
	}{
		{
			name: "flag off stops listening entirely",
			// Not merely "clears bookkeeping": a bare state clear would leave a
			// still-running owned STT session with its handle lost, so a
			// rollback would strand a billed stream until the call ended.
			flagEnabled:  false,
			status:       aicall.StatusProgressing,
			refType:      aicall.ReferenceTypeContactCase,
			expectResult: "skipped_disabled",
		},
		{
			name:         "terminated aicall stops listening",
			flagEnabled:  true,
			status:       aicall.StatusTerminated,
			refType:      aicall.ReferenceTypeContactCase,
			expectResult: "skipped_invalid",
		},
		{
			name:         "non contact_case reference type stops listening",
			flagEnabled:  true,
			status:       aicall.StatusProgressing,
			refType:      aicall.ReferenceTypeCall,
			expectResult: "skipped_invalid",
		},
		{
			name:         "missing listen_transcribe_id metadata stops listening",
			flagEnabled:  true,
			status:       aicall.StatusProgressing,
			refType:      aicall.ReferenceTypeContactCase,
			metadata:     map[string]any{},
			expectResult: "skipped_invalid",
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
			name:            "turn cap exceeded stops listening",
			flagEnabled:     true,
			status:          aicall.StatusProgressing,
			refType:         aicall.ReferenceTypeContactCase,
			expectCountIncr: true,
			turnCount:       61,
			expectResult:    "skipped_cap",
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

			delta := metricDelta(t, tt.expectResult, func() {
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

	delta := metricDelta(t, "ran", func() {
		m.h.runListenTurnWithLines(ctx, c, []string{"[AGENT] bye"})
	})
	if delta != 1 {
		t.Errorf("expected exactly one 'ran' increment. got: %v", delta)
	}
}
