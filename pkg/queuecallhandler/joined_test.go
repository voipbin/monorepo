package queuecallhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallreferencehandler"
)

func TestJoined(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockQueuecallReferenceHandler := queuecallreferencehandler.NewMockQueuecallReferenceHandler(mc)

	h := &queuecallHandler{
		db:                        mockDB,
		reqHandler:                mockReq,
		notifyhandler:             mockNotify,
		queuecallReferenceHandler: mockQueuecallReferenceHandler,
	}

	tests := []struct {
		name string

		referenceID  uuid.UUID
		confbridgeID uuid.UUID

		queuecallReference *queuecallreference.QueuecallReference
		queuecall          *queuecall.Queuecall
		responseQueuecall  *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
			uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),

			&queuecallreference.QueuecallReference{
				ID:                 uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				CurrentQueuecallID: uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				QueuecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				},
			},
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:    uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:    uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
				WebhookURI:      "test.com",
				WebhookMethod:   "",
				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9ca8282-5edf-11ec-a876-df977791e643"),
				},

				Status: queuecall.StatusService,
			},
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:    uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:    uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
				WebhookURI:      "test.com",
				WebhookMethod:   "",
				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9ca8282-5edf-11ec-a876-df977791e643"),
				},

				Status:    queuecall.StatusService,
				TMService: "2021-04-18 03:22:17.994000",
				TMDelete:  "2021-04-18 03:52:17.994000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockQueuecallReferenceHandler.EXPECT().Get(gomock.Any(), tt.referenceID).Return(tt.queuecallReference, nil)
			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.queuecallReference.CurrentQueuecallID).Return(tt.queuecall, nil)
			mockDB.EXPECT().QueuecallSetStatusService(gomock.Any(), tt.queuecall.ID).Return(nil)
			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.queuecall.ID).Return(tt.queuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), queuecall.EventTypeQueuecallServiced, tt.queuecall.WebhookURI, tt.queuecall)

			duration := getDuration(ctx, tt.responseQueuecall.TMCreate, tt.responseQueuecall.TMService)
			mockDB.EXPECT().QueueIncreaseTotalServicedCount(gomock.Any(), tt.responseQueuecall.QueueID, tt.responseQueuecall.ID, duration)

			if tt.queuecall.TimeoutService > 0 {
				mockReq.EXPECT().QMV1QueuecallTiemoutService(gomock.Any(), tt.queuecall.ID, tt.queuecall.TimeoutService).Return(nil)
			}

			h.Joined(ctx, tt.referenceID, tt.confbridgeID)
		})
	}
}
