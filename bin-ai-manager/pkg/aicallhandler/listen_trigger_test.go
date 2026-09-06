package aicallhandler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	kmkase "monorepo/bin-contact-manager/models/kase"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"
)

// listenTriggerFixtureIDs are the ids every test in this file shares.
var (
	ltAIcallID   = uuid.FromStringOrNil("11110000-0000-4000-8000-000000000001")
	ltCustomerID = uuid.FromStringOrNil("11110000-0000-4000-8000-000000000002")
	ltAIID       = uuid.FromStringOrNil("11110000-0000-4000-8000-000000000003")
	ltCaseID     = uuid.FromStringOrNil("11110000-0000-4000-8000-000000000004")
	ltCallID     = uuid.FromStringOrNil("11110000-0000-4000-8000-000000000005")
	ltConfID     = uuid.FromStringOrNil("11110000-0000-4000-8000-000000000006")
)

// listenEligibleAIcall is the AIcall shape that passes every step of
// checkListenEligible unless a test deliberately breaks one field.
func listenEligibleAIcall() *aicall.AIcall {
	return &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: ltAIcallID, CustomerID: ltCustomerID},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   ltAIID,
		ReferenceType:  aicall.ReferenceTypeContactCase,
		ReferenceID:    ltCaseID,
		Status:         aicall.StatusProgressing,
	}
}

func listenEligibleCase() *kmkase.Case {
	return &kmkase.Case{
		ID:            ltCaseID,
		CustomerID:    ltCustomerID,
		ReferenceType: kmkase.ReferenceTypeCall,
		ReferenceID:   ltCallID.String(),
	}
}

func listenEligibleCall() *cmcall.Call {
	return &cmcall.Call{
		Identity:     commonidentity.Identity{ID: ltCallID, CustomerID: ltCustomerID},
		Status:       cmcall.StatusProgressing,
		ConfbridgeID: ltConfID,
	}
}

// Test_checkListenEligible covers every early return of design §5.1.1 steps
// 1-6. Each row is a distinct way listening must not proceed, and every one of
// them must return proceed=false SYNCHRONOUSLY with ZERO goroutines spawned.
func Test_checkListenEligible(t *testing.T) {
	tests := []struct {
		name string

		aiType       ai.Type
		aicallStatus aicall.Status
		aicallDelete bool
		metadata     map[string]any
		listenCallID uuid.UUID

		responseTranscribe *tmtranscribe.Transcribe
		responseCase       *kmkase.Case
		responseCaseErr    error
		responseCall       *cmcall.Call
		responseCallErr    error

		expectTranscribeGet bool
		expectCaseGet       bool
		expectCallGet       bool
		expectProceed       bool
	}{
		{
			name:         "non-insight AI returns immediately",
			aiType:       ai.TypeNormal,
			aicallStatus: aicall.StatusProgressing,
		},
		{
			// New in design rev 16 (review round 13 finding MEDIUM-2). These
			// two rows are the direct regression test for the public
			// endpoint's loss of the old Start hook's implicit "just created,
			// therefore live" guarantee: ZERO TranscribeV1*/ContactV1*/CallV1*
			// calls, and it must not even reach the idempotency check.
			name:         "terminated aicall is refused before any cross-service RPC",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusTerminated,
		},
		{
			name:         "soft-deleted aicall (TMDelete set) is refused before any cross-service RPC",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			aicallDelete: true,
		},
		{
			name:         "already listening on a valid session makes zero transcribe-start calls",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			metadata: map[string]any{
				aicall.MetaKeyListenTranscribeID: uuid.FromStringOrNil("22220000-0000-4000-8000-000000000001").String(),
			},
			listenCallID: ltCallID,
			responseTranscribe: &tmtranscribe.Transcribe{
				Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("22220000-0000-4000-8000-000000000001")},
				Status:      tmtranscribe.StatusProgressing,
				ReferenceID: ltCallID,
			},
			expectTranscribeGet: true,
		},
		{
			name:         "case reference type is neither call nor conversation_message",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			responseCase: &kmkase.Case{
				ID: ltCaseID, CustomerID: ltCustomerID,
				ReferenceType: "email_thread", ReferenceID: ltCallID.String(),
			},
			expectCaseGet: true,
		},
		{
			name:         "case reference id does not parse as a uuid",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			responseCase: &kmkase.Case{
				ID: ltCaseID, CustomerID: ltCustomerID,
				ReferenceType: kmkase.ReferenceTypeCall, ReferenceID: "not-a-uuid",
			},
			expectCaseGet: true,
		},
		{
			name:         "cross-customer case is refused",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			responseCase: &kmkase.Case{
				ID: ltCaseID, CustomerID: uuid.FromStringOrNil("33330000-0000-4000-8000-000000000001"),
				ReferenceType: kmkase.ReferenceTypeCall, ReferenceID: ltCallID.String(),
			},
			expectCaseGet: true,
		},
		{
			name:         "cross-customer call is refused",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			responseCase: listenEligibleCase(),
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{ID: ltCallID, CustomerID: uuid.FromStringOrNil("33330000-0000-4000-8000-000000000002")},
				Status:   cmcall.StatusProgressing,
			},
			expectCaseGet: true,
			expectCallGet: true,
		},
		{
			name:         "call status hangup is not listenable",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			responseCase: listenEligibleCase(),
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{ID: ltCallID, CustomerID: ltCustomerID},
				Status:   cmcall.StatusHangup,
			},
			expectCaseGet: true,
			expectCallGet: true,
		},
		{
			name:         "case lookup failure does not proceed and does not error",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			// Design §6's first row: a lookup failure is logged and metered,
			// and must never fail the triggering POST.
			responseCaseErr: fmt.Errorf("contact-manager unavailable"),
			expectCaseGet:   true,
		},
		{
			name:            "call lookup failure does not proceed and does not error",
			aiType:          ai.TypeInsight,
			aicallStatus:    aicall.StatusProgressing,
			responseCase:    listenEligibleCase(),
			responseCallErr: fmt.Errorf("call-manager unavailable"),
			expectCaseGet:   true,
			expectCallGet:   true,
		},
		{
			name:         "call status dialing is listenable and proceeds",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			responseCase: listenEligibleCase(),
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{ID: ltCallID, CustomerID: ltCustomerID},
				Status:   cmcall.StatusDialing,
			},
			expectCaseGet: true,
			expectCallGet: true,
			expectProceed: true,
		},
		{
			name:         "call status ringing is listenable and proceeds",
			aiType:       ai.TypeInsight,
			aicallStatus: aicall.StatusProgressing,
			responseCase: listenEligibleCase(),
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{ID: ltCallID, CustomerID: ltCustomerID},
				Status:   cmcall.StatusRinging,
			},
			expectCaseGet: true,
			expectCallGet: true,
			expectProceed: true,
		},
		{
			name:          "call status progressing is listenable and proceeds",
			aiType:        ai.TypeInsight,
			aicallStatus:  aicall.StatusProgressing,
			responseCase:  listenEligibleCase(),
			responseCall:  listenEligibleCall(),
			expectCaseGet: true,
			expectCallGet: true,
			expectProceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetListenDefaultsForTest()

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{reqHandler: mockReq, aiHandler: mockAI}
			ctx := context.Background()

			c := listenEligibleAIcall()
			c.Status = tt.aicallStatus
			c.Metadata = tt.metadata
			c.ListenCallID = tt.listenCallID
			if tt.aicallDelete {
				now := time.Now()
				c.TMDelete = &now
			}

			// Step 2 necessarily calls the IN-PROCESS aiHandler.Get first: it
			// cannot check a.Type without the struct. What must NOT happen on
			// the refusal rows is a cross-service RPC, which the Times()
			// expectations below assert.
			mockAI.EXPECT().Get(ctx, ltAIID).Return(&ai.AI{
				Identity: commonidentity.Identity{ID: ltAIID},
				Type:     tt.aiType,
			}, nil)

			if tt.expectTranscribeGet {
				mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, gomock.Any()).Return(tt.responseTranscribe, nil).Times(1)
			} else {
				mockReq.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), gomock.Any()).Times(0)
			}

			if tt.expectCaseGet {
				mockReq.EXPECT().ContactV1CaseGet(ctx, ltCustomerID, ltCaseID).Return(tt.responseCase, tt.responseCaseErr).Times(1)
			} else {
				mockReq.EXPECT().ContactV1CaseGet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			if tt.expectCallGet {
				mockReq.EXPECT().CallV1CallGet(ctx, ltCallID).Return(tt.responseCall, tt.responseCallErr).Times(1)
			} else {
				mockReq.EXPECT().CallV1CallGet(gomock.Any(), gomock.Any()).Times(0)
			}

			// Nothing in steps 1-6 may ever start a transcribe.
			mockReq.EXPECT().TranscribeV1TranscribeStart(
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			).Times(0)

			a, kase, callID, call, proceed, err := h.checkListenEligible(ctx, c)
			if err != nil {
				// Documented invariant: no branch of checkListenEligible ever
				// returns a non-nil error (design §6's first row).
				t.Fatalf("checkListenEligible must never return an error. got: %v", err)
			}
			if proceed != tt.expectProceed {
				t.Errorf("proceed mismatch. expected: %v, got: %v", tt.expectProceed, proceed)
			}

			if !tt.expectProceed {
				if a != nil || kase != nil || call != nil || callID != uuid.Nil {
					t.Errorf("a refusal must return zero values. a: %v, kase: %v, callID: %s, call: %v", a, kase, callID, call)
				}
				return
			}

			// The proceed path must hand back every value steps 7-8 need, so
			// runListenStart never re-fetches (review round 13 HIGH-1).
			if a == nil || kase == nil || call == nil {
				t.Fatalf("proceed must return a/kase/call. a: %v, kase: %v, call: %v", a, kase, call)
			}
			if callID != ltCallID {
				t.Errorf("callID mismatch. expected: %s, got: %s", ltCallID, callID)
			}
		})
	}
}

// Test_ProcessListen covers the sync/async split itself.
//
// This is the handler-layer surface for the whole trigger -- ONE exported
// method, not two (design rev 16, review round 13 findings HIGH-2/MEDIUM-4).
func Test_ProcessListen(t *testing.T) {
	t.Run("unknown id returns the Get error and never runs checkListenEligible", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockAI := aihandler.NewMockAIHandler(mc)

		h := &aicallHandler{db: mockDB, reqHandler: mockReq, aiHandler: mockAI}
		ctx := context.Background()

		mockDB.EXPECT().AIcallGet(ctx, ltAIcallID).Return(nil, dbhandler.ErrNotFound)
		// checkListenEligible's very first cross-boundary call is aiHandler.Get;
		// zero of them proves the method short-circuited on Get.
		mockAI.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)

		res, err := h.ProcessListen(ctx, ltAIcallID)
		if err == nil {
			t.Fatalf("expected an error for an unknown id")
		}
		if res != nil {
			t.Errorf("expected a nil aicall on error. got: %v", res)
		}
	})

	t.Run("terminated aicall is refused and never runs the async stage", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		mockAI := aihandler.NewMockAIHandler(mc)
		h := &aicallHandler{db: mockDB, aiHandler: mockAI}
		ctx := context.Background()

		// Step 2's liveness half is what refuses here: the AIcall is an Insight
		// AIcall, but it is terminated, so the trigger stops before any
		// cross-service RPC and before the async stage (design rev 16).
		c := listenEligibleAIcall()
		c.Status = aicall.StatusTerminated
		mockDB.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)
		mockAI.EXPECT().Get(ctx, ltAIID).Return(&ai.AI{
			Identity: commonidentity.Identity{ID: ltAIID},
			Type:     ai.TypeInsight,
		}, nil)

		var hookCalls int32
		var mu sync.Mutex
		h.runListenStartHook = func(context.Context, *ai.AI, *aicall.AIcall, *kmkase.Case, uuid.UUID, *cmcall.Call) {
			mu.Lock()
			hookCalls++
			mu.Unlock()
		}

		res, err := h.ProcessListen(ctx, ltAIcallID)
		if err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
		if res != c {
			t.Errorf("ProcessListen must return the AIcall unchanged by steps 1-6")
		}

		// The goroutine is never spawned at all on this path, so a short settle
		// is enough to prove it: any spawn would have to run before the mock
		// controller's Finish below.
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		if hookCalls != 0 {
			t.Errorf("runListenStart must not run when checkListenEligible refuses. calls: %d", hookCalls)
		}
	})

	t.Run("happy path returns immediately, runs the async stage exactly once, and re-fetches nothing", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockAI := aihandler.NewMockAIHandler(mc)

		h := &aicallHandler{db: mockDB, reqHandler: mockReq, aiHandler: mockAI}
		ctx := context.Background()

		c := listenEligibleAIcall()
		kase := listenEligibleCase()
		call := listenEligibleCall()

		mockDB.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)
		mockAI.EXPECT().Get(ctx, ltAIID).Return(&ai.AI{
			Identity: commonidentity.Identity{ID: ltAIID},
			Type:     ai.TypeInsight,
		}, nil)

		// EXACTLY ONE of each, both from the SYNCHRONOUS stage. A re-fetch
		// inside the goroutine would make these Times(1) expectations fail --
		// this is the direct regression test for review round 13's HIGH-1.
		mockReq.EXPECT().ContactV1CaseGet(ctx, ltCustomerID, ltCaseID).Return(kase, nil).Times(1)
		mockReq.EXPECT().CallV1CallGet(ctx, ltCallID).Return(call, nil).Times(1)

		done := make(chan struct {
			a      *ai.AI
			c      *aicall.AIcall
			kase   *kmkase.Case
			callID uuid.UUID
			call   *cmcall.Call
		}, 4)

		h.runListenStartHook = func(_ context.Context, a *ai.AI, gotC *aicall.AIcall, gotKase *kmkase.Case, gotCallID uuid.UUID, gotCall *cmcall.Call) {
			// Stand in for step 7's up-to-30s confbridge wait: if ProcessListen
			// blocked on the goroutine, the assertion below its own return
			// could not run before this sleep elapsed.
			time.Sleep(150 * time.Millisecond)
			done <- struct {
				a      *ai.AI
				c      *aicall.AIcall
				kase   *kmkase.Case
				callID uuid.UUID
				call   *cmcall.Call
			}{a, gotC, gotKase, gotCallID, gotCall}
		}

		start := time.Now()
		res, err := h.ProcessListen(ctx, ltAIcallID)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
		if res != c {
			t.Errorf("ProcessListen must return the AIcall unchanged by steps 1-6")
		}

		// THE SINGLE MOST SAFETY-CRITICAL PROPERTY OF THE SPLIT (design §5.1,
		// §7 item 2): ProcessListen must not block on step 7's wait. The
		// assertion is against the hook's own deliberate delay, not a wall
		// clock budget of its own.
		if elapsed >= 150*time.Millisecond {
			t.Errorf("ProcessListen blocked on the detached stage. elapsed: %v", elapsed)
		}

		select {
		case got := <-done:
			if got.c != c || got.kase != kase || got.call != call || got.callID != ltCallID {
				t.Errorf("runListenStart must receive the already-resolved values verbatim. got: %+v", got)
			}
			if got.a == nil {
				t.Errorf("runListenStart must receive the resolved ai")
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("runListenStart was never invoked")
		}

		// Exactly once.
		select {
		case <-done:
			t.Errorf("runListenStart ran more than once")
		case <-time.After(100 * time.Millisecond):
		}
	})

	t.Run("repeated calls on an already-listening aicall are free", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockAI := aihandler.NewMockAIHandler(mc)

		h := &aicallHandler{db: mockDB, reqHandler: mockReq, aiHandler: mockAI}
		ctx := context.Background()

		transcribeID := uuid.FromStringOrNil("44440000-0000-4000-8000-000000000001")

		listening := listenEligibleAIcall()
		listening.ListenCallID = ltCallID
		listening.Metadata = map[string]any{
			aicall.MetaKeyListenTranscribeID:   transcribeID.String(),
			aicall.MetaKeyListenOwnsTranscribe: true,
		}

		mockDB.EXPECT().AIcallGet(ctx, ltAIcallID).Return(listening, nil).Times(2)
		mockAI.EXPECT().Get(ctx, ltAIID).Return(&ai.AI{
			Identity: commonidentity.Identity{ID: ltAIID},
			Type:     ai.TypeInsight,
		}, nil).Times(2)
		mockReq.EXPECT().TranscribeV1TranscribeGet(ctx, transcribeID).Return(&tmtranscribe.Transcribe{
			Identity:    commonidentity.Identity{ID: transcribeID},
			Status:      tmtranscribe.StatusProgressing,
			ReferenceID: ltCallID,
		}, nil).Times(2)

		// The short-circuit costs nothing beyond the transcribe read: no Case
		// lookup, no call lookup, and above all no transcribe start.
		mockReq.EXPECT().ContactV1CaseGet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		mockReq.EXPECT().CallV1CallGet(gomock.Any(), gomock.Any()).Times(0)
		mockReq.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Times(0)

		var mu sync.Mutex
		hookCalls := 0
		h.runListenStartHook = func(context.Context, *ai.AI, *aicall.AIcall, *kmkase.Case, uuid.UUID, *cmcall.Call) {
			mu.Lock()
			hookCalls++
			mu.Unlock()
		}

		for i := 0; i < 2; i++ {
			if _, err := h.ProcessListen(ctx, ltAIcallID); err != nil {
				t.Fatalf("unexpected error on call %d. err: %v", i, err)
			}
		}

		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		if hookCalls != 0 {
			t.Errorf("an already-listening aicall must not spawn the async stage. calls: %d", hookCalls)
		}
	})
}

// startListenHarness bundles the mocks startListenTranscribe needs.
type startListenHarness struct {
	req   *requesthandler.MockRequestHandler
	db    *dbhandler.MockDBHandler
	cache *cachehandler.MockCacheHandler
	util  *utilhandler.MockUtilHandler
	h     *aicallHandler
}

func newStartListenHarness(mc *gomock.Controller) *startListenHarness {
	req := requesthandler.NewMockRequestHandler(mc)
	db := dbhandler.NewMockDBHandler(mc)
	cache := cachehandler.NewMockCacheHandler(mc)
	util := utilhandler.NewMockUtilHandler(mc)

	return &startListenHarness{
		req:   req,
		db:    db,
		cache: cache,
		util:  util,
		h: &aicallHandler{
			reqHandler:  req,
			db:          db,
			cache:       cache,
			utilHandler: util,
		},
	}
}

// expectListenLivenessRecheck wires the fresh, under-the-lock AIcall re-read
// startListenTranscribe performs before its speculative pre-write (review round
// 1 finding HIGH-1).
//
// IT IS THE CACHE-SKIPPING VARIANT, and the distinct method name is the whole
// assertion (review round 2 finding MEDIUM-2): the writer that matters here --
// ProcessTerminate -- is outside this feature and refreshes the aicall cache
// only best-effort, so a cache-first read could return a stale `progressing`
// and defeat the check. Expecting AIcallGetSkipCache means a regression back to
// AIcallGet fails these tests as an unexpected call, not merely as a changed
// comment.
func (m *startListenHarness) expectListenLivenessRecheck(ctx context.Context, cur *aicall.AIcall) *gomock.Call {
	return m.db.EXPECT().AIcallGetSkipCache(ctx, ltAIcallID).Return(cur, nil)
}

// listenReusableRow builds a transcribe row that PASSES
// pickReusableListenTranscribe's ownership re-verification -- the platform
// sentinel customer id, this call's id as the reference, not deleted. Tests
// asserting reuse must use this rather than a bare id, because a bare id is now
// (correctly) rejected as unverifiable.
func listenReusableRow(id uuid.UUID) tmtranscribe.Transcribe {
	return tmtranscribe.Transcribe{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: cmcustomer.IDAIManagerListen,
		},
		ReferenceType: tmtranscribe.ReferenceTypeCall,
		ReferenceID:   ltCallID,
		Status:        tmtranscribe.StatusProgressing,
	}
}

// expectUpdateListenState wires the DB+Redis pair UpdateListenState performs
// for one write, returning nothing -- callers add ordering constraints
// themselves where the ORDER is what is under test.
func (m *startListenHarness) expectUpdateListenState(ctx context.Context, cur *aicall.AIcall, transcribeID uuid.UUID) (*gomock.Call, *gomock.Call) {
	get := m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(cur, nil)
	update := m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)
	add := m.cache.EXPECT().ListenAIcallIDAdd(ctx, transcribeID, ltAIcallID, listenResolverTTL).Return(nil)
	_ = get
	return update, add
}

// Test_runListenStart_EventOrdering pins design §5.2.2/§5.2.4's ordering fix.
//
// The whole point is that the DB write and the Redis SADD land BEFORE the
// transcribe exists, so the session cannot emit events for a listener nobody
// has registered yet.
func Test_runListenStart_EventOrdering(t *testing.T) {
	newTranscribeID := uuid.FromStringOrNil("55550000-0000-4000-8000-000000000001")
	lockToken := uuid.FromStringOrNil("55550000-0000-4000-8000-0000000000ff")
	winnerID := uuid.FromStringOrNil("55550000-0000-4000-8000-000000000002")

	alreadyProgressing := &cerrors.VoipbinError{Reason: transcribeReasonAlreadyProgressing}

	t.Run("the speculative pre-write lands before TranscribeV1TranscribeStart", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)
		m.expectListenLivenessRecheck(ctx, c)
		m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(nil, nil)
		m.util.EXPECT().UUIDCreate().Return(newTranscribeID)

		update, add := m.expectUpdateListenState(ctx, c, newTranscribeID)
		start := m.req.EXPECT().TranscribeV1TranscribeStart(
			ctx, newTranscribeID, cmcustomer.IDAIManagerListen, gomock.Any(), uuid.Nil,
			tmtranscribe.ReferenceTypeCall, ltCallID, gomock.Any(),
			tmtranscribe.DirectionBoth, tmtranscribe.ProviderEmpty, defaultListenTranscribeStartTimeout,
		).Return(&tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: newTranscribeID}}, nil)

		// Call ORDER, not merely "both eventually happened".
		gomock.InOrder(update, add, start)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
		if res != "started" {
			t.Errorf("result mismatch. expected: started, got: %s", res)
		}
	})

	t.Run("pre-write failure never reaches TranscribeV1TranscribeStart", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)
		m.expectListenLivenessRecheck(ctx, c)
		m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(nil, nil)
		m.util.EXPECT().UUIDCreate().Return(newTranscribeID)

		// UpdateListenState's own fresh read fails.
		m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(nil, fmt.Errorf("db down"))

		// The pre-write failure now ROLLS BACK rather than assuming there is
		// nothing to undo (review round 1 finding MEDIUM-4): the DB row is
		// written before the Redis SADD, so a SADD failure genuinely leaves
		// state behind.
		m.cache.EXPECT().ListenAIcallIDRemove(ctx, newTranscribeID, ltAIcallID).Return(nil)
		m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(nil, fmt.Errorf("db down"))

		// Fail closed: nothing was created, so nothing is billed.
		m.req.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Times(0)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err == nil {
			t.Fatalf("expected an error")
		}
		if res != "failed" {
			t.Errorf("result mismatch. expected: failed, got: %s", res)
		}
	})

	t.Run("a non-already-progressing start failure rolls back the pre-generated id", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)
		m.expectListenLivenessRecheck(ctx, c)
		m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(nil, nil)
		m.util.EXPECT().UUIDCreate().Return(newTranscribeID)
		m.expectUpdateListenState(ctx, c, newTranscribeID)

		m.req.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(nil, fmt.Errorf("transcribe-manager unavailable"))

		// rollbackListenState, against the pre-generated id specifically.
		m.cache.EXPECT().ListenAIcallIDRemove(ctx, newTranscribeID, ltAIcallID).Return(nil)
		m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)
		m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err == nil {
			t.Fatalf("expected an error")
		}
		if res != "failed" {
			t.Errorf("result mismatch. expected: failed, got: %s", res)
		}
	})

	t.Run("already-progressing with a winner rewrites state and does NOT roll back", func(t *testing.T) {
		// The direct regression test for review round 13's MEDIUM-3: an
		// earlier draft rolled back and gave up here, silently dropping the
		// reuse-on-conflict behaviour design §6 promises.
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)

		gomock.InOrder(
			m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(nil, nil),
			m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(
				[]tmtranscribe.Transcribe{listenReusableRow(winnerID)}, nil),
		)
		m.util.EXPECT().UUIDCreate().Return(newTranscribeID)

		// One liveness re-read under the lock (cache-skipping), then the
		// pre-write against our own speculative id, then a second write against
		// the WINNER's id with owns=false -- the latter two each doing
		// UpdateListenState's own ordinary cache-first read.
		m.expectListenLivenessRecheck(ctx, c)
		m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil).Times(2)
		m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil).Times(2)
		m.cache.EXPECT().ListenAIcallIDAdd(ctx, newTranscribeID, ltAIcallID, listenResolverTTL).Return(nil)
		m.cache.EXPECT().ListenAIcallIDAdd(ctx, winnerID, ltAIcallID, listenResolverTTL).Return(nil)

		m.req.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(nil, alreadyProgressing)

		// Only OUR never-created speculative membership is removed -- never
		// the winner's.
		m.cache.EXPECT().ListenAIcallIDRemove(ctx, newTranscribeID, ltAIcallID).Return(nil).Times(1)
		m.cache.EXPECT().ListenAIcallIDRemove(ctx, winnerID, gomock.Any()).Times(0)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
		if res != "reused" {
			t.Errorf("result mismatch. expected: reused, got: %s", res)
		}
	})

	t.Run("already-progressing with an empty re-run list rolls back and gives up", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)
		m.expectListenLivenessRecheck(ctx, c)
		m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(nil, nil).Times(2)
		m.util.EXPECT().UUIDCreate().Return(newTranscribeID)
		m.expectUpdateListenState(ctx, c, newTranscribeID)

		m.req.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(nil, alreadyProgressing)

		m.cache.EXPECT().ListenAIcallIDRemove(ctx, newTranscribeID, ltAIcallID).Return(nil)
		m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)
		m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err == nil {
			t.Fatalf("expected an error")
		}
		if res != "failed" {
			t.Errorf("result mismatch. expected: failed, got: %s", res)
		}
	})

	t.Run("an early transcript_created between pre-write and start resolves normally", func(t *testing.T) {
		// The direct regression test for review round 13's HIGH-3: because the
		// pre-write puts listen_transcribe_id on the row BEFORE the transcribe
		// exists, an event arriving in that window finds the metadata already
		// set and is processed as an ordinary segment -- it must NOT read as
		// skipped_invalid and tear the just-created state back down.
		c := listenEligibleAIcall()
		c.ListenCallID = ltCallID
		c.Metadata = map[string]any{
			aicall.MetaKeyListenTranscribeID:   newTranscribeID.String(),
			aicall.MetaKeyListenOwnsTranscribe: true,
		}

		if got := listenTranscribeIDFromMetadata(c); got != newTranscribeID {
			t.Errorf("the pre-written listen_transcribe_id must be readable during the window. expected: %s, got: %s", newTranscribeID, got)
		}
		if !listenOwnsTranscribeFromMetadata(c) {
			t.Errorf("the pre-written ownership flag must be readable during the window")
		}
	})
}

// Test_runListenStart_StartLock pins the per-AIcall create-or-reuse lock
// (design §5.2.2, §7 item 2). Every case below is a regression test for a
// specific defect a review round found in an earlier version of this lock.
func Test_runListenStart_StartLock(t *testing.T) {
	lockToken := uuid.FromStringOrNil("66660000-0000-4000-8000-0000000000ff")

	t.Run("a second concurrent attempt for the same aicall stands down untouched", func(t *testing.T) {
		// The direct regression test for review round 14's HIGH-2 clobbering
		// scenario: the loser must make ZERO TranscribeV1TranscribeStart and
		// ZERO UpdateListenState calls of its own.
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(false, nil)

		// Nothing else at all: no release (we never held it), no list, no
		// start, no state write.
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		m.req.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Times(0)
		m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err != nil {
			t.Fatalf("a held lock is not an error. err: %v", err)
		}
		if res != "skipped_start_locked" {
			t.Errorf("result mismatch. expected: skipped_start_locked, got: %s", res)
		}
	})

	t.Run("release runs on a context detached from an already-cancelled ctx", func(t *testing.T) {
		// The direct regression test for round 16's MEDIUM-2 -- a release still
		// keyed off the cancelled ctx would silently no-op.
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		c := listenEligibleAIcall()

		ctx, cancel := context.WithCancel(context.Background())

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.expectListenLivenessRecheck(ctx, c)
		m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).
			DoAndReturn(func(context.Context, string, uint64, map[tmtranscribe.Field]any) ([]tmtranscribe.Transcribe, error) {
				// The goroutine reaches its own outer timeout while still
				// finishing legitimate work -- the exact case the detached
				// release exists for.
				cancel()
				return nil, fmt.Errorf("aborted")
			})

		// Both properties are captured AT CALL TIME. Inspecting the context
		// after startListenTranscribe returns would be meaningless: the
		// release's own deferred cancel() has fired by then, so it is Done for
		// a reason that has nothing to do with the goroutine's ctx.
		called := false
		sameAsCallerCtx := false
		doneAtCallTime := false
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).
			DoAndReturn(func(rc context.Context, _ uuid.UUID, _ string) error {
				called = true
				sameAsCallerCtx = rc == ctx
				select {
				case <-rc.Done():
					doneAtCallTime = true
				default:
				}
				return nil
			})

		_, _ = m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)

		if !called {
			t.Fatalf("ListenStartLockRelease was never called")
		}
		if sameAsCallerCtx {
			t.Errorf("the release context must not be the goroutine's own ctx")
		}
		if doneAtCallTime {
			t.Errorf("the release context must not already be Done while the goroutine's own ctx is cancelled -- a release keyed off ctx would silently no-op (round 16 MEDIUM-2)")
		}
		// And the caller's own ctx really was cancelled, so the assertion above
		// is testing the case it claims to.
		select {
		case <-ctx.Done():
		default:
			t.Errorf("the test did not actually cancel the goroutine ctx")
		}
	})

	t.Run("acquire error fails closed and still attempts a best-effort release", func(t *testing.T) {
		// Round 17 B-7. The best-effort release itself failing must NOT change
		// the returned error -- the ORIGINAL acquire error is what surfaces.
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		acquireErr := fmt.Errorf("redis timeout")

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(false, acquireErr)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).
			Return(fmt.Errorf("release failed too")).Times(1)

		m.req.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Times(0)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if res != "failed" {
			t.Errorf("result mismatch. expected: failed, got: %s", res)
		}
		if err != acquireErr {
			t.Errorf("the ORIGINAL acquire error must surface, not the release error. got: %v", err)
		}
	})

	t.Run("a deferred-release error is swallowed and not separately metered", func(t *testing.T) {
		// Round 17 B-6 / round 16 MEDIUM-4: distinguishable from the
		// acquire-error path above, which DOES fail.
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		newTranscribeID := uuid.FromStringOrNil("66660000-0000-4000-8000-000000000010")

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).
			Return(fmt.Errorf("release failed"))
		m.expectListenLivenessRecheck(ctx, c)
		m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(nil, nil)
		m.util.EXPECT().UUIDCreate().Return(newTranscribeID)
		m.expectUpdateListenState(ctx, c, newTranscribeID)
		m.req.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(&tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: newTranscribeID}}, nil)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err != nil {
			t.Errorf("a deferred-release error must not propagate. err: %v", err)
		}
		if res != "started" {
			t.Errorf("result mismatch. expected: started, got: %s", res)
		}
	})

	t.Run("compare-and-delete semantics: a stale token release is a no-op", func(t *testing.T) {
		// The lock's release is an atomic compare-and-delete keyed on the
		// caller's own token (round 15 HIGH-1(b), re-verified round 16). This
		// asserts the CONTRACT the handler relies on directly at the release
		// layer: the token this goroutine passes is always its OWN, so a
		// goroutine whose TTL lapsed can never delete a different holder's
		// still-live lock.
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		otherToken := uuid.FromStringOrNil("66660000-0000-4000-8000-000000000099")

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.expectListenLivenessRecheck(ctx, c)
		m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(nil, fmt.Errorf("stop here"))

		// Never the other goroutine's token -- that is what makes the
		// compare-and-delete a no-op rather than a clobber.
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, otherToken.String()).Times(0)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil).Times(1)

		_, _ = m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
	})

	t.Run("simulated crash: the lock is held for its full TTL, then acquirable again", func(t *testing.T) {
		// The defer never runs at all (pod loss), so the key survives on its
		// TTL alone. Uses SetAIcallListenStartLockTTLForTest rather than
		// waiting 60 real seconds. Round 16 MEDIUM-3 corrected the earlier
		// claim that any later goroutine could just acquire: it can only do so
		// once the TTL has ACTUALLY elapsed.
		config.SetListenDefaultsForTest()
		config.SetAIcallListenStartLockTTLForTest(1)
		defer config.SetListenDefaultsForTest()

		if got := config.Get().AIcallListenStartLockTTLSeconds; got != 1 {
			t.Fatalf("the TTL override did not take. got: %d", got)
		}

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		beforeToken := uuid.FromStringOrNil("66660000-0000-4000-8000-000000000021")
		afterToken := uuid.FromStringOrNil("66660000-0000-4000-8000-000000000022")

		// A goroutine attempting Acquire BEFORE the window elapses observes
		// acquired=false, and that is skipped_start_locked, not an error.
		m.util.EXPECT().UUIDCreate().Return(beforeToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, beforeToken.String(), 1*time.Second).Return(false, nil)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err != nil || res != "skipped_start_locked" {
			t.Fatalf("before the TTL elapses. expected: skipped_start_locked/nil, got: %s/%v", res, err)
		}

		// One attempting AFTER it elapses acquires normally and proceeds.
		m.util.EXPECT().UUIDCreate().Return(afterToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, afterToken.String(), 1*time.Second).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, afterToken.String()).Return(nil)
		m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).
			Return([]tmtranscribe.Transcribe{listenReusableRow(uuid.FromStringOrNil("66660000-0000-4000-8000-000000000031"))}, nil)
		// One liveness re-read under the lock (cache-skipping), plus
		// UpdateListenState's own ordinary cache-first read.
		m.expectListenLivenessRecheck(ctx, c)
		m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)
		m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)
		m.cache.EXPECT().ListenAIcallIDAdd(ctx, gomock.Any(), ltAIcallID, listenResolverTTL).Return(nil)

		res, err = m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err != nil || res != "reused" {
			t.Fatalf("after the TTL elapses. expected: reused/nil, got: %s/%v", res, err)
		}
	})
}

// Test_UpdateListenState_OwnsMerge pins the SCOPED owns-merge rule
// (design §5.2.4, review round 14 finding HIGH-1).
func Test_UpdateListenState_OwnsMerge(t *testing.T) {
	oldID := uuid.FromStringOrNil("77770000-0000-4000-8000-000000000001")
	newID := uuid.FromStringOrNil("77770000-0000-4000-8000-000000000002")

	tests := []struct {
		name string

		// The row as UpdateListenState's OWN fresh read sees it -- deliberately
		// different from anything the caller passes around (review round 15
		// finding LOW-7).
		currentMetadata map[string]any

		writeTranscribeID uuid.UUID
		writeOwns         bool

		expectSremOldID bool
		expectOwns      bool
	}{
		{
			name: "same id, owns=false after a prior owns=true, merges to TRUE",
			currentMetadata: map[string]any{
				aicall.MetaKeyListenTranscribeID:   oldID.String(),
				aicall.MetaKeyListenOwnsTranscribe: true,
			},
			writeTranscribeID: oldID,
			writeOwns:         false,
			expectSremOldID:   false,
			expectOwns:        true,
		},
		{
			name: "DIFFERENT id, owns=false after a prior owns=true, stays FALSE",
			// The direct regression test for HIGH-1: a stale carried-forward
			// owns=true would make this AIcall believe it owns a session it
			// fell back away from, and design §5.7.2's stop path would then
			// tear down another Case's still-live session.
			currentMetadata: map[string]any{
				aicall.MetaKeyListenTranscribeID:   oldID.String(),
				aicall.MetaKeyListenOwnsTranscribe: true,
			},
			writeTranscribeID: newID,
			writeOwns:         false,
			expectSremOldID:   true,
			expectOwns:        false,
		},
		{
			name: "same id, owns=true is written as true",
			currentMetadata: map[string]any{
				aicall.MetaKeyListenTranscribeID:   oldID.String(),
				aicall.MetaKeyListenOwnsTranscribe: false,
			},
			writeTranscribeID: oldID,
			writeOwns:         true,
			expectSremOldID:   false,
			expectOwns:        true,
		},
		{
			name:              "no prior id: nothing to SREM",
			currentMetadata:   map[string]any{"prompt_snapshots": []any{}},
			writeTranscribeID: newID,
			writeOwns:         true,
			expectSremOldID:   false,
			expectOwns:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			m := newStartListenHarness(mc)
			ctx := context.Background()

			// The row the FRESH read returns. The caller's in-hand copy below
			// deliberately carries different metadata, so a merge decision
			// made from the caller's copy would produce the wrong answer.
			fresh := listenEligibleAIcall()
			fresh.Metadata = tt.currentMetadata

			m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(fresh, nil)

			if tt.expectSremOldID {
				m.cache.EXPECT().ListenAIcallIDRemove(ctx, oldID, ltAIcallID).Return(nil).Times(1)
			} else {
				m.cache.EXPECT().ListenAIcallIDRemove(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			var wroteFields map[aicall.Field]any
			write := m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).
				DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[aicall.Field]any) error {
					wroteFields = fields
					return nil
				})
			add := m.cache.EXPECT().ListenAIcallIDAdd(ctx, tt.writeTranscribeID, ltAIcallID, listenResolverTTL).Return(nil)
			gomock.InOrder(write, add)

			res, err := m.h.UpdateListenState(ctx, ltAIcallID, ltCallID, tt.writeTranscribeID, tt.writeOwns)
			if err != nil {
				t.Fatalf("unexpected error. err: %v", err)
			}

			metadata, ok := wroteFields[aicall.FieldMetadata].(map[string]any)
			if !ok {
				t.Fatalf("the write must carry a metadata map. got: %v", wroteFields[aicall.FieldMetadata])
			}
			if got := metadata[aicall.MetaKeyListenOwnsTranscribe]; got != tt.expectOwns {
				t.Errorf("owns mismatch. expected: %v, got: %v", tt.expectOwns, got)
			}
			if got := metadata[aicall.MetaKeyListenTranscribeID]; got != tt.writeTranscribeID.String() {
				t.Errorf("transcribe id mismatch. expected: %s, got: %v", tt.writeTranscribeID, got)
			}
			if got := wroteFields[aicall.FieldListenCallID]; got != ltCallID {
				t.Errorf("listen_call_id mismatch. expected: %s, got: %v", ltCallID, got)
			}
			// Every other metadata key survives -- FieldMetadata is a
			// whole-column write.
			for k := range tt.currentMetadata {
				if k == aicall.MetaKeyListenTranscribeID || k == aicall.MetaKeyListenOwnsTranscribe {
					continue
				}
				if _, present := metadata[k]; !present {
					t.Errorf("metadata key %q was destroyed by the write", k)
				}
			}
			if res.ListenCallID != ltCallID {
				t.Errorf("returned ListenCallID mismatch. expected: %s, got: %s", ltCallID, res.ListenCallID)
			}
		})
	}
}

// confbridgePoll is one scripted poll iteration for Test_waitForConfbridgeReady.
type confbridgePoll struct {
	callStatus cmcall.Status
	callErr    error
	callDelete bool

	confbridgeID uuid.UUID // uuid.Nil means "not yet bridged"
	confErr      error
	channelCount int
	confDeleted  bool
	confStatus   cmconfbridge.Status
}

// Test_waitForConfbridgeReady covers the bounded retry design §5.1.1 step 7
// added in rev 11-14, and specifically its rev-12 fix for review round 10's
// finding HIGH-A.
//
// It asserts BOTH return values. The second -- the last observed party count --
// is what design §6 requires in the NOT-READY give-up branch's log line, and it
// is the only thing that distinguishes a stuck-at-1 timeout from a stuck-at-3
// one, since both deliberately share the skipped_confbridge_not_ready label.
func Test_waitForConfbridgeReady(t *testing.T) {
	ok2 := confbridgePoll{callStatus: cmcall.StatusProgressing, confbridgeID: ltConfID, channelCount: 2, confStatus: cmconfbridge.StatusProgressing}
	ringing1 := confbridgePoll{callStatus: cmcall.StatusRinging, confbridgeID: ltConfID, channelCount: 1, confStatus: cmconfbridge.StatusProgressing}
	unbridged := confbridgePoll{callStatus: cmcall.StatusRinging}
	three := confbridgePoll{callStatus: cmcall.StatusProgressing, confbridgeID: ltConfID, channelCount: 3, confStatus: cmconfbridge.StatusProgressing}

	tests := []struct {
		name string

		polls []confbridgePoll

		// maxWaitSeconds is compressed to 0 on the rows whose whole point is
		// the GIVE-UP branch: with a zero budget the deadline check fires
		// immediately after the first poll, so exactly one poll runs and both
		// the outcome and the last-observed count are asserted with no
		// wall-clock dependence at all. That the function genuinely RE-POLLS
		// rather than checking once is proven separately, by the multi-poll
		// rows below, which carry a real budget.
		maxWaitSeconds int

		expectResult         confbridgeReadyResult
		expectLastPartyCount int
	}{
		{
			name:                 "already 2 parties on the first poll -- zero extra polls",
			polls:                []confbridgePoll{ok2},
			maxWaitSeconds:       10,
			expectResult:         confbridgeReady,
			expectLastPartyCount: 2,
		},
		{
			name: "ringing, then answers: 1 party for several polls, then 2 -- must actually re-poll",
			// The direct regression test for review round 9's BLOCKING-1: a
			// one-shot check on this exact sequence would silently never listen.
			polls:                []confbridgePoll{ringing1, ringing1, ringing1, ok2},
			maxWaitSeconds:       10,
			expectResult:         confbridgeReady,
			expectLastPartyCount: 2,
		},
		{
			name:                 "confbridge not yet assigned (still queued), then resolves to 2 parties",
			polls:                []confbridgePoll{unbridged, unbridged, ok2},
			maxWaitSeconds:       10,
			expectResult:         confbridgeReady,
			expectLastPartyCount: 2,
		},
		{
			name: "wait budget exhausted while ConfbridgeID never resolves -- last observed count is -1",
			// Pins that "never bridged" stays distinguishable from "bridged but
			// empty" in the give-up log line (design §6).
			polls:                []confbridgePoll{unbridged},
			expectResult:         confbridgeNotReady,
			expectLastPartyCount: -1,
		},
		{
			name:                 "wait budget exhausted while still stuck at 1 party",
			polls:                []confbridgePoll{ringing1},
			expectResult:         confbridgeNotReady,
			expectLastPartyCount: 1,
		},
		{
			name: "3 parties, then settles back to 2 within the budget -- proceeds, does NOT give up",
			// The direct regression test for review round 10's HIGH-A: an
			// earlier version fast-failed on exactly this sequence, breaking
			// the early_media multi-destination connect scenario.
			polls:                []confbridgePoll{three, three, ok2},
			maxWaitSeconds:       10,
			expectResult:         confbridgeReady,
			expectLastPartyCount: 2,
		},
		{
			name: "3 parties for the whole budget -- times out exactly like a stuck 1-party count",
			// This design deliberately does NOT have a separate "invalid
			// topology" result. The last-observed count of 3 in the give-up log
			// line is the ONLY thing distinguishing it from the stuck-at-1 row.
			polls:                []confbridgePoll{three},
			expectResult:         confbridgeNotReady,
			expectLastPartyCount: 3,
		},
		{
			name: "call ends mid-poll after a 1-party read -- last observed count is 1, not reset",
			polls: []confbridgePoll{
				ringing1,
				{callStatus: cmcall.StatusHangup},
			},
			maxWaitSeconds:       10,
			expectResult:         confbridgeCallEnded,
			expectLastPartyCount: 1,
		},
		{
			name:                 "CallV1ConfbridgeGet errors on the first poll -- last observed count is -1",
			polls:                []confbridgePoll{{callStatus: cmcall.StatusProgressing, confbridgeID: ltConfID, confErr: fmt.Errorf("boom")}},
			maxWaitSeconds:       10,
			expectResult:         confbridgeError,
			expectLastPartyCount: -1,
		},
		{
			name: "CallV1ConfbridgeGet errors after a 1-party read -- last observed count is 1, not reset",
			polls: []confbridgePoll{
				ringing1,
				{callStatus: cmcall.StatusProgressing, confbridgeID: ltConfID, confErr: fmt.Errorf("boom")},
			},
			maxWaitSeconds:       10,
			expectResult:         confbridgeError,
			expectLastPartyCount: 1,
		},
		{
			name:                 "CallV1CallGet errors on the first poll",
			polls:                []confbridgePoll{{callErr: fmt.Errorf("boom")}},
			maxWaitSeconds:       10,
			expectResult:         confbridgeError,
			expectLastPartyCount: -1,
		},
		{
			name: "confbridge exists but TMDelete is set -- treated as not-ready, not as an error",
			polls: []confbridgePoll{
				{callStatus: cmcall.StatusProgressing, confbridgeID: ltConfID, channelCount: 2, confStatus: cmconfbridge.StatusProgressing, confDeleted: true},
			},
			expectResult:         confbridgeNotReady,
			expectLastPartyCount: 2,
		},
		{
			name: "confbridge Status is not progressing -- treated as not-ready, not as an error",
			polls: []confbridgePoll{
				{callStatus: cmcall.StatusProgressing, confbridgeID: ltConfID, channelCount: 2, confStatus: cmconfbridge.Status("terminating")},
			},
			expectResult:         confbridgeNotReady,
			expectLastPartyCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Zero poll interval plus the per-row budget above keeps every row
			// fast AND deterministic.
			config.SetListenDefaultsForTest()
			config.SetAIcallListenConfbridgeReadyPollIntervalForTest(0)
			config.SetAIcallListenConfbridgeReadyMaxWaitForTest(tt.maxWaitSeconds)
			defer config.SetListenDefaultsForTest()

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}
			ctx := context.Background()

			var callCalls []*gomock.Call
			var confCalls []*gomock.Call
			for _, p := range tt.polls {
				poll := p

				if poll.callErr != nil {
					callCalls = append(callCalls, mockReq.EXPECT().CallV1CallGet(ctx, ltCallID).Return(nil, poll.callErr))
					continue
				}

				call := &cmcall.Call{
					Identity:     commonidentity.Identity{ID: ltCallID, CustomerID: ltCustomerID},
					Status:       poll.callStatus,
					ConfbridgeID: poll.confbridgeID,
				}
				if poll.callDelete {
					now := time.Now()
					call.TMDelete = &now
				}
				callCalls = append(callCalls, mockReq.EXPECT().CallV1CallGet(ctx, ltCallID).Return(call, nil))

				if poll.confbridgeID == uuid.Nil {
					continue
				}
				if poll.confErr != nil {
					confCalls = append(confCalls, mockReq.EXPECT().CallV1ConfbridgeGet(ctx, ltConfID).Return(nil, poll.confErr))
					continue
				}

				cb := &cmconfbridge.Confbridge{
					Identity:       commonidentity.Identity{ID: ltConfID},
					Status:         poll.confStatus,
					ChannelCallIDs: map[string]uuid.UUID{},
				}
				for i := 0; i < poll.channelCount; i++ {
					cb.ChannelCallIDs[fmt.Sprintf("ch-%d", i)] = uuid.Must(uuid.NewV4())
				}
				if poll.confDeleted {
					now := time.Now()
					cb.TMDelete = &now
				}
				confCalls = append(confCalls, mockReq.EXPECT().CallV1ConfbridgeGet(ctx, ltConfID).Return(cb, nil))
			}
			orderCalls(callCalls)
			orderCalls(confCalls)

			gotResult, gotCount := h.waitForConfbridgeReady(ctx, ltCallID)
			if gotResult != tt.expectResult {
				t.Errorf("result mismatch. expected: %d, got: %d", tt.expectResult, gotResult)
			}
			if gotCount != tt.expectLastPartyCount {
				t.Errorf("last party count mismatch. expected: %d, got: %d", tt.expectLastPartyCount, gotCount)
			}
		})
	}
}

// orderCalls applies gomock.InOrder to a slice of calls. gomock.InOrder takes
// ...any, so a []*gomock.Call cannot be spread into it directly.
func orderCalls(calls []*gomock.Call) {
	for i := 1; i < len(calls); i++ {
		calls[i].After(calls[i-1])
	}
}

// Test_startListenTranscribe_LivenessRecheck is the direct regression test for
// review round 1's HIGH-1.
//
// checkListenEligible's step 2 checks Status/TMDelete BEFORE
// waitForConfbridgeReady, which can block for the whole confbridge-readiness
// budget. Nothing re-checked the AIcall after that wait, so an AIcall
// terminated (or deleted) during it went on to pre-write state and start a
// BILLED STT session. The re-read under the start lock closes that.
func Test_startListenTranscribe_LivenessRecheck(t *testing.T) {
	lockToken := uuid.FromStringOrNil("88880000-0000-4000-8000-0000000000ff")
	tmDelete := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		// The row as the FRESH under-the-lock read sees it -- deliberately
		// different from the caller's in-hand copy, which is what the pre-wait
		// check saw.
		fresh *aicall.AIcall
	}{
		{
			name: "terminated during the confbridge wait",
			fresh: func() *aicall.AIcall {
				c := listenEligibleAIcall()
				c.Status = aicall.StatusTerminated
				return c
			}(),
		},
		{
			name: "terminating during the confbridge wait",
			fresh: func() *aicall.AIcall {
				c := listenEligibleAIcall()
				c.Status = aicall.StatusTerminating
				return c
			}(),
		},
		{
			name: "deleted during the confbridge wait",
			fresh: func() *aicall.AIcall {
				c := listenEligibleAIcall()
				c.TMDelete = &tmDelete
				return c
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetListenDefaultsForTest()

			mc := gomock.NewController(t)
			defer mc.Finish()

			m := newStartListenHarness(mc)
			ctx := context.Background()

			// The caller's own copy still says progressing -- exactly the
			// stale view checkListenEligible left behind before the wait.
			c := listenEligibleAIcall()

			m.util.EXPECT().UUIDCreate().Return(lockToken)
			m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
			m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)

			// THE DB SAYS TERMINATED; THE CACHE-FIRST PATH WOULD STILL SAY
			// PROGRESSING (review round 2 finding MEDIUM-2). ProcessTerminate
			// lives outside this feature and refreshes the aicall cache only
			// best-effort, so a refresh that silently failed leaves exactly
			// this divergence. Wiring AIcallGet to return the STALE, still
			// listenable row -- and asserting it is never called -- proves the
			// re-check reads through to the database rather than believing a
			// stale snapshot and starting a billed session on a dead AIcall.
			m.db.EXPECT().AIcallGetSkipCache(ctx, ltAIcallID).Return(tt.fresh, nil)
			m.db.EXPECT().AIcallGet(gomock.Any(), gomock.Any()).Return(listenEligibleAIcall(), nil).Times(0)

			// NOTHING past the re-check: no list, no speculative id, no state
			// write, and above all no billed transcribe.
			m.req.EXPECT().TranscribeV1TranscribeList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			m.req.EXPECT().TranscribeV1TranscribeStart(
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			).Times(0)
			m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			m.cache.EXPECT().ListenAIcallIDAdd(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			before := testutil.ToFloat64(promListenStartTotal.WithLabelValues("call", "skipped_not_listenable"))

			res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
			if err != nil {
				t.Fatalf("a no-longer-listenable aicall is not an error. err: %v", err)
			}
			if res != "skipped_not_listenable" {
				t.Errorf("result mismatch. expected: skipped_not_listenable, got: %s", res)
			}

			// runListenStart is what actually emits the label, so emit it here
			// the same way and assert the counter moved by exactly one.
			promListenStartTotal.WithLabelValues("call", res).Inc()
			if got := testutil.ToFloat64(promListenStartTotal.WithLabelValues("call", "skipped_not_listenable")) - before; got != 1 {
				t.Errorf("the outcome must be metered exactly once as skipped_not_listenable. got: %v", got)
			}
		})
	}

	t.Run("a failed re-read fails closed and starts nothing", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)
		m.db.EXPECT().AIcallGetSkipCache(ctx, ltAIcallID).Return(nil, fmt.Errorf("db down"))

		m.req.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Times(0)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err == nil {
			t.Fatalf("expected an error")
		}
		if res != "failed" {
			t.Errorf("result mismatch. expected: failed, got: %s", res)
		}
	})
}

// Test_startListenTranscribe_ReadsFieldsOffTheFreshAIcall pins review round 2's
// LOW-3: once the liveness re-check has a fresh copy of the row in hand, every
// field read after it must come from THAT copy, not from the caller's
// pre-confbridge-wait `c`.
//
// STTLanguage is the one such read, and it is observable: it becomes the
// language argument of the billed transcribe session.
func Test_startListenTranscribe_ReadsFieldsOffTheFreshAIcall(t *testing.T) {
	config.SetListenDefaultsForTest()

	lockToken := uuid.FromStringOrNil("88880000-0000-4000-8000-0000000002ff")
	newTranscribeID := uuid.FromStringOrNil("88880000-0000-4000-8000-000000000201")

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newStartListenHarness(mc)
	ctx := context.Background()

	// The caller's copy carries the language as it stood BEFORE the confbridge
	// wait; the row has since been updated.
	c := listenEligibleAIcall()
	c.STTLanguage = "en-US"

	fresh := listenEligibleAIcall()
	fresh.STTLanguage = "ko-KR"

	m.util.EXPECT().UUIDCreate().Return(lockToken)
	m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
	m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)
	m.expectListenLivenessRecheck(ctx, fresh)
	m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(nil, nil)
	m.util.EXPECT().UUIDCreate().Return(newTranscribeID)
	m.expectUpdateListenState(ctx, fresh, newTranscribeID)

	m.req.EXPECT().TranscribeV1TranscribeStart(
		ctx, newTranscribeID, cmcustomer.IDAIManagerListen, gomock.Any(), uuid.Nil,
		tmtranscribe.ReferenceTypeCall, ltCallID, "ko-KR",
		tmtranscribe.DirectionBoth, tmtranscribe.ProviderEmpty, defaultListenTranscribeStartTimeout,
	).Return(&tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: newTranscribeID}}, nil)

	res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
	if err != nil {
		t.Fatalf("unexpected error. err: %v", err)
	}
	if res != "started" {
		t.Errorf("result mismatch. expected: started, got: %s", res)
	}
}

// Test_runListenStart_TerminatedDuringConfbridgeWait exercises the SAME defect
// through the whole detached stage, so the assertion is not merely about
// startListenTranscribe in isolation: the confbridge wait really does run
// first, and the transcribe start really is never reached afterwards.
func Test_runListenStart_TerminatedDuringConfbridgeWait(t *testing.T) {
	config.SetListenDefaultsForTest()

	mc := gomock.NewController(t)
	defer mc.Finish()

	m := newStartListenHarness(mc)
	ctx := context.Background()

	lockToken := uuid.FromStringOrNil("88880000-0000-4000-8000-0000000001ff")
	c := listenEligibleAIcall()

	terminated := listenEligibleAIcall()
	terminated.Status = aicall.StatusTerminated

	// Step 7: the confbridge settles immediately -- the wait itself is not
	// what is under test here.
	m.req.EXPECT().CallV1CallGet(ctx, ltCallID).Return(listenEligibleCall(), nil)
	m.req.EXPECT().CallV1ConfbridgeGet(ctx, ltConfID).Return(&cmconfbridge.Confbridge{
		Status:         cmconfbridge.StatusProgressing,
		ChannelCallIDs: map[string]uuid.UUID{"a": ltCallID, "b": ltAIcallID},
	}, nil)

	// Step 8: the AIcall was terminated while step 7 was polling.
	m.util.EXPECT().UUIDCreate().Return(lockToken)
	m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
	m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)
	m.db.EXPECT().AIcallGetSkipCache(ctx, ltAIcallID).Return(terminated, nil)

	m.req.EXPECT().TranscribeV1TranscribeStart(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Times(0)

	before := testutil.ToFloat64(promListenStartTotal.WithLabelValues("call", "skipped_not_listenable"))

	m.h.runListenStart(ctx, &ai.AI{Type: ai.TypeInsight}, c, listenEligibleCase(), ltCallID, listenEligibleCall())

	if got := testutil.ToFloat64(promListenStartTotal.WithLabelValues("call", "skipped_not_listenable")) - before; got != 1 {
		t.Errorf("the outcome must be metered exactly once as skipped_not_listenable. got: %v", got)
	}
}

// Test_pickReusableListenTranscribe is the direct regression test for review
// round 1's security MEDIUM-1: TranscribeV1TranscribeList's filter map is
// caller-supplied and NOT server-enforced, so a returned row's ownership
// fields must be re-verified before its id is adopted.
func Test_pickReusableListenTranscribe(t *testing.T) {
	goodID := uuid.FromStringOrNil("99990000-0000-4000-8000-000000000001")
	otherCallID := uuid.FromStringOrNil("99990000-0000-4000-8000-0000000000aa")
	tmDelete := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		rows []tmtranscribe.Transcribe

		expectID uuid.UUID
		expectOK bool
	}{
		{
			name:     "an empty list yields no winner",
			rows:     nil,
			expectID: uuid.Nil,
			expectOK: false,
		},
		{
			name:     "a fully matching row is adopted",
			rows:     []tmtranscribe.Transcribe{listenReusableRow(goodID)},
			expectID: goodID,
			expectOK: true,
		},
		{
			name: "a foreign customer id is rejected",
			rows: []tmtranscribe.Transcribe{func() tmtranscribe.Transcribe {
				row := listenReusableRow(goodID)
				row.CustomerID = ltCustomerID // the tenant's own id, not the platform sentinel
				return row
			}()},
			expectID: uuid.Nil,
			expectOK: false,
		},
		{
			name: "a mismatched reference id is rejected",
			rows: []tmtranscribe.Transcribe{func() tmtranscribe.Transcribe {
				row := listenReusableRow(goodID)
				row.ReferenceID = otherCallID
				return row
			}()},
			expectID: uuid.Nil,
			expectOK: false,
		},
		{
			name: "a deleted row is rejected",
			rows: []tmtranscribe.Transcribe{func() tmtranscribe.Transcribe {
				row := listenReusableRow(goodID)
				row.TMDelete = &tmDelete
				return row
			}()},
			expectID: uuid.Nil,
			expectOK: false,
		},
		{
			name:     "a nil id is rejected",
			rows:     []tmtranscribe.Transcribe{listenReusableRow(uuid.Nil)},
			expectID: uuid.Nil,
			expectOK: false,
		},
		{
			// Review round 2 MEDIUM-1: dupFilters constrains Status to
			// progressing, so a non-progressing row coming back at all means
			// the RPC's own filtering cannot be trusted. Adopting it would
			// record "reused" as a success while no listening ever starts.
			name: "a non-progressing row is rejected",
			rows: []tmtranscribe.Transcribe{func() tmtranscribe.Transcribe {
				row := listenReusableRow(goodID)
				row.Status = tmtranscribe.StatusDone
				return row
			}()},
			expectID: uuid.Nil,
			expectOK: false,
		},
		{
			// Same finding: this design only ever lists ReferenceType == Call
			// sessions, so a row of any other reference type is a
			// filter-mapping regression, not a candidate.
			name: "a mismatched reference type is rejected",
			rows: []tmtranscribe.Transcribe{func() tmtranscribe.Transcribe {
				row := listenReusableRow(goodID)
				row.ReferenceType = tmtranscribe.ReferenceTypeRecording
				return row
			}()},
			expectID: uuid.Nil,
			expectOK: false,
		},
		{
			name: "an unverifiable row does not mask a genuine one behind it",
			rows: []tmtranscribe.Transcribe{
				func() tmtranscribe.Transcribe {
					row := listenReusableRow(otherCallID)
					row.ReferenceID = otherCallID
					return row
				}(),
				listenReusableRow(goodID),
			},
			expectID: goodID,
			expectOK: true,
		},
		{
			name: "a non-progressing row does not mask a genuine one behind it",
			rows: []tmtranscribe.Transcribe{
				func() tmtranscribe.Transcribe {
					row := listenReusableRow(otherCallID)
					row.Status = tmtranscribe.StatusDone
					return row
				}(),
				listenReusableRow(goodID),
			},
			expectID: goodID,
			expectOK: true,
		},
		{
			name: "a wrong-reference-type row does not mask a genuine one behind it",
			rows: []tmtranscribe.Transcribe{
				func() tmtranscribe.Transcribe {
					row := listenReusableRow(otherCallID)
					row.ReferenceType = tmtranscribe.ReferenceTypeRecording
					return row
				}(),
				listenReusableRow(goodID),
			},
			expectID: goodID,
			expectOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := pickReusableListenTranscribe(tt.rows, ltCallID)
			if gotOK != tt.expectOK {
				t.Fatalf("ok mismatch. expected: %v, got: %v", tt.expectOK, gotOK)
			}
			if gotID != tt.expectID {
				t.Errorf("id mismatch. expected: %s, got: %s", tt.expectID, gotID)
			}
		})
	}
}

// Test_startListenTranscribe_ReuseOwnershipVerification pins the same finding
// at BOTH call sites -- the initial reuse check and the
// already-progressing re-run -- since either one adopting an unverified row is
// the actual defect.
func Test_startListenTranscribe_ReuseOwnershipVerification(t *testing.T) {
	lockToken := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-0000000000ff")
	newTranscribeID := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-000000000001")
	foreignID := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-000000000002")
	otherCallID := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-0000000000aa")

	// Superficially a hit for the filter that was sent, but its reference is a
	// DIFFERENT call -- exactly the row a non-enforcing list endpoint could
	// return.
	mismatched := listenReusableRow(foreignID)
	mismatched.ReferenceID = otherCallID

	t.Run("the initial reuse check does not adopt a mismatched row", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)
		m.expectListenLivenessRecheck(ctx, c)
		m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).
			Return([]tmtranscribe.Transcribe{mismatched}, nil)

		// The mismatched row is treated as "no session found": a fresh session
		// is created instead, and the foreign id is NEVER written anywhere.
		m.util.EXPECT().UUIDCreate().Return(newTranscribeID)
		m.expectUpdateListenState(ctx, c, newTranscribeID)
		m.cache.EXPECT().ListenAIcallIDAdd(ctx, foreignID, gomock.Any(), gomock.Any()).Times(0)
		m.req.EXPECT().TranscribeV1TranscribeStart(
			ctx, newTranscribeID, cmcustomer.IDAIManagerListen, gomock.Any(), uuid.Nil,
			tmtranscribe.ReferenceTypeCall, ltCallID, gomock.Any(),
			tmtranscribe.DirectionBoth, tmtranscribe.ProviderEmpty, defaultListenTranscribeStartTimeout,
		).Return(&tmtranscribe.Transcribe{Identity: commonidentity.Identity{ID: newTranscribeID}}, nil)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
		if res != "started" {
			t.Errorf("an unverifiable row must not read as a reusable session. expected: started, got: %s", res)
		}
	})

	t.Run("the already-progressing re-run does not adopt a mismatched row", func(t *testing.T) {
		config.SetListenDefaultsForTest()

		mc := gomock.NewController(t)
		defer mc.Finish()

		m := newStartListenHarness(mc)
		ctx := context.Background()
		c := listenEligibleAIcall()

		alreadyProgressing := &cerrors.VoipbinError{Reason: transcribeReasonAlreadyProgressing}

		m.util.EXPECT().UUIDCreate().Return(lockToken)
		m.cache.EXPECT().ListenStartLockAcquire(ctx, ltAIcallID, lockToken.String(), gomock.Any()).Return(true, nil)
		m.cache.EXPECT().ListenStartLockRelease(gomock.Any(), ltAIcallID, lockToken.String()).Return(nil)
		m.expectListenLivenessRecheck(ctx, c)

		gomock.InOrder(
			m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).Return(nil, nil),
			m.req.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(10), gomock.Any()).
				Return([]tmtranscribe.Transcribe{mismatched}, nil),
		)
		m.util.EXPECT().UUIDCreate().Return(newTranscribeID)
		m.expectUpdateListenState(ctx, c, newTranscribeID)
		m.req.EXPECT().TranscribeV1TranscribeStart(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(nil, alreadyProgressing)

		// Same treatment as "no winner found": roll back our own speculative
		// id and give up. The foreign id is never adopted.
		m.cache.EXPECT().ListenAIcallIDRemove(ctx, newTranscribeID, ltAIcallID).Return(nil)
		m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)
		m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)
		m.cache.EXPECT().ListenAIcallIDAdd(ctx, foreignID, gomock.Any(), gomock.Any()).Times(0)

		res, err := m.h.startListenTranscribe(ctx, c, listenEligibleCall(), ltCallID)
		if err == nil {
			t.Fatalf("expected an error")
		}
		if res != "failed" {
			t.Errorf("result mismatch. expected: failed, got: %s", res)
		}
	})
}

// Test_checkListenEligible_ReferenceTypeGate pins review round 1's security
// LOW-1: c.ReferenceID is only a Case id for a contact_case AIcall, so the
// Case lookup must not be attempted for any other reference type.
func Test_checkListenEligible_ReferenceTypeGate(t *testing.T) {
	for _, referenceType := range []aicall.ReferenceType{
		aicall.ReferenceTypeCall,
		aicall.ReferenceTypeConversation,
		aicall.ReferenceType(""),
	} {
		t.Run(string("reference_type="+referenceType), func(t *testing.T) {
			config.SetListenDefaultsForTest()
			defer config.SetListenDefaultsForTest()

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				reqHandler: mockReq,
				aiHandler:  mockAI,
			}

			ctx := context.Background()
			c := listenEligibleAIcall()
			c.ReferenceType = referenceType

			mockAI.EXPECT().Get(ctx, ltAIID).Return(&ai.AI{Type: ai.TypeInsight}, nil)
			// The gate is BEFORE the Case lookup, so no Case RPC is made at all.
			mockReq.EXPECT().ContactV1CaseGet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			_, _, _, _, proceed, err := h.checkListenEligible(ctx, c)
			if err != nil {
				t.Fatalf("a non-listenable reference type is not an error. err: %v", err)
			}
			if proceed {
				t.Errorf("listening must not proceed for reference_type %s", referenceType)
			}
		})
	}
}
