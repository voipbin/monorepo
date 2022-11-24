package queuecallhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallreferencehandler"
)

func Test_Leaved(t *testing.T) {
	tests := []struct {
		name string

		referenceID  uuid.UUID
		conferenceID uuid.UUID

		responseQueuecallReference *queuecallreference.QueuecallReference
		responseQueuecall          *queuecall.Queuecall
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
				ConferenceID:    uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
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
				TMDelete:  "2021-04-18 03:52:17.994000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockQueuecallMasterHandler := queuecallreferencehandler.NewMockQueuecallReferenceHandler(mc)

			h := &queuecallHandler{
				db:                        mockDB,
				reqHandler:                mockReq,
				notifyhandler:             mockNotify,
				queuecallReferenceHandler: mockQueuecallMasterHandler,
			}

			ctx := context.Background()

			mockQueuecallMasterHandler.EXPECT().Get(ctx, tt.referenceID).Return(tt.responseQueuecallReference, nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecallReference.CurrentQueuecallID).Return(tt.responseQueuecall, nil)
			mockDB.EXPECT().QueuecallSetDurationService(ctx, tt.responseQueuecall.ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallDelete(ctx, tt.responseQueuecall.ID, queuecall.StatusDone, gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecallReference.CurrentQueuecallID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallDone, tt.responseQueuecall)

			mockDB.EXPECT().QueueRemoveServiceQueueCall(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID)

			mockReq.EXPECT().FlowV1VariableDeleteVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockReq.EXPECT().ConferenceV1ConferenceDelete(ctx, tt.responseQueuecall.ConferenceID).Return(nil)

			h.Leaved(ctx, tt.referenceID, tt.conferenceID)
		})
	}
}
