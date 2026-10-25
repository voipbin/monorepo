package queuecallhandler

import (
	"context"
	"errors"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

var errServiceTest = errors.New("simulated dependency failure")

func Test_ServiceStart_errors(t *testing.T) {
	queueID := uuid.FromStringOrNil("6d2e3f4a-3001-11f1-9c00-000000000001")
	activeflowID := uuid.FromStringOrNil("6d2e3f4a-3002-11f1-9c00-000000000002")
	referenceID := uuid.FromStringOrNil("6d2e3f4a-3003-11f1-9c00-000000000003")
	customerID := uuid.FromStringOrNil("6d2e3f4a-3004-11f1-9c00-000000000004")
	queuecallID := uuid.FromStringOrNil("6d2e3f4a-3005-11f1-9c00-000000000005")
	confbridgeID := uuid.FromStringOrNil("6d2e3f4a-3006-11f1-9c00-000000000006")

	q := &queue.Queue{Identity: commonidentity.Identity{ID: queueID, CustomerID: customerID}}
	c := &cmcall.Call{Direction: cmcall.DirectionIncoming}

	tests := []struct {
		name          string
		referenceType queuecall.ReferenceType

		setup func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockQueue *queuehandler.MockQueueHandler)
	}{
		{
			name:          "unsupported reference type",
			referenceType: queuecall.ReferenceType("unsupported"),
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockQueue *queuehandler.MockQueueHandler) {
			},
		},
		{
			name:          "queue lookup fails",
			referenceType: queuecall.ReferenceTypeCall,
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockQueue *queuehandler.MockQueueHandler) {
				mockQueue.EXPECT().Get(ctx, queueID).Return(nil, errServiceTest)
			},
		},
		{
			name:          "call lookup fails",
			referenceType: queuecall.ReferenceTypeCall,
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockQueue *queuehandler.MockQueueHandler) {
				mockQueue.EXPECT().Get(ctx, queueID).Return(q, nil)
				mockReq.EXPECT().CallV1CallGet(ctx, referenceID).Return(nil, errServiceTest)
			},
		},
		{
			name:          "confbridge creation fails",
			referenceType: queuecall.ReferenceTypeCall,
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockQueue *queuehandler.MockQueueHandler) {
				mockQueue.EXPECT().Get(ctx, queueID).Return(q, nil)
				mockReq.EXPECT().CallV1CallGet(ctx, referenceID).Return(c, nil)
				mockUtil.EXPECT().UUIDCreate().Return(queuecallID)
				mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, customerID, activeflowID, cmconfbridge.ReferenceTypeQueue, queuecallID, cmconfbridge.TypeConnect).Return(nil, errServiceTest)
			},
		},
		{
			name:          "queuecall creation fails",
			referenceType: queuecall.ReferenceTypeCall,
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockReq *requesthandler.MockRequestHandler, mockQueue *queuehandler.MockQueueHandler) {
				mockQueue.EXPECT().Get(ctx, queueID).Return(q, nil)
				mockReq.EXPECT().CallV1CallGet(ctx, referenceID).Return(c, nil)
				mockUtil.EXPECT().UUIDCreate().Return(queuecallID)
				mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, customerID, activeflowID, cmconfbridge.ReferenceTypeQueue, queuecallID, cmconfbridge.TypeConnect).Return(&cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: confbridgeID}}, nil)
				// the ids of the target, loop and forward actions
				mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("6d2e3f4a-3007-11f1-9c00-000000000007")).Times(3)
				mockDB.EXPECT().QueuecallCreate(ctx, gomock.Any()).Return(errServiceTest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &queuecallHandler{
				utilHandler:  mockUtil,
				db:           mockDB,
				reqHandler:   mockReq,
				queueHandler: mockQueue,
			}
			ctx := context.Background()

			tt.setup(ctx, mockUtil, mockDB, mockReq, mockQueue)

			res, err := h.ServiceStart(ctx, queueID, activeflowID, tt.referenceType, referenceID)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil (res: %v)", res)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_getSource(t *testing.T) {
	source := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"}
	destination := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100002"}

	tests := []struct {
		name      string
		direction cmcall.Direction
		expectRes commonaddress.Address
	}{
		{"incoming call uses the source", cmcall.DirectionIncoming, source},
		{"outgoing call uses the destination", cmcall.DirectionOutgoing, destination},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &queuecallHandler{}

			c := &cmcall.Call{Direction: tt.direction, Source: source, Destination: destination}
			res := h.getSource(c)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
