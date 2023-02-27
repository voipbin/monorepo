package queuecallhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuehandler"
)

func Test_EventCallCallHangup(t *testing.T) {

	tests := []struct {
		name string

		referenceID uuid.UUID

		responseQueuecall *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),

			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:    uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:    uuid.FromStringOrNil("d7357136-5ee0-11ec-abd0-a7463d258061"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9ca8282-5edf-11ec-a876-df977791e643"),
				},

				Status:   queuecall.StatusWaiting,
				TMCreate: "2021-04-18 03:22:17.994000",
				TMEnd:    dbhandler.DefaultTimeStamp,
				TMDelete: "2021-04-18 03:52:17.994000",
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &queuecallHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
				queueHandler:  mockQueue,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseQueuecall, nil)

			// UpdateStatusAbandoned
			mockUtil.EXPECT().GetCurTime().Return(utilhandler.GetCurTime())
			mockDB.EXPECT().QueuecallSetStatusAbandoned(ctx, tt.responseQueuecall.ID, gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallAbandoned, tt.responseQueuecall)
			mockQueue.EXPECT().AddAbandonedQueuecallID(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID).Return(&queue.Queue{}, nil)

			mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.responseQueuecall.ConfbridgeID).Return(nil)
			mockReq.EXPECT().FlowV1VariableDeleteVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			h.EventCallCallHangup(ctx, tt.referenceID)
		})
	}
}

func Test_EventCallConfbridgeJoined(t *testing.T) {

	tests := []struct {
		name string

		referenceID  uuid.UUID
		confbridgeID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		responseCurTime   string

		expectDuration int
	}{
		{
			"normal",

			uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
			uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),

			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:    uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:    uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9ca8282-5edf-11ec-a876-df977791e643"),
				},

				Status:    queuecall.StatusService,
				TMCreate:  "2021-04-18 03:22:17.994000",
				TMService: "2021-04-18 03:22:17.994000",
				TMEnd:     dbhandler.DefaultTimeStamp,
				TMDelete:  dbhandler.DefaultTimeStamp,
			},
			"2021-04-18 03:23:17.994000",

			60000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &queuecallHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
				queueHandler:  mockQueue,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseQueuecall, nil)

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockDB.EXPECT().QueuecallSetStatusService(ctx, tt.responseQueuecall.ID, tt.expectDuration, tt.responseCurTime).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)

			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallServiced, tt.responseQueuecall)
			mockQueue.EXPECT().AddServiceQueuecallID(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID).Return(&queue.Queue{}, nil)

			if tt.responseQueuecall.TimeoutService > 0 {
				mockReq.EXPECT().QueueV1QueuecallTimeoutService(ctx, tt.responseQueuecall.ID, tt.responseQueuecall.TimeoutService).Return(nil)
			}

			h.EventCallConfbridgeJoined(ctx, tt.referenceID, tt.confbridgeID)
		})
	}
}

func Test_EventCallConfbridgeLeaved(t *testing.T) {
	tests := []struct {
		name string

		referenceID  uuid.UUID
		conferenceID uuid.UUID

		responseQueuecall *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
			uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),

			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:    uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:    uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9ca8282-5edf-11ec-a876-df977791e643"),
				},

				Status:    queuecall.StatusService,
				TMService: "2021-04-18 03:22:17.994000",
				TMEnd:     dbhandler.DefaultTimeStamp,
				TMDelete:  "2021-04-18 03:52:17.994000",
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &queuecallHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
				queueHandler:  mockQueue,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseQueuecall, nil)

			// UpdateStatusDone
			mockUtil.EXPECT().GetCurTime().Return(utilhandler.GetCurTime())
			mockDB.EXPECT().QueuecallSetStatusDone(ctx, tt.responseQueuecall.ID, gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallDone, tt.responseQueuecall)
			mockQueue.EXPECT().RemoveServiceQueuecallID(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID).Return(&queue.Queue{}, nil)

			mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.responseQueuecall.ConfbridgeID).Return(nil)
			mockReq.EXPECT().FlowV1VariableDeleteVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			h.EventCallConfbridgeLeaved(ctx, tt.referenceID, tt.conferenceID)
		})
	}
}
