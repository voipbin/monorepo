package aiaudithandler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/participant"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/geminiaudithandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

var (
	g4Cust    = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000001")
	g4OtherC  = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000002")
	g4CallID  = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000003")
	g4AIID    = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000004")
	g4AI2ID   = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000005")
	g4AuditID = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000006")
	g4HistID  = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000007")
	g4MsgID1  = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000008")
	g4MsgID2  = uuid.FromStringOrNil("10000000-0000-4000-8000-000000000009")
)

type g4Mocks struct {
	h      *aiauditHandler
	db     *dbhandler.MockDBHandler
	util   *utilhandler.MockUtilHandler
	gemini *geminiaudithandler.MockGeminiAuditHandler
}

func g4New(mc *gomock.Controller) *g4Mocks {
	m := &g4Mocks{
		db:     dbhandler.NewMockDBHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
		gemini: geminiaudithandler.NewMockGeminiAuditHandler(mc),
	}
	m.h = &aiauditHandler{
		utilHandler:   m.util,
		db:            m.db,
		geminiHandler: m.gemini,
		semaphore:     make(chan struct{}, maxConcurrentGlobal),
	}
	return m
}

func g4Wait(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for the background job")
	}
}

func g4Call(assist aicall.AssistanceType) *aicall.AIcall {
	return &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: g4CallID, CustomerID: g4Cust},
		AssistanceType: assist,
		AssistanceID:   g4AIID,
		Status:         aicall.StatusTerminated,
		Metadata: map[string]any{
			aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
				{AIID: g4AIID, PromptHistoryID: g4HistID, Prompt: "snapshot prompt"},
			},
		},
	}
}

// ---- Create ----

func Test_Create_errors(t *testing.T) {
	rpcErr := errors.New("backend failed")
	terminated := func() *aicall.AIcall { return g4Call(aicall.AssistanceTypeAI) }

	tests := []struct {
		name    string
		setup   func(m *g4Mocks)
		wantErr string
		wantIs  error
	}{
		{
			name: "aicall lookup fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(nil, rpcErr)
			},
			wantErr: "could not get aicall",
			wantIs:  rpcErr,
		},
		{
			name: "aicall belongs to another customer",
			setup: func(m *g4Mocks) {
				ac := terminated()
				ac.CustomerID = g4OtherC
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(ac, nil)
			},
			wantIs: dbhandler.ErrNotFound,
		},
		{
			name: "aicall still progressing",
			setup: func(m *g4Mocks) {
				ac := terminated()
				ac.Status = aicall.StatusProgressing
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(ac, nil)
			},
			wantErr: "aicall not terminated",
		},
		{
			name: "progressing count lookup fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(terminated(), nil)
				m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(0), rpcErr)
			},
			wantErr: "could not count progressing audits",
			wantIs:  rpcErr,
		},
		{
			name: "rate limit reached",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(terminated(), nil)
				m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(maxConcurrentCustomer), nil)
			},
			wantErr: "rate limit exceeded",
		},
		{
			name: "participant list fails for a team call",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(g4Call(aicall.AssistanceTypeTeam), nil)
				m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
				m.db.EXPECT().ParticipantListByAIcallID(gomock.Any(), g4CallID, uint64(100), "").Return(nil, rpcErr)
			},
			wantErr: "could not list participants",
			wantIs:  rpcErr,
		},
		{
			name: "team call without participants",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(g4Call(aicall.AssistanceTypeTeam), nil)
				m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
				m.db.EXPECT().ParticipantListByAIcallID(gomock.Any(), g4CallID, uint64(100), "").Return([]*participant.Participant{}, nil)
			},
			wantErr: "no AI participants found",
		},
		{
			name: "upsert fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(terminated(), nil)
				m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
				m.util.EXPECT().UUIDCreate().Return(g4AuditID)
				m.db.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).Return(int64(0), rpcErr)
			},
			wantErr: "could not upsert audit",
			wantIs:  rpcErr,
		},
		{
			name: "audit already progressing (no row affected)",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(terminated(), nil)
				m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
				m.util.EXPECT().UUIDCreate().Return(g4AuditID)
				m.db.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).Return(int64(0), nil)
			},
			wantErr: "already progressing",
		},
		{
			name: "reload fails",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(terminated(), nil)
				m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
				m.util.EXPECT().UUIDCreate().Return(g4AuditID)
				m.db.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).Return(int64(1), nil)
				m.db.EXPECT().AIAuditList(gomock.Any(), uint64(1), "", gomock.Any()).Return(nil, rpcErr)
			},
			wantErr: "could not reload audit record",
			wantIs:  rpcErr,
		},
		{
			name: "reload returns nothing",
			setup: func(m *g4Mocks) {
				m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(terminated(), nil)
				m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
				m.util.EXPECT().UUIDCreate().Return(g4AuditID)
				m.db.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).Return(int64(1), nil)
				m.db.EXPECT().AIAuditList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*aiaudit.AIAudit{}, nil)
			},
			wantErr: "no audit record found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)
			tt.setup(m)

			res, err := m.h.Create(context.Background(), g4Cust, g4CallID, "en-US")
			if res != nil || err == nil {
				t.Fatalf("expected an error, got %v, %v", res, err)
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("expected error chain to contain %v, got: %v", tt.wantIs, err)
			}
		})
	}
}

func Test_Create_upsertCarriesResolvedLanguageAndPromptHistory(t *testing.T) {
	// The controller finishes through its own cleanup, which runs after the wait registered below.
	mc := gomock.NewController(t)
	m := g4New(mc)

	ac := g4Call(aicall.AssistanceTypeAI)
	ac.STTLanguage = "ja-JP"

	// The first created job is waited for before the controller finishes.
	done := make(chan struct{})
	t.Cleanup(func() { g4Wait(t, done) })

	m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(ac, nil)
	m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(maxConcurrentCustomer-1), nil)
	m.util.EXPECT().UUIDCreate().Return(g4AuditID)
	m.db.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, a *aiaudit.AIAudit) (int64, error) {
			if a.ID != g4AuditID || a.CustomerID != g4Cust || a.AIcallID != g4CallID || a.AIID != g4AIID ||
				a.PromptHistoryID != g4HistID || a.Language != "ja-JP" {
				t.Errorf("unexpected upsert row: %+v", a)
			}
			return 1, nil
		})
	record := &aiaudit.AIAudit{Identity: commonidentity.Identity{ID: g4AuditID, CustomerID: g4Cust}, AIcallID: g4CallID, AIID: g4AIID}
	m.db.EXPECT().AIAuditList(gomock.Any(), uint64(1), "", map[aiaudit.Field]any{
		aiaudit.FieldAIcallID: g4CallID,
		aiaudit.FieldAIID:     g4AIID,
		aiaudit.FieldDeleted:  false,
	}).Return([]*aiaudit.AIAudit{record}, nil)

	// detached job: fails fast on the message lookup, then writes the final result.
	m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db down"))
	m.db.EXPECT().AIAuditUpdateFinal(gomock.Any(), g4AuditID, aiaudit.StatusFailed, nil, json.RawMessage(nil), string(aiaudit.ErrorEvaluatorUnavailable), nil).
		DoAndReturn(func(context.Context, uuid.UUID, aiaudit.Status, *int, json.RawMessage, string, []uuid.UUID) (int64, error) {
			close(done)
			return 1, nil
		})

	res, err := m.h.Create(context.Background(), g4Cust, g4CallID, "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(res) != 1 || res[0] != record {
		t.Errorf("unexpected result: %v", res)
	}
}

func Test_Create_teamCallStartsOneJobPerParticipant(t *testing.T) {
	// The controller finishes through its own cleanup, which runs after the wait registered below.
	mc := gomock.NewController(t)
	m := g4New(mc)

	ac := g4Call(aicall.AssistanceTypeTeam)

	done := make(chan struct{}, 2)
	t.Cleanup(func() {
		g4Wait(t, chanFromCount(done, 2))
	})

	m.db.EXPECT().AIcallGet(gomock.Any(), g4CallID).Return(ac, nil)
	m.db.EXPECT().AIAuditCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
	m.db.EXPECT().ParticipantListByAIcallID(gomock.Any(), g4CallID, uint64(100), "").Return([]*participant.Participant{
		{AIID: g4AIID}, {AIID: g4AI2ID},
	}, nil)

	auditIDs := []uuid.UUID{g4AuditID, uuid.FromStringOrNil("10000000-0000-4000-8000-00000000000a")}
	m.util.EXPECT().UUIDCreate().Return(auditIDs[0])
	m.util.EXPECT().UUIDCreate().Return(auditIDs[1])
	m.db.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, a *aiaudit.AIAudit) (int64, error) {
			// Only the first AI has a snapshot with a history id.
			if a.AIID == g4AIID && a.PromptHistoryID != g4HistID {
				t.Errorf("first AI must carry its prompt history id: %+v", a)
			}
			if a.AIID == g4AI2ID && a.PromptHistoryID != uuid.Nil {
				t.Errorf("second AI has no snapshot, history id must be nil: %+v", a)
			}
			return 1, nil
		}).Times(2)
	m.db.EXPECT().AIAuditList(gomock.Any(), uint64(1), "", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uint64, _ string, f map[aiaudit.Field]any) ([]*aiaudit.AIAudit, error) {
			id := auditIDs[0]
			if f[aiaudit.FieldAIID] == g4AI2ID {
				id = auditIDs[1]
			}
			return []*aiaudit.AIAudit{{Identity: commonidentity.Identity{ID: id, CustomerID: g4Cust}}}, nil
		}).Times(2)

	// The first job stops at the message lookup; the second AI has no snapshot and stops earlier.
	m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db down")).Times(1)
	m.db.EXPECT().AIAuditUpdateFinal(gomock.Any(), gomock.Any(), aiaudit.StatusFailed, nil, json.RawMessage(nil), gomock.Any(), nil).
		DoAndReturn(func(context.Context, uuid.UUID, aiaudit.Status, *int, json.RawMessage, string, []uuid.UUID) (int64, error) {
			done <- struct{}{}
			return 1, nil
		}).Times(2)

	res, err := m.h.Create(context.Background(), g4Cust, g4CallID, "en-US")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 records, got %d", len(res))
	}
}

// chanFromCount returns a channel that is closed once n values were received from src.
func chanFromCount(src <-chan struct{}, n int) <-chan struct{} {
	out := make(chan struct{})
	go func() {
		for i := 0; i < n; i++ {
			<-src
		}
		close(out)
	}()
	return out
}

// ---- runAuditJob ----

type g4Final struct {
	status aiaudit.Status
	score  *int
	eval   json.RawMessage
	errStr string
	msgIDs []uuid.UUID
}

// g4ExpectFinal records the single final write into out.
func g4ExpectFinal(m *g4Mocks, out *g4Final, n int64, err error) {
	m.db.EXPECT().AIAuditUpdateFinal(gomock.Any(), g4AuditID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, st aiaudit.Status, sc *int, ev json.RawMessage, es string, ids []uuid.UUID) (int64, error) {
			*out = g4Final{status: st, score: sc, eval: ev, errStr: es, msgIDs: ids}
			return n, err
		})
}

func Test_runAuditJob_earlyFailures(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(ac *aicall.AIcall)
		wantErr aiaudit.Error
	}{
		{
			name:    "metadata missing",
			mutate:  func(ac *aicall.AIcall) { ac.Metadata = nil },
			wantErr: aiaudit.ErrorInvalidCallMetadata,
		},
		{
			name: "snapshot for this AI missing",
			mutate: func(ac *aicall.AIcall) {
				ac.Metadata = map[string]any{aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{{AIID: g4AI2ID, PromptHistoryID: g4HistID}}}
			},
			wantErr: aiaudit.ErrorPromptSnapshotNotFound,
		},
		{
			name: "snapshot without history id",
			mutate: func(ac *aicall.AIcall) {
				ac.Metadata = map[string]any{aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{{AIID: g4AIID, Prompt: "p"}}}
			},
			wantErr: aiaudit.ErrorPromptSnapshotNoHistoryID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)

			ac := g4Call(aicall.AssistanceTypeAI)
			tt.mutate(ac)
			var fin g4Final
			g4ExpectFinal(m, &fin, 1, nil)

			m.h.runAuditJob(context.Background(), g4AuditID, ac, g4AIID, "en-US")

			if fin.status != aiaudit.StatusFailed || fin.errStr != string(tt.wantErr) || fin.score != nil || fin.msgIDs != nil {
				t.Errorf("unexpected final write: %+v", fin)
			}
			if len(m.h.semaphore) != 0 {
				t.Errorf("semaphore slot not released")
			}
		})
	}
}

func Test_runAuditJob_messageListFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().MessageList(gomock.Any(), uint64(maxMessages), "", map[message.Field]any{
		message.FieldAIcallID: g4CallID,
		message.FieldDeleted:  false,
	}).Return(nil, errors.New("db down"))
	var fin g4Final
	g4ExpectFinal(m, &fin, 1, nil)

	m.h.runAuditJob(context.Background(), g4AuditID, g4Call(aicall.AssistanceTypeAI), g4AIID, "en-US")

	if fin.status != aiaudit.StatusFailed || fin.errStr != string(aiaudit.ErrorEvaluatorUnavailable) {
		t.Errorf("unexpected final write: %+v", fin)
	}
}

func Test_runAuditJob_teamCallFiltersByActiveAI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().MessageList(gomock.Any(), uint64(maxMessages), "", map[message.Field]any{
		message.FieldAIcallID:   g4CallID,
		message.FieldDeleted:    false,
		message.FieldActiveAIID: g4AIID,
	}).Return(nil, errors.New("db down"))
	var fin g4Final
	g4ExpectFinal(m, &fin, 1, nil)

	m.h.runAuditJob(context.Background(), g4AuditID, g4Call(aicall.AssistanceTypeTeam), g4AIID, "en-US")

	if fin.status != aiaudit.StatusFailed {
		t.Errorf("unexpected final write: %+v", fin)
	}
}

func Test_runAuditJob_cancelledBeforeEvaluation(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*message.Message{{Content: "hi"}}, nil)
	var fin g4Final
	g4ExpectFinal(m, &fin, 1, nil)

	// No Evaluate expectation: a cancelled job must not reach the evaluator.
	m.h.runAuditJob(ctx, g4AuditID, g4Call(aicall.AssistanceTypeAI), g4AIID, "en-US")

	if fin.status != aiaudit.StatusFailed || fin.errStr != string(aiaudit.ErrorCancelled) || fin.msgIDs != nil {
		t.Errorf("unexpected final write: %+v", fin)
	}
}

func Test_runAuditJob_evaluatorErrors(t *testing.T) {
	tests := []struct {
		name    string
		result  *geminiaudithandler.EvaluationResponse
		err     error
		wantErr aiaudit.Error
	}{
		{"invalid response", nil, errors.New("invalid_evaluator_response: bad json"), aiaudit.ErrorInvalidEvaluatorResponse},
		{"transport failure", nil, errors.New("connection reset"), aiaudit.ErrorEvaluatorUnavailable},
		{"nil result without error", nil, nil, aiaudit.ErrorEvaluatorUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)

			m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*message.Message{{ID: g4MsgID1, Content: "hi"}}, nil)
			m.gemini.EXPECT().Evaluate(gomock.Any(), "snapshot prompt", gomock.Any(), "en-US", false).Return(tt.result, nil, tt.err)
			var fin g4Final
			g4ExpectFinal(m, &fin, 1, nil)

			m.h.runAuditJob(context.Background(), g4AuditID, g4Call(aicall.AssistanceTypeAI), g4AIID, "en-US")

			if fin.status != aiaudit.StatusFailed || fin.errStr != string(tt.wantErr) || fin.score != nil || fin.msgIDs != nil {
				t.Errorf("unexpected final write: %+v", fin)
			}
		})
	}
}

func Test_runAuditJob_successStoresScoreEvaluationAndMessageIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	msgs := []*message.Message{
		{Identity: commonidentity.Identity{ID: g4MsgID1}, Role: message.RoleUser, Content: "hello"},
		{Identity: commonidentity.Identity{ID: g4MsgID2}, Role: message.RoleAssistant, ToolCalls: []message.ToolCall{{ID: "tc1"}}},
	}
	m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(msgs, nil)
	raw := json.RawMessage(`{"overall_score":77}`)
	m.gemini.EXPECT().Evaluate(gomock.Any(), "snapshot prompt", "[user]: hello\n[assistant]: <tool_call>\n", "ko-KR", true).
		Return(&geminiaudithandler.EvaluationResponse{OverallScore: 77}, raw, nil)
	var fin g4Final
	g4ExpectFinal(m, &fin, 1, nil)

	m.h.runAuditJob(context.Background(), g4AuditID, g4Call(aicall.AssistanceTypeAI), g4AIID, "ko-KR")

	if fin.status != aiaudit.StatusCompleted || fin.errStr != "" || fin.score == nil || *fin.score != 77 {
		t.Fatalf("unexpected final write: %+v", fin)
	}
	if string(fin.eval) != string(raw) {
		t.Errorf("evaluation mismatch: %s", fin.eval)
	}
	if len(fin.msgIDs) != 2 || fin.msgIDs[0] != g4MsgID1 || fin.msgIDs[1] != g4MsgID2 {
		t.Errorf("message ids mismatch: %v", fin.msgIDs)
	}
}

func Test_runAuditJob_finalWriteOutcomes(t *testing.T) {
	tests := []struct {
		name string
		n    int64
		err  error
	}{
		{"write error", 0, errors.New("write failed")},
		{"no rows updated", 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := g4New(mc)

			ac := g4Call(aicall.AssistanceTypeAI)
			ac.Metadata = nil
			var fin g4Final
			g4ExpectFinal(m, &fin, tt.n, tt.err)

			m.h.runAuditJob(context.Background(), g4AuditID, ac, g4AIID, "en-US")

			if fin.status != aiaudit.StatusFailed {
				t.Errorf("unexpected final write: %+v", fin)
			}
			if len(m.h.semaphore) != 0 {
				t.Errorf("semaphore slot not released")
			}
		})
	}
}

func Test_runAuditJob_evaluatorPanicIsRecovered(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	m.gemini.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, string, string, string, bool) (*geminiaudithandler.EvaluationResponse, json.RawMessage, error) {
			panic("synthetic evaluator panic")
		})
	var fin g4Final
	g4ExpectFinal(m, &fin, 1, nil)

	m.h.runAuditJob(context.Background(), g4AuditID, g4Call(aicall.AssistanceTypeAI), g4AIID, "en-US")

	if fin.status != aiaudit.StatusFailed || fin.errStr != string(aiaudit.ErrorEvaluatorUnavailable) {
		t.Errorf("unexpected final write: %+v", fin)
	}
	if len(m.h.semaphore) != 0 {
		t.Errorf("semaphore slot not released")
	}
}

// ---- Get / List / Delete ----

func Test_Get_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	m.db.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(nil, dbhandler.ErrNotFound)

	res, err := m.h.Get(context.Background(), g4AuditID)
	if res != nil || !errors.Is(err, dbhandler.ErrNotFound) || !strings.Contains(err.Error(), "could not get audit") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_List_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := g4New(mc)

	rpcErr := errors.New("db down")
	m.db.EXPECT().AIAuditList(gomock.Any(), uint64(7), "tok", gomock.Any()).Return(nil, rpcErr)

	res, err := m.h.List(context.Background(), 7, "tok", nil)
	if res != nil || !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), "could not list audits") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Delete_errors(t *testing.T) {
	rpcErr := errors.New("db down")

	t.Run("lookup fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(nil, rpcErr)

		res, err := m.h.Delete(context.Background(), g4AuditID)
		if res != nil || !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), "before delete") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})

	t.Run("delete fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.db.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{}, nil)
		m.db.EXPECT().AIAuditDelete(gomock.Any(), g4AuditID).Return(rpcErr)

		res, err := m.h.Delete(context.Background(), g4AuditID)
		if res != nil || !errors.Is(err, rpcErr) || !strings.Contains(err.Error(), "could not delete audit") {
			t.Errorf("unexpected result: %v, err: %v", res, err)
		}
	})
}

// ---- SweepStaleAudits ----

func Test_SweepStaleAudits(t *testing.T) {
	const cutoff = "2026-01-02 03:04:05.000000"
	filters := map[aiaudit.Field]any{
		aiaudit.FieldStatus:  aiaudit.StatusProgressing,
		aiaudit.FieldDeleted: false,
	}

	t.Run("list fails and nothing is updated", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.util.EXPECT().TimeGetCurTimeAdd(-staleAuditAgeMinutes * time.Minute).Return(cutoff)
		m.db.EXPECT().AIAuditList(gomock.Any(), uint64(1000), cutoff, filters).Return(nil, errors.New("db down"))

		m.h.SweepStaleAudits(context.Background())
	})

	t.Run("nothing stale", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		m.util.EXPECT().TimeGetCurTimeAdd(-staleAuditAgeMinutes * time.Minute).Return(cutoff)
		m.db.EXPECT().AIAuditList(gomock.Any(), uint64(1000), cutoff, filters).Return([]*aiaudit.AIAudit{}, nil)

		m.h.SweepStaleAudits(context.Background())
	})

	t.Run("every stale audit is marked failed even if one update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		m := g4New(mc)
		second := uuid.FromStringOrNil("10000000-0000-4000-8000-00000000000b")
		m.util.EXPECT().TimeGetCurTimeAdd(-staleAuditAgeMinutes * time.Minute).Return(cutoff)
		m.db.EXPECT().AIAuditList(gomock.Any(), uint64(1000), cutoff, filters).Return([]*aiaudit.AIAudit{
			{Identity: commonidentity.Identity{ID: g4AuditID}},
			{Identity: commonidentity.Identity{ID: second}},
		}, nil)
		m.db.EXPECT().AIAuditUpdateFinal(gomock.Any(), g4AuditID, aiaudit.StatusFailed, nil, json.RawMessage(nil), string(aiaudit.ErrorEvaluatorUnavailable), nil).
			Return(int64(0), errors.New("write failed"))
		m.db.EXPECT().AIAuditUpdateFinal(gomock.Any(), second, aiaudit.StatusFailed, nil, json.RawMessage(nil), string(aiaudit.ErrorEvaluatorUnavailable), nil).
			Return(int64(1), nil)

		m.h.SweepStaleAudits(context.Background())
	})
}

// ---- helpers ----

func Test_extractPromptSnapshots_errors(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
		wantErr  string
	}{
		{"value cannot be marshalled", map[string]any{aicall.MetaKeyPromptSnapshots: make(chan int)}, "could not marshal"},
		{"value has the wrong shape", map[string]any{aicall.MetaKeyPromptSnapshots: "not-a-list"}, "could not unmarshal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := extractPromptSnapshots(&aicall.AIcall{Metadata: tt.metadata})
			if res != nil || err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("unexpected result: %v, err: %v", res, err)
			}
		})
	}
}

func Test_extractPromptSnapshots_decodesFields(t *testing.T) {
	member := uuid.FromStringOrNil("10000000-0000-4000-8000-00000000000c")
	res, err := extractPromptSnapshots(&aicall.AIcall{Metadata: map[string]any{
		aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{{AIID: g4AIID, PromptHistoryID: g4HistID, Prompt: "p", MemberID: member}},
	}})
	if err != nil || len(res) != 1 {
		t.Fatalf("unexpected result: %v, err: %v", res, err)
	}
	if res[0].AIID != g4AIID || res[0].PromptHistoryID != g4HistID || res[0].Prompt != "p" || res[0].MemberID != member {
		t.Errorf("decoded snapshot mismatch: %+v", res[0])
	}
}

func Test_buildTranscript_contentLimitsAndFormats(t *testing.T) {
	long := strings.Repeat("a", maxMessageContentLen+10)
	msgs := []*message.Message{
		{Role: message.RoleUser, Content: long},
		{Role: message.RoleTool, ToolCallID: "tc1", Content: "result"},
		{Role: message.RoleAssistant, ToolCalls: []message.ToolCall{{ID: "tc1"}}, Content: "ignored"},
	}
	var truncated bool
	got := buildTranscript(msgs, &truncated)
	want := "[user]: " + strings.Repeat("a", maxMessageContentLen) + "...\n" +
		"[tool]: <tool_result> result\n" +
		"[assistant]: <tool_call>\n"
	if got != want {
		t.Errorf("transcript mismatch:\n got: %q\nwant: %q", got, want)
	}
	if truncated {
		t.Errorf("3 messages must not be flagged as truncated")
	}
}

func Test_buildTranscript_atContentLimitIsNotCut(t *testing.T) {
	exact := strings.Repeat("b", maxMessageContentLen)
	got := buildTranscript([]*message.Message{{Role: message.RoleUser, Content: exact}}, nil)
	if got != "[user]: "+exact+"\n" {
		t.Errorf("content of exactly the limit must be kept intact, got len %d", len(got))
	}
}

func Test_buildTranscript_oneBelowMaxMessagesNotTruncated(t *testing.T) {
	msgs := make([]*message.Message, maxMessages-1)
	for i := range msgs {
		msgs[i] = &message.Message{Role: message.RoleUser, Content: "x"}
	}
	var truncated bool
	buildTranscript(msgs, &truncated)
	if truncated {
		t.Errorf("maxMessages-1 messages must not be flagged as truncated")
	}
}
