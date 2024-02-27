package queuecallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuehandler"
)

func Test_Kick(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
		responseCurTime   string

		expectDurationWaiting int
	}{
		{
			"status is waiting",

			uuid.FromStringOrNil("101b66d0-d1b5-11ec-b1fc-03d1a45f37e3"),

			&queuecall.Queuecall{
				ID:                    uuid.FromStringOrNil("101b66d0-d1b5-11ec-b1fc-03d1a45f37e3"),
				QueueID:               uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:         queuecall.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ReferenceActiveflowID: uuid.FromStringOrNil("f1268a84-bcc3-11ed-8326-9bd295279b92"),
				ForwardActionID:       uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:          uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:          uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9ca8282-5edf-11ec-a876-df977791e643"),
				},

				Status:    queuecall.StatusWaiting,
				TMCreate:  "2021-04-18 03:22:17.994000",
				TMService: "2021-04-18 03:22:17.994000",
				TMDelete:  dbhandler.DefaultTimeStamp,
			},
			"2021-04-18 03:23:17.994000",

			60000,
		},
		{
			"status is service",

			uuid.FromStringOrNil("07beb726-d291-11ec-b24c-9b1e9b9bd80b"),

			&queuecall.Queuecall{
				ID:                    uuid.FromStringOrNil("07beb726-d291-11ec-b24c-9b1e9b9bd80b"),
				QueueID:               uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:         queuecall.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ReferenceActiveflowID: uuid.FromStringOrNil("fbe9b6b2-bcc3-11ed-895f-57ec8caa42da"),
				ForwardActionID:       uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:          uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:          uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
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
				TMService: "2021-04-18 03:23:17.994000",
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

			mockDB.EXPECT().QueuecallGet(ctx, tt.queuecallID).Return(tt.responseQueuecall, nil)

			mockReq.EXPECT().FlowV1ActiveflowUpdateForwardActionID(ctx, tt.responseQueuecall.ReferenceActiveflowID, tt.responseQueuecall.ExitActionID, true).Return(nil)
			if tt.responseQueuecall.Status != queuecall.StatusService {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockDB.EXPECT().QueuecallSetStatusAbandoned(ctx, tt.responseQueuecall.ID, tt.expectDurationWaiting, tt.responseCurTime)

				mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallAbandoned, tt.responseQueuecall)

				mockQueue.EXPECT().RemoveQueuecallID(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID).Return(&queue.Queue{}, nil)
				mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.responseQueuecall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
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

		responseQueuecall *queuecall.Queuecall
		responseCurTime   string
	}{
		{
			"queuecall's status is waiting",

			uuid.FromStringOrNil("04f47408-d1b6-11ec-b497-c7ab793dd73d"),

			&queuecall.Queuecall{
				ID:                    uuid.FromStringOrNil("101b66d0-d1b5-11ec-b1fc-03d1a45f37e3"),
				QueueID:               uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:         queuecall.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("04f47408-d1b6-11ec-b497-c7ab793dd73d"),
				ReferenceActiveflowID: uuid.FromStringOrNil("0f15c000-bcc4-11ed-a64f-ab5cd3031ac1"),
				ForwardActionID:       uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:          uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:          uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
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
				TMEnd:     dbhandler.DefaultTimeStamp,
				TMDelete:  dbhandler.DefaultTimeStamp,
			},
			"2021-04-18 03:22:17.994000",
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
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)

			mockReq.EXPECT().FlowV1ActiveflowUpdateForwardActionID(ctx, tt.responseQueuecall.ReferenceActiveflowID, tt.responseQueuecall.ExitActionID, true).Return(nil)
			if tt.responseQueuecall.Status != queuecall.StatusService {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockDB.EXPECT().QueuecallSetStatusAbandoned(ctx, tt.responseQueuecall.ID, gomock.Any(), tt.responseCurTime)

				mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallAbandoned, tt.responseQueuecall)

				mockQueue.EXPECT().RemoveQueuecallID(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID).Return(&queue.Queue{}, nil)
				mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.responseQueuecall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
			}
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

func Test_kickForce(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseQueuecall *queuecall.Queuecall
		responseCurTime   string
	}{
		{
			"status is waiting",

			uuid.FromStringOrNil("33fb29fa-d541-11ee-aa2a-1353084c4f20"),

			&queuecall.Queuecall{
				ID:                    uuid.FromStringOrNil("33fb29fa-d541-11ee-aa2a-1353084c4f20"),
				QueueID:               uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:         queuecall.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("04f47408-d1b6-11ec-b497-c7ab793dd73d"),
				ReferenceActiveflowID: uuid.FromStringOrNil("0f15c000-bcc4-11ed-a64f-ab5cd3031ac1"),
				ForwardActionID:       uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:          uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:          uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
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
				TMEnd:     dbhandler.DefaultTimeStamp,
				TMDelete:  dbhandler.DefaultTimeStamp,
			},
			"2021-04-18 03:22:17.994000",
		},
		{
			"status is service",

			uuid.FromStringOrNil("610b8dc6-d542-11ee-a00a-e77abee6dec8"),

			&queuecall.Queuecall{
				ID:                    uuid.FromStringOrNil("610b8dc6-d542-11ee-a00a-e77abee6dec8"),
				QueueID:               uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:         queuecall.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("04f47408-d1b6-11ec-b497-c7ab793dd73d"),
				ReferenceActiveflowID: uuid.FromStringOrNil("0f15c000-bcc4-11ed-a64f-ab5cd3031ac1"),
				ForwardActionID:       uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
				ExitActionID:          uuid.FromStringOrNil("d708bbbe-5ee0-11ec-aca3-530babc708dd"),
				ConfbridgeID:          uuid.FromStringOrNil("ece5e716-5efb-11ec-a6ad-3fe3ed6844cb"),
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
				TMDelete:  dbhandler.DefaultTimeStamp,
			},
			"2021-04-18 03:22:17.994000",
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

			mockDB.EXPECT().QueuecallGet(ctx, tt.id).Return(tt.responseQueuecall, nil)

			mockReq.EXPECT().FlowV1ActiveflowUpdateForwardActionID(ctx, tt.responseQueuecall.ReferenceActiveflowID, tt.responseQueuecall.ExitActionID, true).Return(nil)

			if tt.responseQueuecall.Status == queuecall.StatusService {
				// update status done
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockDB.EXPECT().QueuecallSetStatusDone(ctx, tt.responseQueuecall.ID, gomock.Any(), tt.responseCurTime).Return(nil)

				mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallDone, tt.responseQueuecall)

				mockQueue.EXPECT().RemoveQueuecallID(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID).Return(&queue.Queue{}, nil)
				mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.responseQueuecall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)

				mockReq.EXPECT().FlowV1VariableDeleteVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			} else {
				// update status abandoned
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockDB.EXPECT().QueuecallSetStatusAbandoned(ctx, tt.responseQueuecall.ID, gomock.Any(), tt.responseCurTime).Return(nil)

				mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallAbandoned, tt.responseQueuecall)

				mockQueue.EXPECT().RemoveQueuecallID(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID).Return(&queue.Queue{}, nil)
				mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.responseQueuecall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)

				mockReq.EXPECT().FlowV1VariableDeleteVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			}

			res, err := h.kickForce(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueuecall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseQueuecall, res)
			}
		})
	}
}
