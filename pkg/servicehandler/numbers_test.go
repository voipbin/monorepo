package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestOrderNumberGets(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []nmnumber.Number
		expectRes []*nmnumber.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"2021-03-01 01:00:00.995000",
			10,

			[]nmnumber.Number{
				{
					ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
				},
			},
			[]*nmnumber.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
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
			ctx := context.Background()

			mockReq.EXPECT().NumberV1NumberGets(ctx, tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)

			res, err := h.NumberGets(ctx, tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, num := range res {
				num.TMCreate = ""
				num.TMUpdate = ""
				num.TMDelete = ""
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func TestOrderNumberGet(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer
		id       uuid.UUID

		response  *nmnumber.Number
		expectRes *nmnumber.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),

			&nmnumber.Number{
				ID:         uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				TMDelete:   defaultTimestamp,
			},
			&nmnumber.WebhookMessage{
				ID:         uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				TMDelete:   defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().NumberV1NumberGet(ctx, tt.id).Return(tt.response, nil)

			res, err := h.NumberGet(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestOrderNumberGetError(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer
		id       uuid.UUID

		response *nmnumber.Number
	}

	tests := []test{
		{
			"deleted item",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("b6ad4c06-7c99-11eb-b2c9-fbe9ecb397e0"),

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("b6ad4c06-7c99-11eb-b2c9-fbe9ecb397e0"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            "2021-03-02 01:00:00.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().NumberV1NumberGet(ctx, tt.id).Return(tt.response, nil)

			_, err := h.NumberGet(ctx, tt.customer, tt.id)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func TestNumberCreate(t *testing.T) {

	tests := []struct {
		name     string
		customer *cscustomer.Customer

		num           string
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		numberName    string
		detail        string

		response  *nmnumber.Number
		expectRes *nmnumber.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			"+821021656521",
			uuid.FromStringOrNil("c7301f68-88af-11ec-bb03-33d26b9b7e37"),
			uuid.FromStringOrNil("4872b1e4-a881-11ec-b15b-efa630b95991"),
			"test name",
			"test detail",

			&nmnumber.Number{
				ID: uuid.FromStringOrNil("f06c8c36-7b1d-11eb-8b01-83e94e91b409"),
			},
			&nmnumber.WebhookMessage{
				ID: uuid.FromStringOrNil("f06c8c36-7b1d-11eb-8b01-83e94e91b409"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().NumberV1NumberCreate(ctx, tt.customer.ID, tt.num, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail).Return(tt.response, nil)
			res, err := h.NumberCreate(ctx, tt.customer, tt.num, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestNumberDelete(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer
		id       uuid.UUID

		responseGet    *nmnumber.Number
		responseDelete *nmnumber.Number
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusDeleted,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMCreate:            "2021-10-15 00:00:00.000001",
				TMUpdate:            "2021-10-16 00:00:00.000001",
				TMDelete:            "2021-10-16 00:00:00.000001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().NumberV1NumberGet(ctx, tt.id).Return(tt.responseGet, nil)
			mockReq.EXPECT().NumberV1NumberDelete(ctx, tt.id).Return(tt.responseDelete, nil)

			res, err := h.NumberDelete(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseDelete.ConvertWebhookMessage()) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.responseDelete.ConvertWebhookMessage(), res)
			}
		})
	}
}

func TestNumberUpdate(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer

		id         uuid.UUID
		numberName string
		detail     string

		responseGet    *nmnumber.Number
		responseUpdate *nmnumber.Number
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
			"update name",
			"update detail",

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				CallFlowID:          uuid.FromStringOrNil("7e46cf4a-7c5d-11eb-8aa3-17a63e21c25f"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().NumberV1NumberGet(ctx, tt.id).Return(tt.responseGet, nil)
			mockReq.EXPECT().NumberV1NumberUpdateBasicInfo(ctx, tt.id, tt.numberName, tt.detail).Return(tt.responseUpdate, nil)

			res, err := h.NumberUpdate(ctx, tt.customer, tt.id, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseUpdate.ConvertWebhookMessage()) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.responseUpdate.ConvertWebhookMessage(), res)
			}
		})
	}
}

func TestNumberUpdateError(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer

		id         uuid.UUID
		numberName string
		detail     string

		responseGet *nmnumber.Number
	}

	tests := []test{
		{
			"deleted item",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
			"update name",
			"update detail",

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            "2021-03-02 01:00:00.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().NumberV1NumberGet(ctx, tt.id).Return(tt.responseGet, nil)

			_, err := h.NumberUpdate(ctx, tt.customer, tt.id, tt.numberName, tt.detail)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}
