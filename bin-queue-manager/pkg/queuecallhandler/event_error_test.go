package queuecallhandler

import (
	"context"
	"errors"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

var errEventTest = errors.New("simulated dependency failure")

// Test_Event_statusUpdateFails verifies that a failing status write on the call
// events does not panic and does not trigger any further side effect.
func Test_Event_statusUpdateFails(t *testing.T) {
	referenceID := uuid.FromStringOrNil("7e3f4a5b-4001-11f1-9d00-000000000001")
	confbridgeID := uuid.FromStringOrNil("7e3f4a5b-4002-11f1-9d00-000000000002")
	id := uuid.FromStringOrNil("7e3f4a5b-4003-11f1-9d00-000000000003")

	tests := []struct {
		name   string
		status queuecall.Status
		run    func(ctx context.Context, h *queuecallHandler)

		expectCall func(ctx context.Context, mockDB *dbhandler.MockDBHandler)
	}{
		{
			name:   "hangup of a waiting queuecall",
			status: queuecall.StatusWaiting,
			run: func(ctx context.Context, h *queuecallHandler) {
				h.EventCallCallHangup(ctx, referenceID)
			},
			expectCall: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().QueuecallSetStatusAbandoned(ctx, id, gomock.Any(), gomock.Any()).Return(int64(0), errEventTest)
			},
		},
		{
			name:   "confbridge joined",
			status: queuecall.StatusConnecting,
			run: func(ctx context.Context, h *queuecallHandler) {
				h.EventCallConfbridgeJoined(ctx, referenceID, confbridgeID)
			},
			expectCall: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().QueuecallSetStatusService(ctx, id, gomock.Any(), gomock.Any()).Return(int64(0), errEventTest)
			},
		},
		{
			name:   "confbridge leaved",
			status: queuecall.StatusService,
			run: func(ctx context.Context, h *queuecallHandler) {
				h.EventCallConfbridgeLeaved(ctx, referenceID, confbridgeID)
			},
			expectCall: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().QueuecallSetStatusDone(ctx, id, gomock.Any(), gomock.Any()).Return(int64(0), errEventTest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queuecallHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			qc := &queuecall.Queuecall{
				Identity:     commonidentity.Identity{ID: id},
				ReferenceID:  referenceID,
				ConfbridgeID: confbridgeID,
				Status:       tt.status,
			}
			mockDB.EXPECT().QueuecallGetByReferenceID(ctx, referenceID).Return(qc, nil)
			mockUtil.EXPECT().TimeNow().Return(nil)
			tt.expectCall(ctx, mockDB)

			// no notification is expected, a notify call would fail the test.
			tt.run(ctx, h)
		})
	}
}

func Test_EventAMAgentAvailable_errors(t *testing.T) {
	agent := amagent.Agent{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("7e3f4a5b-4011-11f1-9d00-000000000011")},
		Status:   amagent.StatusAvailable,
	}
	queueID1 := uuid.FromStringOrNil("7e3f4a5b-4012-11f1-9d00-000000000012")
	queueID2 := uuid.FromStringOrNil("7e3f4a5b-4013-11f1-9d00-000000000013")
	queuecallID := uuid.FromStringOrNil("7e3f4a5b-4014-11f1-9d00-000000000014")

	qs := []*queue.Queue{
		{Identity: commonidentity.Identity{ID: queueID1}},
		{Identity: commonidentity.Identity{ID: queueID2}},
	}

	tests := []struct {
		name  string
		setup func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockQueue *queuehandler.MockQueueHandler)
	}{
		{
			name: "queue lookup fails",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockQueue *queuehandler.MockQueueHandler) {
				mockQueue.EXPECT().GetQueuesByAgent(ctx, agent).Return(nil, errEventTest)
			},
		},
		{
			name: "a failing queue is skipped and the next one is tried",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockQueue *queuehandler.MockQueueHandler) {
				mockQueue.EXPECT().GetQueuesByAgent(ctx, agent).Return(qs, nil)
				mockDB.EXPECT().QueuecallListOldestWaiting(ctx, queueID1, uint64(1)).Return(nil, errEventTest)
				mockDB.EXPECT().QueuecallListOldestWaiting(ctx, queueID2, uint64(1)).Return([]*queuecall.Queuecall{}, nil)
			},
		},
		{
			name: "a failed match does not stop the loop",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockQueue *queuehandler.MockQueueHandler) {
				mockQueue.EXPECT().GetQueuesByAgent(ctx, agent).Return(qs, nil)
				waiting := []*queuecall.Queuecall{{Identity: commonidentity.Identity{ID: queuecallID}}}
				mockDB.EXPECT().QueuecallListOldestWaiting(ctx, queueID1, uint64(1)).Return(waiting, nil)
				// Execute fails at the first lookup
				mockDB.EXPECT().QueuecallGet(ctx, queuecallID).Return(nil, errEventTest)
				mockDB.EXPECT().QueuecallListOldestWaiting(ctx, queueID2, uint64(1)).Return([]*queuecall.Queuecall{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &queuecallHandler{
				db:           mockDB,
				reqHandler:   mockReq,
				queueHandler: mockQueue,
			}
			ctx := context.Background()

			tt.setup(ctx, mockDB, mockQueue)

			h.EventAMAgentAvailable(ctx, agent)
		})
	}
}
