package queuecallhandler

import (
	"context"
	reflect "reflect"
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

func Test_Kick(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
	}{
		{
			"status is waiting",

			uuid.FromStringOrNil("101b66d0-d1b5-11ec-b1fc-03d1a45f37e3"),

			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("101b66d0-d1b5-11ec-b1fc-03d1a45f37e3"),
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

				Status:    queuecall.StatusWaiting,
				TMService: "2021-04-18 03:22:17.994000",
				TMDelete:  "2021-04-18 03:52:17.994000",
			},
		},
		{
			"status is service",

			uuid.FromStringOrNil("07beb726-d291-11ec-b24c-9b1e9b9bd80b"),

			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("07beb726-d291-11ec-b24c-9b1e9b9bd80b"),
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
			mockQueuecallReferenceHandler := queuecallreferencehandler.NewMockQueuecallReferenceHandler(mc)

			h := &queuecallHandler{
				db:                        mockDB,
				reqHandler:                mockReq,
				notifyhandler:             mockNotify,
				queuecallReferenceHandler: mockQueuecallReferenceHandler,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(ctx, tt.queuecallID).Return(tt.responseQueuecall, nil)
			mockReq.EXPECT().FlowV1ActiveflowUpdateForwardActionID(ctx, tt.responseQueuecall.ReferenceID, tt.responseQueuecall.ExitActionID, true).Return(nil)
			if tt.responseQueuecall.Status != queuecall.StatusService {
				mockDB.EXPECT().QueuecallSetDurationWaiting(ctx, tt.responseQueuecall.ID, gomock.Any()).Return(nil)
				mockDB.EXPECT().QueuecallDelete(ctx, tt.responseQueuecall.ID, queuecall.StatusAbandoned, gomock.Any()).Return(nil)
				mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallAbandoned, tt.responseQueuecall)
			}

			// deleteVariables
			mockReq.EXPECT().FlowV1VariableDeleteVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			res, err := h.Kick(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueuecall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseQueuecall, res)
			}
		})
	}
}

func Test_KickByReferenceID(t *testing.T) {

	tests := []struct {
		name string

		referenceID uuid.UUID

		responseQueuecallReference *queuecallreference.QueuecallReference
		responseQueuecall          *queuecall.Queuecall
	}{
		{
			"queuecall's status is waiting",

			uuid.FromStringOrNil("04f47408-d1b6-11ec-b497-c7ab793dd73d"),

			&queuecallreference.QueuecallReference{
				ID:                 uuid.FromStringOrNil("04f47408-d1b6-11ec-b497-c7ab793dd73d"),
				CurrentQueuecallID: uuid.FromStringOrNil("101b66d0-d1b5-11ec-b1fc-03d1a45f37e3"),
			},
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("101b66d0-d1b5-11ec-b1fc-03d1a45f37e3"),
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

				Status:    queuecall.StatusWaiting,
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
			mockQueuecallReferenceHandler := queuecallreferencehandler.NewMockQueuecallReferenceHandler(mc)

			h := &queuecallHandler{
				db:                        mockDB,
				reqHandler:                mockReq,
				notifyhandler:             mockNotify,
				queuecallReferenceHandler: mockQueuecallReferenceHandler,
			}

			ctx := context.Background()

			mockQueuecallReferenceHandler.EXPECT().Get(ctx, tt.referenceID).Return(tt.responseQueuecallReference, nil)

			// Kick
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecallReference.CurrentQueuecallID).Return(tt.responseQueuecall, nil)
			mockReq.EXPECT().FlowV1ActiveflowUpdateForwardActionID(ctx, tt.responseQueuecall.ReferenceID, tt.responseQueuecall.ExitActionID, true).Return(nil)
			mockDB.EXPECT().QueuecallSetDurationWaiting(ctx, tt.responseQueuecall.ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallDelete(ctx, tt.responseQueuecall.ID, queuecall.StatusAbandoned, gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallAbandoned, tt.responseQueuecall)

			// deleteVariables
			mockReq.EXPECT().FlowV1VariableDeleteVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			res, err := h.KickByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueuecall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseQueuecall, res)
			}
		})
	}
}
