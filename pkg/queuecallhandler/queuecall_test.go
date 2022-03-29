package queuecallhandler

import (
	"context"
	reflect "reflect"
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

func TestGets(t *testing.T) {
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
			ctx := context.Background()

			mockDB.EXPECT().QueuecallGets(gomock.Any(), tt.customerID, tt.size, tt.token).Return(tt.response, nil)

			res, err := h.Gets(ctx, tt.customerID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestGet(t *testing.T) {
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

func TestGetByReferenceID(t *testing.T) {
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

func TestCreate(t *testing.T) {
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

	tests := []struct {
		name string

		customerID uuid.UUID
		queueID    uuid.UUID

		referenceType         queuecall.ReferenceType
		referenceID           uuid.UUID
		referenceActiveflowID uuid.UUID

		flowID          uuid.UUID
		forwardActionID uuid.UUID
		exitActionID    uuid.UUID
		confbridgeID    uuid.UUID

		source        cmaddress.Address
		routingMethod queue.RoutingMethod
		tagIDs        []uuid.UUID

		timeoutWait    int
		timeoutService int

		queuecall *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("c910ccc8-7f55-11ec-9c6e-a356bdf34421"),
			uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),

			queuecall.ReferenceTypeCall,
			uuid.FromStringOrNil("a875b472-5e5a-11ec-9467-8f2c600000f3"),
			uuid.FromStringOrNil("28063f02-af52-11ec-9025-6775fa083464"),

			uuid.FromStringOrNil("c9e87138-7699-11ec-aa80-0321af12db91"),
			uuid.FromStringOrNil("a89d0acc-5e5a-11ec-8f3b-274070e9fa26"),
			uuid.FromStringOrNil("a8bd43fa-5e5a-11ec-8e43-236c955d6691"),
			uuid.FromStringOrNil("a8dca420-5e5a-11ec-87e3-eff5c9e3d170"),

			cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821021656521",
			},
			queue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
			},
			100000,
			1000000,

			&queuecall.Queuecall{
				Source:      cmaddress.Address{},
				TagIDs:      []uuid.UUID{},
				TimeoutWait: 100000,
			},
			&queuecall.Queuecall{
				Source:      cmaddress.Address{},
				TagIDs:      []uuid.UUID{},
				TimeoutWait: 100000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().QueuecallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(tt.queuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.queuecall.CustomerID, queuecall.EventTypeQueuecallCreated, tt.queuecall)

			if tt.timeoutWait > 0 {
				mockReq.EXPECT().QMV1QueuecallTimeoutWait(gomock.Any(), gomock.Any(), tt.timeoutWait).Return(nil)
			}

			res, err := h.Create(
				ctx,
				tt.customerID,
				tt.queueID,
				tt.referenceType,
				tt.referenceID,
				tt.referenceActiveflowID,
				tt.flowID,
				tt.forwardActionID,
				tt.exitActionID,
				tt.confbridgeID,
				tt.source,
				tt.routingMethod,
				tt.tagIDs,
				tt.timeoutWait,
				tt.timeoutService,
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
