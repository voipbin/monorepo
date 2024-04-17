package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"

	fmflow "monorepo/bin-flow-manager/models/flow"

	nmnumber "monorepo/bin-number-manager/models/number"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_OrderNumberGets(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		pageToken string
		pageSize  uint64

		response []nmnumber.Number

		expectFilters map[string]string
		expectRes     []*nmnumber.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			"2021-03-01 01:00:00.995000",
			10,

			[]nmnumber.Number{
				{
					ID: uuid.FromStringOrNil("2130337e-7b1c-11eb-a431-b714a0a4b6fc"),
				},
			},

			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
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

			mockReq.EXPECT().NumberV1NumberGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.response, nil)

			res, err := h.NumberGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res[0], tt.expectRes[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_OrderNumberGet(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent
		id    uuid.UUID

		response  *nmnumber.Number
		expectRes *nmnumber.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),

			&nmnumber.Number{
				ID:         uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			&nmnumber.WebhookMessage{
				ID:         uuid.FromStringOrNil("17bd8d64-7be4-11eb-b887-8f1b24b98639"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.NumberGet(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OrderNumberGetError(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent
		id    uuid.UUID

		responseNumber *nmnumber.Number
	}

	tests := []test{
		{
			"deleted item",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("b6ad4c06-7c99-11eb-b2c9-fbe9ecb397e0"),

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("b6ad4c06-7c99-11eb-b2c9-fbe9ecb397e0"),
				CustomerID:          uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Number:              "+821021656521",
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

			mockReq.EXPECT().NumberV1NumberGet(ctx, tt.id).Return(tt.responseNumber, nil)

			_, err := h.NumberGet(ctx, tt.agent, tt.id)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func Test_NumberCreate(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

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
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
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

			mockReq.EXPECT().NumberV1NumberCreate(ctx, tt.agent.CustomerID, tt.num, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail).Return(tt.response, nil)
			res, err := h.NumberCreate(ctx, tt.agent, tt.num, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_NumberDelete(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent
		id    uuid.UUID

		responseGet    *nmnumber.Number
		responseDelete *nmnumber.Number
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),
				CustomerID:          uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Number:              "+821021656521",
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("10bd9968-7be5-11eb-9c49-7fe12b631d76"),
				CustomerID:          uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Number:              "+821021656521",
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

			res, err := h.NumberDelete(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseDelete.ConvertWebhookMessage()) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.responseDelete.ConvertWebhookMessage(), res)
			}
		})
	}
}

func Test_NumberUpdate(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		id            uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		numberName    string
		detail        string

		responseGet         *nmnumber.Number
		responseFlowCall    *fmflow.Flow
		responseFlowMessage *fmflow.Flow
		responseUpdate      *nmnumber.Number
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
			uuid.FromStringOrNil("72001c3a-2ca2-11ee-96c3-4730286893af"),
			uuid.FromStringOrNil("7240534a-2ca2-11ee-bb9a-8f1c5dafa508"),
			"update name",
			"update detail",

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				CustomerID:          uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Number:              "+821021656521",
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMDelete:            defaultTimestamp,
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("72001c3a-2ca2-11ee-96c3-4730286893af"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("7240534a-2ca2-11ee-bb9a-8f1c5dafa508"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				CustomerID:          uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				CallFlowID:          uuid.FromStringOrNil("7e46cf4a-7c5d-11eb-8aa3-17a63e21c25f"),
				Number:              "+821021656521",
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
			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.callFlowID).Return(tt.responseFlowCall, nil)
			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.messageFlowID).Return(tt.responseFlowMessage, nil)
			mockReq.EXPECT().NumberV1NumberUpdate(ctx, tt.id, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail).Return(tt.responseUpdate, nil)

			res, err := h.NumberUpdate(ctx, tt.agent, tt.id, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseUpdate.ConvertWebhookMessage()) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.responseUpdate.ConvertWebhookMessage(), res)
			}
		})
	}
}

func Test_NumberUpdateError(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		id            uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		numberName    string
		detail        string

		responseGet *nmnumber.Number
	}

	tests := []test{
		{
			"deleted item",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
			uuid.FromStringOrNil("bfa09172-2ca2-11ee-88a7-775c33dab2a6"),
			uuid.FromStringOrNil("bfd41d3a-2ca2-11ee-8663-1713f43b6555"),
			"update name",
			"update detail",

			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("7c718a8e-7c5d-11eb-8d3d-63ea567a6da9"),
				CustomerID:          uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Number:              "+821021656521",
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

			_, err := h.NumberUpdate(ctx, tt.agent, tt.id, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func Test_NumberRenew(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		tmRenew string

		responseNumbers []nmnumber.Number
		expectRes       []*nmnumber.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},

			tmRenew: "2021-03-02 01:00:00.995000",

			responseNumbers: []nmnumber.Number{
				{
					ID: uuid.FromStringOrNil("92647ae8-161d-11ee-9746-6387778bd96f"),
				},
				{
					ID: uuid.FromStringOrNil("92d0da8a-161d-11ee-924f-ab88344e1aa3"),
				},
			},
			expectRes: []*nmnumber.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("92647ae8-161d-11ee-9746-6387778bd96f"),
				},
				{
					ID: uuid.FromStringOrNil("92d0da8a-161d-11ee-924f-ab88344e1aa3"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().NumberV1NumberRenewByTmRenew(ctx, tt.tmRenew).Return(tt.responseNumbers, nil)

			res, err := h.NumberRenew(ctx, tt.agent, tt.tmRenew)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_numberVerifyFlow(t *testing.T) {

	type test struct {
		name string

		agent  *amagent.Agent
		flowID uuid.UUID

		responseFlow *fmflow.Flow

		expectRes bool
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("5a2ac626-d57d-11ee-ad66-1f8e01bf7985"),
				CustomerID: uuid.FromStringOrNil("5a64f30a-d57d-11ee-bf12-0f63b531e815"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			flowID: uuid.FromStringOrNil("3abc48b4-d57d-11ee-bbca-a713327af69d"),

			responseFlow: &fmflow.Flow{
				ID:         uuid.FromStringOrNil("3abc48b4-d57d-11ee-bbca-a713327af69d"),
				CustomerID: uuid.FromStringOrNil("5a64f30a-d57d-11ee-bf12-0f63b531e815"),
				TMDelete:   defaultTimestamp,
			},
			expectRes: true,
		},
		{
			name: "flow id nil",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("5a2ac626-d57d-11ee-ad66-1f8e01bf7985"),
				CustomerID: uuid.FromStringOrNil("5a64f30a-d57d-11ee-bf12-0f63b531e815"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			flowID: uuid.Nil,

			expectRes: true,
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

			if tt.flowID != uuid.Nil {
				mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.responseFlow, nil)
			}

			res := h.numberVerifyFlow(ctx, tt.agent, tt.flowID)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
