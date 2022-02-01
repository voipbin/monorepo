package messagetargethandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/messagetarget"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
)

func TestGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := messagetargetHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string
		id   uuid.UUID

		responseGet *messagetarget.MessageTarget

		expectRes *messagetarget.MessageTarget
	}{
		{
			"normal",
			uuid.FromStringOrNil("6515992a-833c-11ec-b53e-ff69ef240833"),
			&messagetarget.MessageTarget{
				ID:            uuid.FromStringOrNil("6515992a-833c-11ec-b53e-ff69ef240833"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},

			&messagetarget.MessageTarget{
				ID:            uuid.FromStringOrNil("6515992a-833c-11ec-b53e-ff69ef240833"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().MessageTargetGet(gomock.Any(), tt.id).Return(tt.responseGet, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func TestGetErrorDB(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := messagetargetHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string
		id   uuid.UUID

		responseGet *cscustomer.Customer
	}{
		{
			"normal",
			uuid.FromStringOrNil("bfe9d244-833c-11ec-943e-67954ab4201c"),
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("bfe9d244-833c-11ec-943e-67954ab4201c"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().MessageTargetGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf(""))
			mockReq.EXPECT().CSV1CustomerGet(gomock.Any(), tt.id).Return(tt.responseGet, nil)

			tmp := messagetarget.CreateMessageTargetFromCustomer(tt.responseGet)
			mockDB.EXPECT().MessageTargetSet(gomock.Any(), tmp).Return(nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tmp) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tmp, res)
			}

		})
	}
}

func TestUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := messagetargetHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string
		data *messagetarget.MessageTarget
	}{
		{
			"normal",
			&messagetarget.MessageTarget{
				ID:            uuid.FromStringOrNil("6515992a-833c-11ec-b53e-ff69ef240833"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().MessageTargetSet(gomock.Any(), tt.data).Return(nil)

			if err := h.Update(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestUpdateByCustomer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := messagetargetHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string

		customerInfo *cscustomer.Customer
		data         *messagetarget.MessageTarget
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("51a5325a-833d-11ec-8759-c79414e8a44e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
			&messagetarget.MessageTarget{
				ID:            uuid.FromStringOrNil("51a5325a-833d-11ec-8759-c79414e8a44e"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tmp := messagetarget.CreateMessageTargetFromCustomer(tt.customerInfo)
			mockDB.EXPECT().MessageTargetSet(gomock.Any(), tmp).Return(nil)

			res, err := h.UpdateByCustomer(ctx, tt.customerInfo)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.data) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.data, res)
			}

		})
	}
}
