package requesthandler

import (
	"context"
	"reflect"
	"testing"

	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_NumberV1NumberCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		num           string
		numberName    string
		detail        string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *nmnumber.Number
	}{
		{
			"normal",

			uuid.FromStringOrNil("b7041f62-7ff5-11ec-b1dd-d7e05b3c5096"),
			uuid.FromStringOrNil("55b69e86-881c-11ec-8901-3b828e31a38d"),
			uuid.FromStringOrNil("7cfce5fa-a873-11ec-b620-577094655392"),
			"+821021656521",
			"test name",
			"test detail",

			"bin-manager.number-manager.request",
			&sock.Request{
				URI:      "/v1/numbers",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"b7041f62-7ff5-11ec-b1dd-d7e05b3c5096","number":"+821021656521","call_flow_id":"55b69e86-881c-11ec-8901-3b828e31a38d","message_flow_id":"7cfce5fa-a873-11ec-b620-577094655392","name":"test name","detail":"test detail"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3eda6a34-7b17-11eb-a2fa-8f4c0fd14c20"}`),
			},
			&nmnumber.Number{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("3eda6a34-7b17-11eb-a2fa-8f4c0fd14c20"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NumberV1NumberCreate(ctx, tt.customerID, tt.num, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_NumberV1NumberList(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[nmnumber.Field]any

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult []nmnumber.Number
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000Z",
			10,
			map[nmnumber.Field]any{
				nmnumber.FieldCustomerID: uuid.FromStringOrNil("b7041f62-7ff5-11ec-b1dd-d7e05b3c5096"),
			},

			"bin-manager.number-manager.request",
			&sock.Request{
				URI:      "/v1/numbers?page_token=2020-09-20T03%3A23%3A20.995000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"b7041f62-7ff5-11ec-b1dd-d7e05b3c5096"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0e00bb78-7b19-11eb-a238-9f1154b2c92e","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","customer_id":"b7041f62-7ff5-11ec-b1dd-d7e05b3c5096","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
			[]nmnumber.Number{
				{
					Identity: identity.Identity{
						ID:         uuid.FromStringOrNil("0e00bb78-7b19-11eb-a238-9f1154b2c92e"),
						CustomerID: uuid.FromStringOrNil("b7041f62-7ff5-11ec-b1dd-d7e05b3c5096"),
					},
					Number:              "+821021656521",
					ProviderName:        "telnyx",
					ProviderReferenceID: "",
					Status:              nmnumber.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
					TMPurchase:          "",
					TMCreate:            "",
					TMUpdate:            "",
					TMDelete:            "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NumberV1NumberList(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_NumberV1NumberGet(t *testing.T) {

	tests := []struct {
		name string

		numberID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *nmnumber.Number
	}{
		{
			"normal",

			uuid.FromStringOrNil("74a2f4bc-7be2-11eb-bb71-c767ac6ed931"),

			"bin-manager.number-manager.request",
			&sock.Request{
				URI:      "/v1/numbers/74a2f4bc-7be2-11eb-bb71-c767ac6ed931",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"74a2f4bc-7be2-11eb-bb71-c767ac6ed931","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","customer_id":"b7041f62-7ff5-11ec-b1dd-d7e05b3c5096","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&nmnumber.Number{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("74a2f4bc-7be2-11eb-bb71-c767ac6ed931"),
					CustomerID: uuid.FromStringOrNil("b7041f62-7ff5-11ec-b1dd-d7e05b3c5096"),
				},
				Number:              "+821021656521",
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMPurchase:          "",
				TMCreate:            "",
				TMUpdate:            "",
				TMDelete:            "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NumberV1NumberGet(ctx, tt.numberID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_NumberV1NumberDelete(t *testing.T) {

	tests := []struct {
		name string

		numberID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *nmnumber.Number
	}{
		{
			"normal",

			uuid.FromStringOrNil("aa0b1c7e-7be2-11eb-89f2-a7882f79d5b5"),

			"bin-manager.number-manager.request",
			&sock.Request{
				URI:      "/v1/numbers/aa0b1c7e-7be2-11eb-89f2-a7882f79d5b5",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"aa0b1c7e-7be2-11eb-89f2-a7882f79d5b5","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","customer_id":"b7041f62-7ff5-11ec-b1dd-d7e05b3c5096","provider_name":"telnyx","provider_reference_id":"","status":"deleted","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&nmnumber.Number{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("aa0b1c7e-7be2-11eb-89f2-a7882f79d5b5"),
					CustomerID: uuid.FromStringOrNil("b7041f62-7ff5-11ec-b1dd-d7e05b3c5096"),
				},
				Number:              "+821021656521",
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusDeleted,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMPurchase:          "",
				TMCreate:            "",
				TMUpdate:            "",
				TMDelete:            "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NumberV1NumberDelete(ctx, tt.numberID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_NumberV1NumberUpdate(t *testing.T) {
	tests := []struct {
		name string

		id            uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		numberName    string
		detail        string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *nmnumber.Number
	}{
		{
			"normal",

			uuid.FromStringOrNil("d3877fec-7c5b-11eb-bb46-07fe08c74815"),
			uuid.FromStringOrNil("338f6098-2c7d-11ee-86a3-67a8ca2722ce"),
			uuid.FromStringOrNil("33f5363e-2c7d-11ee-ba15-0762eae47333"),
			"test name",
			"test detail",

			"bin-manager.number-manager.request",
			&sock.Request{
				URI:      "/v1/numbers/d3877fec-7c5b-11eb-bb46-07fe08c74815",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_flow_id":"338f6098-2c7d-11ee-86a3-67a8ca2722ce","message_flow_id":"33f5363e-2c7d-11ee-ba15-0762eae47333","name":"test name","detail":"test detail"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d3877fec-7c5b-11eb-bb46-07fe08c74815","number":"+821021656521","call_flow_id":"d45aae76-7c5b-11eb-9542-eb46d11b1c1a","message_flow_id":"b409020e-a873-11ec-bce6-3fcf97b72d44","customer_id":"b7041f62-7ff5-11ec-b1dd-d7e05b3c5096","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&nmnumber.Number{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("d3877fec-7c5b-11eb-bb46-07fe08c74815"),
					CustomerID: uuid.FromStringOrNil("b7041f62-7ff5-11ec-b1dd-d7e05b3c5096"),
				},
				CallFlowID:          uuid.FromStringOrNil("d45aae76-7c5b-11eb-9542-eb46d11b1c1a"),
				MessageFlowID:       uuid.FromStringOrNil("b409020e-a873-11ec-bce6-3fcf97b72d44"),
				Number:              "+821021656521",
				ProviderName:        "telnyx",
				ProviderReferenceID: "",
				Status:              nmnumber.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMPurchase:          "",
				TMCreate:            "",
				TMUpdate:            "",
				TMDelete:            "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NumberV1NumberUpdate(ctx, tt.id, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_NumberV1NumberUpdateFlowID(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *nmnumber.Number
	}{
		{
			"normal",

			uuid.FromStringOrNil("d3877fec-7c5b-11eb-bb46-07fe08c74815"),
			uuid.FromStringOrNil("5f69889c-881e-11ec-b32e-93104f30aa92"),
			uuid.FromStringOrNil("d04e2a5c-a873-11ec-b16f-23f1e4cf842e"),

			"bin-manager.number-manager.request",
			&sock.Request{
				URI:      "/v1/numbers/d3877fec-7c5b-11eb-bb46-07fe08c74815/flow_ids",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_flow_id":"5f69889c-881e-11ec-b32e-93104f30aa92","message_flow_id":"d04e2a5c-a873-11ec-b16f-23f1e4cf842e"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d3877fec-7c5b-11eb-bb46-07fe08c74815"}`),
			},
			&nmnumber.Number{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d3877fec-7c5b-11eb-bb46-07fe08c74815"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NumberV1NumberUpdateFlowID(ctx, tt.id, tt.callFlowID, tt.messageFlowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_NumberV1NumberRenewByTmRenew(t *testing.T) {

	tests := []struct {
		name string

		tmRenew string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult []nmnumber.Number
	}{
		{
			name: "normal",

			tmRenew: "2021-02-26T18:26:49.000Z",

			expectTarget: "bin-manager.number-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/numbers/renew",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"tm_renew":"2021-02-26T18:26:49.000Z"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"03557c8a-1519-11ee-a0be-83a2eb2e0f67"},{"id":"0398e86c-1519-11ee-8664-773e49d76156"}]`),
			},
			expectResult: []nmnumber.Number{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("03557c8a-1519-11ee-a0be-83a2eb2e0f67"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("0398e86c-1519-11ee-8664-773e49d76156"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NumberV1NumberRenewByTmRenew(ctx, tt.tmRenew)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_NumberV1NumberRenewByDays(t *testing.T) {

	tests := []struct {
		name string

		days int

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult []nmnumber.Number
	}{
		{
			name: "normal",

			days: 3,

			expectTarget: "bin-manager.number-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/numbers/renew",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"days":3}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"26790cc8-1e3c-11ee-acde-4f8cd0d02ae0"},{"id":"26eca39a-1e3c-11ee-a66a-1b46f3425926"}]`),
			},
			expectResult: []nmnumber.Number{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("26790cc8-1e3c-11ee-acde-4f8cd0d02ae0"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("26eca39a-1e3c-11ee-a66a-1b46f3425926"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NumberV1NumberRenewByDays(ctx, tt.days)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_NumberV1NumberRenewByHours(t *testing.T) {

	tests := []struct {
		name string

		hours int

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult []nmnumber.Number
	}{
		{
			name: "normal",

			hours: 30,

			expectTarget: "bin-manager.number-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/numbers/renew",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"hours":30}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"45ebe9cc-1e3c-11ee-9fbd-3f228b2366aa"},{"id":"4610db9c-1e3c-11ee-9596-c3e05288283d"}]`),
			},
			expectResult: []nmnumber.Number{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("45ebe9cc-1e3c-11ee-9fbd-3f228b2366aa"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("4610db9c-1e3c-11ee-9596-c3e05288283d"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NumberV1NumberRenewByHours(ctx, tt.hours)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
