package pipecatcallhandler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_startReferenceTypeCall_callGetFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-1111-2222-3333-444455556666")

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pcID,
			CustomerID: uuid.FromStringOrNil("c3d4e5f6-1111-2222-3333-444455556666"),
		},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
	}

	// CallV1CallGet fails
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).
		Return(nil, fmt.Errorf("call not found"))

	err := h.startReferenceTypeCall(context.Background(), pc)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func Test_startReferenceTypeCall_externalMediaFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockPythonRunner := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)

	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		pythonRunner:          mockPythonRunner,
		toolHandler:           mockTool,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-1111-2222-3333-444455556666")
	callID := uuid.FromStringOrNil("d4e5f6a7-1111-2222-3333-444455556666")

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pcID,
			CustomerID: uuid.FromStringOrNil("c3d4e5f6-1111-2222-3333-444455556666"),
		},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
	}

	// CallV1CallGet succeeds
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).
		Return(&cmcall.Call{
			Identity: commonidentity.Identity{
				ID: callID,
			},
		}, nil)

	// CallV1ExternalMediaStart fails
	mockReq.EXPECT().CallV1ExternalMediaStart(
		gomock.Any(),
		pcID,
		cmexternalmedia.ReferenceTypeCall,
		callID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"",
		defaultConnectionType,
		defaultFormat,
		cmexternalmedia.DirectionIn,
		cmexternalmedia.DirectionOut,
	).Return(nil, fmt.Errorf("external media creation failed"))

	// RunnerStart goroutine may call these before context is cancelled
	mockTool.EXPECT().GetAll().AnyTimes()
	mockPythonRunner.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockPythonRunner.EXPECT().Stop(gomock.Any(), gomock.Any()).AnyTimes()

	err := h.startReferenceTypeCall(context.Background(), pc)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func Test_markTerminatedOnce_idempotent(t *testing.T) {
	h := &pipecatcallHandler{
		terminatedPublished: make(map[uuid.UUID]struct{}),
		muTerminated:        sync.Mutex{},
	}

	id, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("could not generate uuid: %v", err)
	}

	if !h.markTerminatedOnce(id) {
		t.Fatalf("first call should claim")
	}
	if h.markTerminatedOnce(id) {
		t.Fatalf("second call should not claim")
	}
}

func Test_terminatedDeleteEntry(t *testing.T) {
	h := &pipecatcallHandler{
		terminatedPublished: make(map[uuid.UUID]struct{}),
		muTerminated:        sync.Mutex{},
	}

	id, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("could not generate uuid: %v", err)
	}

	// Deleting a missing entry is a safe no-op.
	h.terminatedDeleteEntry(id)

	// Claim, delete, then re-claim — second claim should succeed.
	if !h.markTerminatedOnce(id) {
		t.Fatalf("first claim should succeed")
	}
	h.terminatedDeleteEntry(id)
	if !h.markTerminatedOnce(id) {
		t.Fatalf("re-claim after delete should succeed")
	}
}

func Test_startReferenceTypeCall_websocketDialFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockWS := NewMockWebsocketHandler(mc)
	mockPythonRunner := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)

	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		websocketHandler:      mockWS,
		pythonRunner:          mockPythonRunner,
		toolHandler:           mockTool,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-1111-2222-3333-444455556666")
	callID := uuid.FromStringOrNil("d4e5f6a7-1111-2222-3333-444455556666")
	emID := uuid.FromStringOrNil("e5f6a7b8-1111-2222-3333-444455556666")
	mediaURI := "ws://asterisk:8088/ws/test-media"

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pcID,
			CustomerID: uuid.FromStringOrNil("c3d4e5f6-1111-2222-3333-444455556666"),
		},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
	}

	// CallV1CallGet succeeds
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).
		Return(&cmcall.Call{
			Identity: commonidentity.Identity{
				ID: callID,
			},
		}, nil)

	// CallV1ExternalMediaStart succeeds
	mockReq.EXPECT().CallV1ExternalMediaStart(
		gomock.Any(),
		pcID,
		cmexternalmedia.ReferenceTypeCall,
		callID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"",
		defaultConnectionType,
		defaultFormat,
		cmexternalmedia.DirectionIn,
		cmexternalmedia.DirectionOut,
	).Return(&cmexternalmedia.ExternalMedia{
		ID:       emID,
		MediaURI: mediaURI,
	}, nil)

	// WebSocket dial fails
	mockWS.EXPECT().DialContext(gomock.Any(), mediaURI, gomock.Any()).
		Return(nil, nil, fmt.Errorf("connection refused"))

	// Cleanup: CallV1ExternalMediaStop should be called with em.ID
	mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), emID).
		Return(&cmexternalmedia.ExternalMedia{ID: emID}, nil)

	// RunnerStart goroutine may call these before context is cancelled
	mockTool.EXPECT().GetAll().AnyTimes()
	mockPythonRunner.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockPythonRunner.EXPECT().Stop(gomock.Any(), gomock.Any()).AnyTimes()

	err := h.startReferenceTypeCall(context.Background(), pc)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

// Test_terminate_publishesTerminatedEventOnce asserts that calling terminate()
// twice for the same pipecatcall publishes the pipecatcall_terminated event
// exactly once (deduped via markTerminatedOnce).
func Test_terminate_publishesTerminatedEventOnce(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockPythonRunner := NewMockPythonRunner(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &pipecatcallHandler{
		notifyHandler:         mockNotify,
		pythonRunner:          mockPythonRunner,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
		muTerminated:          sync.Mutex{},
		terminatedPublished:   make(map[uuid.UUID]struct{}),
	}

	pcID := uuid.FromStringOrNil("c1d2e3f4-5555-6666-7777-888899990000")
	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pcID,
			CustomerID: uuid.FromStringOrNil("d2e3f4a5-5555-6666-7777-888899990000"),
		},
		// Use an unknown reference type so terminate() hits the outer default
		// branch — no extra reference-type RPCs needed.
		ReferenceType: pipecatcall.ReferenceType("unknown"),
	}

	// Seed a session so flushAndFinalize takes the noop_never_started branch
	// and SessionStop can find + delete it.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connAstReady := make(chan struct{})
	close(connAstReady)
	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID: pcID,
		},
		Ctx:          ctx,
		Cancel:       cancel,
		ConnAstReady: connAstReady,
	}
	h.mapPipecatcallSession[pcID] = se

	// Count terminated-event publishes via a recorder. Other event types
	// shouldn't be emitted here, but allow them with AnyTimes for resilience.
	var terminatedCount atomic.Int32
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, eventType string, _ any) {
			if eventType == pipecatcall.EventTypePipecatcallTerminated {
				terminatedCount.Add(1)
			}
		},
	).AnyTimes()

	// First terminate(): SessionStop will run pythonRunner.Stop once and remove
	// the session. The second terminate() finds no session, so SessionGet
	// inside SessionStop logs and returns without invoking pythonRunner.Stop
	// again.
	mockPythonRunner.EXPECT().Stop(gomock.Any(), pcID).Return(nil).Times(1)

	h.terminate(context.Background(), pc)
	h.terminate(context.Background(), pc) // second call must be a no-op for publish

	if got := terminatedCount.Load(); got != 1 {
		t.Fatalf("expected exactly 1 pipecatcall_terminated publish, got %d", got)
	}
}

// Test_terminate_callsFlushBeforeStop asserts that terminate() calls
// flushAndFinalize *before* tearing the session down, and that the flush
// helper attributes its exit to StopReasonTerminateForce.
func Test_terminate_callsFlushBeforeStop(t *testing.T) {
	// Patch the timeout so the timeout branch returns quickly when the
	// stalled flush goroutine never closes LLMDoneChan.
	origTimeout := flushFinalizeTimeout
	flushFinalizeTimeout = 50 * time.Millisecond
	t.Cleanup(func() { flushFinalizeTimeout = origTimeout })

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockPythonRunner := NewMockPythonRunner(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	h := &pipecatcallHandler{
		notifyHandler:         mockNotify,
		pythonRunner:          mockPythonRunner,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
		muTerminated:          sync.Mutex{},
		terminatedPublished:   make(map[uuid.UUID]struct{}),
	}

	pcID := uuid.FromStringOrNil("e3f4a5b6-5555-6666-7777-888899990000")
	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         pcID,
			CustomerID: uuid.FromStringOrNil("f4a5b6c7-5555-6666-7777-888899990000"),
		},
		ReferenceType: pipecatcall.ReferenceType("unknown"),
	}

	// Arm a stalled flush: channels + flag set, but no goroutine drains
	// LLMDoneChan. flushAndFinalize will CAS LLMStopReason to
	// StopReasonTerminateForce, close LLMStopChan, then time out — but the
	// CAS already records the reason we assert on.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connAstReady := make(chan struct{})
	close(connAstReady)
	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID: pcID,
		},
		Ctx:          ctx,
		Cancel:       cancel,
		ConnAstReady: connAstReady,
		LLMMessageID: uuid.FromStringOrNil("11112222-3333-4444-5555-666677778888"),
		LLMTokenChan: make(chan string, 64),
		LLMStopChan:  make(chan struct{}),
		LLMDoneChan:  make(chan struct{}),
	}
	se.LLMFlushing.Store(true)
	h.mapPipecatcallSession[pcID] = se

	mockPythonRunner.EXPECT().Stop(gomock.Any(), pcID).Return(nil).Times(1)

	h.terminate(context.Background(), pc)

	if got := StopReason(se.LLMStopReason.Load()); got != StopReasonTerminateForce {
		t.Fatalf("expected StopReasonTerminateForce (%d), got %d", StopReasonTerminateForce, got)
	}
}
