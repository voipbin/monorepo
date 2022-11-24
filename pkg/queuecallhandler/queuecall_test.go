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

func Test_GetsByCustomerID(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string

		response []*queuecall.Queuecall

		expectRes []*queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("073e9dfe-7f56-11ec-97c6-a7b797137c40"),
			1000,
			"2021-04-18 03:22:17.994000",

			[]*queuecall.Queuecall{
				{
					ID: uuid.FromStringOrNil("3dc05a40-6401-11ec-a3f4-db880e583b3d"),
				},
			},

			[]*queuecall.Queuecall{
				{
					ID: uuid.FromStringOrNil("3dc05a40-6401-11ec-a3f4-db880e583b3d"),
				},
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

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallGetsByCustomerID(gomock.Any(), tt.customerID, tt.size, tt.token).Return(tt.response, nil)

			res, err := h.GetsByCustomerID(ctx, tt.customerID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_GetsByQueueIDAndStatus(t *testing.T) {
	tests := []struct {
		name string

		queueID uuid.UUID
		status  queuecall.Status
		size    uint64
		token   string

		response []*queuecall.Queuecall

		expectRes []*queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("14faed80-d140-11ec-9255-8f61c2013693"),
			queuecall.StatusWaiting,
			1000,
			"2021-04-18 03:22:17.994000",

			[]*queuecall.Queuecall{
				{
					ID: uuid.FromStringOrNil("15816982-d140-11ec-9736-4f2812aeda51"),
				},
			},

			[]*queuecall.Queuecall{
				{
					ID: uuid.FromStringOrNil("15816982-d140-11ec-9736-4f2812aeda51"),
				},
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

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallGetsByQueueIDAndStatus(gomock.Any(), tt.queueID, tt.status, tt.size, tt.token).Return(tt.response, nil)

			res, err := h.GetsByQueueIDAndStatus(ctx, tt.queueID, tt.status, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {
	tests := []struct {
		name string

		queuecallID uuid.UUID

		response *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("857cd7b4-6401-11ec-b348-371db9f3524c"),

			&queuecall.Queuecall{

				ID: uuid.FromStringOrNil("857cd7b4-6401-11ec-b348-371db9f3524c"),
			},

			&queuecall.Queuecall{

				ID: uuid.FromStringOrNil("857cd7b4-6401-11ec-b348-371db9f3524c"),
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

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.queuecallID).Return(tt.response, nil)

			res, err := h.Get(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_GetByReferenceID(t *testing.T) {

	tests := []struct {
		name string

		referenceID uuid.UUID

		responseQueuecallReference *queuecallreference.QueuecallReference
		response                   *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("ba4d1864-6401-11ec-8970-97b9f94d41cf"),

			&queuecallreference.QueuecallReference{
				ID:                 uuid.FromStringOrNil("ba4d1864-6401-11ec-8970-97b9f94d41cf"),
				CurrentQueuecallID: uuid.FromStringOrNil("dc96c7e4-6401-11ec-87e2-0b5e8ae66d96"),
			},
			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("dc96c7e4-6401-11ec-87e2-0b5e8ae66d96"),
			},

			&queuecall.Queuecall{

				ID: uuid.FromStringOrNil("dc96c7e4-6401-11ec-87e2-0b5e8ae66d96"),
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
			mockQueuecallReference := queuecallreferencehandler.NewMockQueuecallReferenceHandler(mc)

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,

				queuecallReferenceHandler: mockQueuecallReference,
			}

			ctx := context.Background()

			mockQueuecallReference.EXPECT().Get(gomock.Any(), tt.referenceID).Return(tt.responseQueuecallReference, nil)
			mockDB.EXPECT().QueuecallGet(gomock.Any(), tt.responseQueuecallReference.CurrentQueuecallID).Return(tt.response, nil)

			res, err := h.GetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		queue *queue.Queue

		referenceType         queuecall.ReferenceType
		referenceID           uuid.UUID
		referenceActiveflowID uuid.UUID

		flowID          uuid.UUID
		forwardActionID uuid.UUID
		exitActionID    uuid.UUID
		conferenceID    uuid.UUID

		source commonaddress.Address

		queuecall *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			&queue.Queue{
				ID:         uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
				CustomerID: uuid.FromStringOrNil("c910ccc8-7f55-11ec-9c6e-a356bdf34421"),

				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
				},

				WaitTimeout:    100000,
				ServiceTimeout: 1000000,
			},

			queuecall.ReferenceTypeCall,
			uuid.FromStringOrNil("a875b472-5e5a-11ec-9467-8f2c600000f3"),
			uuid.FromStringOrNil("28063f02-af52-11ec-9025-6775fa083464"),

			uuid.FromStringOrNil("c9e87138-7699-11ec-aa80-0321af12db91"),
			uuid.FromStringOrNil("a89d0acc-5e5a-11ec-8f3b-274070e9fa26"),
			uuid.FromStringOrNil("a8bd43fa-5e5a-11ec-8e43-236c955d6691"),
			uuid.FromStringOrNil("a8dca420-5e5a-11ec-87e3-eff5c9e3d170"),

			commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821021656521",
			},

			&queuecall.Queuecall{
				Source:      commonaddress.Address{},
				TagIDs:      []uuid.UUID{},
				TimeoutWait: 100000,
			},
			&queuecall.Queuecall{
				Source:      commonaddress.Address{},
				TagIDs:      []uuid.UUID{},
				TimeoutWait: 100000,
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

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(tt.queuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.queuecall.CustomerID, queuecall.EventTypeQueuecallCreated, tt.queuecall)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			if tt.queue.WaitTimeout > 0 {
				mockReq.EXPECT().QueueV1QueuecallTimeoutWait(gomock.Any(), gomock.Any(), tt.queue.WaitTimeout).Return(nil)
			}

			res, err := h.Create(
				ctx,
				tt.queue,
				tt.referenceType,
				tt.referenceID,
				tt.referenceActiveflowID,
				tt.flowID,
				tt.forwardActionID,
				tt.exitActionID,
				tt.conferenceID,
				tt.source,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_UpdateStatusConnecting(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID
		agentID     uuid.UUID

		responseQueuecall *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("ca971d82-d1ca-11ec-9291-ebaa2b055c3a"),
			uuid.FromStringOrNil("cad009da-d1ca-11ec-ae58-b780e0d24f05"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("ca971d82-d1ca-11ec-9291-ebaa2b055c3a"),
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

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallSetStatusConnecting(ctx, tt.queuecallID, tt.agentID).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.queuecallID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallConnecting, tt.responseQueuecall)

			res, err := h.UpdateStatusConnecting(ctx, tt.queuecallID, tt.agentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueuecall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueuecall, res)
			}
		})
	}
}

func Test_UpdateStatusWaiting(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("1713ed3e-d1cb-11ec-b70b-3f756e5181f3"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("1713ed3e-d1cb-11ec-b70b-3f756e5181f3"),
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

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallSetStatusWaiting(ctx, tt.queuecallID).Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.queuecallID).Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallWaiting, tt.responseQueuecall)
			mockReq.EXPECT().QueueV1QueueUpdateExecute(ctx, tt.responseQueuecall.QueueID, queue.ExecuteRun).Return(&queue.Queue{}, nil).AnyTimes()

			res, err := h.UpdateStatusWaiting(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueuecall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueuecall, res)
			}
		})
	}
}
