package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestTagCreate(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		tagName  string
		detail   string

		response  *amtag.Tag
		expectRes *amtag.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"test1 name",
			"test1 detail",

			&amtag.Tag{
				ID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			},
			&amtag.WebhookMessage{
				ID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().AMV1TagCreate(gomock.Any(), tt.customer.ID, tt.tagName, tt.detail).Return(tt.response, nil)

			res, err := h.TagCreate(tt.customer, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func TestTagGets(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		size     uint64
		token    string

		response  []amtag.Tag
		expectRes []*amtag.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]amtag.Tag{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
			[]*amtag.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
			},
		},
		{
			"2 results",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]amtag.Tag{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
				{
					ID: uuid.FromStringOrNil("c0f620ee-4fbf-11ec-87b2-7372cbac1bb0"),
				},
			},
			[]*amtag.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				},
				{
					ID: uuid.FromStringOrNil("c0f620ee-4fbf-11ec-87b2-7372cbac1bb0"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().AMV1TagGets(gomock.Any(), tt.customer.ID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.TagGets(tt.customer, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func TestTagGet(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		tagID    uuid.UUID

		response  *amtag.Tag
		expectRes *amtag.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),

			&amtag.Tag{
				ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&amtag.WebhookMessage{
				ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().AMV1TagGet(gomock.Any(), tt.tagID).Return(tt.response, nil)

			res, err := h.TagGet(tt.customer, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func TestTagDelete(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		tagID    uuid.UUID

		resTagGet *amtag.Tag
		expectRes *amtag.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),

			&amtag.Tag{
				ID:         uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&amtag.WebhookMessage{
				ID: uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().AMV1TagGet(gomock.Any(), tt.tagID).Return(tt.resTagGet, nil)
			mockReq.EXPECT().AMV1TagDelete(gomock.Any(), tt.tagID).Return(tt.resTagGet, nil)

			res, err := h.TagDelete(tt.customer, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestTagUpdate(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		tagID    uuid.UUID
		tagName  string
		detail   string

		resTagGet *amtag.Tag
		expectRes *amtag.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
			"test1",
			"detail",

			&amtag.Tag{
				ID:         uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&amtag.WebhookMessage{
				ID: uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().AMV1TagGet(gomock.Any(), tt.tagID).Return(tt.resTagGet, nil)
			mockReq.EXPECT().AMV1TagUpdate(gomock.Any(), tt.tagID, tt.tagName, tt.detail).Return(tt.resTagGet, nil)

			res, err := h.TagUpdate(tt.customer, tt.tagID, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
