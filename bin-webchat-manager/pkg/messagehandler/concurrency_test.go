package messagehandler

import (
	"context"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

// Test_Create_ConcurrentFirstMessages_TriggersFlowExactlyOnce is the
// real multi-goroutine concurrency test for the Round 4 review finding:
// two "first" inbound messages racing on the SAME Session must trigger
// FlowV1ActiveflowCreate exactly once, not twice. This is the test the
// sequencing-only tests above cannot substitute for -- it actually
// exercises h.lockSession/unlockSession under real goroutine contention.
func Test_Create_ConcurrentFirstMessages_TriggersFlowExactlyOnce(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), message.EventTypeMessageCreated, gomock.Any()).AnyTimes()

	h := &messageHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
		sessionLocks:  map[uuid.UUID]chan struct{}{},
	}

	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	flowID := uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30")
	activeflowID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")

	w := &widget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		FlowID:   flowID,
	}

	// A mutable "current session state" the mocked SessionGet/SessionUpdate
	// operate against, guarded by its own mutex to simulate a real DB row
	// (SessionUpdate's write must be visible to the NEXT SessionGet, which
	// is exactly the invariant that makes the lock meaningful).
	var stateMu sync.Mutex
	currentActiveflowID := uuid.Nil

	mockDB.EXPECT().SessionGet(ctx, sessionID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*session.Session, error) {
		stateMu.Lock()
		defer stateMu.Unlock()
		return &session.Session{
			Identity:     commonidentity.Identity{ID: sessionID, CustomerID: customerID},
			WidgetID:     widgetID,
			Status:       session.StatusActive,
			ActiveflowID: currentActiveflowID,
		}, nil
	}).AnyTimes()

	mockDB.EXPECT().SessionUpdate(ctx, sessionID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[session.Field]any) error {
		stateMu.Lock()
		defer stateMu.Unlock()
		if id, ok := fields[session.FieldActiveflowID].(uuid.UUID); ok {
			currentActiveflowID = id
		}
		return nil
	}).AnyTimes()

	mockDB.EXPECT().WidgetGet(ctx, widgetID).Return(w, nil).AnyTimes()

	// Each concurrent call gets its own fresh message ID via DoAndReturn
	// (a fixed UUID would make every goroutine's insert collide on the
	// same id and defeat the point of the test).
	var idCounter int
	var idMu sync.Mutex
	mockUtil.EXPECT().UUIDCreate().DoAndReturn(func() uuid.UUID {
		idMu.Lock()
		defer idMu.Unlock()
		idCounter++
		return uuid.FromStringOrNil("00000000-0000-0000-0000-00000000000" + string(rune('0'+idCounter)))
	}).AnyTimes()

	mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil).AnyTimes()
	mockDB.EXPECT().MessageGet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*message.Message, error) {
		return &message.Message{
			Identity:  commonidentity.Identity{ID: id, CustomerID: customerID},
			SessionID: sessionID,
			Direction: message.DirectionInbound,
			Status:    message.StatusSent,
		}, nil
	}).AnyTimes()

	// The critical assertion: FlowV1ActiveflowCreate/Execute must each
	// fire EXACTLY ONCE across all concurrent callers, enforced via
	// .Times(1) -- if the lock is broken and both goroutines observe
	// ActiveflowID==uuid.Nil, gomock fails the test on the second call.
	mockReq.EXPECT().FlowV1ActiveflowCreate(
		ctx, uuid.Nil, customerID, flowID, fmactiveflow.ReferenceTypeWebchat, sessionID, uuid.Nil, gomock.Any(), "", fmactiveflow.WebhookMethodNone,
	).Times(1).Return(&fmactiveflow.Activeflow{
		Identity: commonidentity.Identity{ID: activeflowID},
	}, nil)
	mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, activeflowID).Times(1).Return(nil)

	const numGoroutines = 8
	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := h.Create(ctx, customerID, sessionID, message.DirectionInbound, uuid.Nil, "hello"); err != nil {
				errCh <- err
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for concurrent Create calls -- possible deadlock in lockSession")
	}
	close(errCh)

	for err := range errCh {
		t.Errorf("unexpected error from concurrent Create: %v", err)
	}
}
