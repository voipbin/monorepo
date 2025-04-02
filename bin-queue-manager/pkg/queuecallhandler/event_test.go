package queuecallhandler

import (
	"context"
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				},
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
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
			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().QueuecallSetStatusAbandoned(ctx, tt.responseQueuecall.ID, gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallAbandoned, tt.responseQueuecall)
			mockQueue.EXPECT().RemoveQueuecallID(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID).Return(&queue.Queue{}, nil)

			mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.responseQueuecall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				},
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b5bbd69e-5ef9-11ec-a39e-73a3a50a1e26"),
				},
				QueueID:         uuid.FromStringOrNil("c935c7d0-5edf-11ec-8d87-5b567f32807e"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("3d626154-5ef9-11ec-9406-77e6457e61c9"),
				ForwardActionID: uuid.FromStringOrNil("bedfbc86-5ee0-11ec-a327-cbb8abfda595"),
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
			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().QueuecallSetStatusDone(ctx, tt.responseQueuecall.ID, gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallDone, tt.responseQueuecall)
			mockQueue.EXPECT().RemoveQueuecallID(ctx, tt.responseQueuecall.QueueID, tt.responseQueuecall.ID).Return(&queue.Queue{}, nil)

			mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.responseQueuecall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
			mockReq.EXPECT().FlowV1VariableDeleteVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			h.EventCallConfbridgeLeaved(ctx, tt.referenceID, tt.conferenceID)
		})
	}
}

func Test_EventCUCustomerDeleted(t *testing.T) {
	tests := []struct {
		name string

		customer *cucustomer.Customer

		responseQueuecalls []*queuecall.Queuecall

		expectFilters map[string]string
	}{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("51813b9e-f08b-11ee-ae42-e79a06af2749"),
			},

			responseQueuecalls: []*queuecall.Queuecall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6cf821a8-f08b-11ee-ba34-87171b9d8aec"),
					},
					Status: queuecall.StatusDone,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6d6aca96-f08b-11ee-8eda-b357cf02292b"),
					},
					Status: queuecall.StatusDone,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6d999c9a-f08b-11ee-a47a-03dddb8092f7"),
					},
					Status: queuecall.StatusDone,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6dccccc8-f08b-11ee-a0e6-1fccc2222158"),
					},
					Status: queuecall.StatusDone,
				},
			},

			expectFilters: map[string]string{
				"customer_id": "51813b9e-f08b-11ee-ae42-e79a06af2749",
				"deleted":     "false",
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

			mockDB.EXPECT().QueuecallGets(ctx, uint64(1000), "", tt.expectFilters).Return(tt.responseQueuecalls, nil)

			// kick
			for _, qc := range tt.responseQueuecalls {
				mockDB.EXPECT().QueuecallGet(ctx, qc.ID).Return(qc, nil)
			}

			// delete
			for _, qc := range tt.responseQueuecalls {
				mockDB.EXPECT().QueuecallDelete(ctx, qc.ID).Return(nil)
				mockDB.EXPECT().QueuecallGet(ctx, qc.ID).Return(qc, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, qc.CustomerID, queuecall.EventTypeQueuecallDeleted, qc)
			}

			if errDelete := h.EventCUCustomerDeleted(ctx, tt.customer); errDelete != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDelete)
			}
		})
	}
}
