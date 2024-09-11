package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/numberhandler"
)

func Test_processV1NumbersPost(t *testing.T) {

	type test struct {
		name string

		customerID    uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		num           string
		numberName    string
		detail        string

		createdNumber *number.Number

		request  *sock.Request
		response *sock.Response
	}

	tests := []test{
		{
			"1 number",

			uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
			uuid.FromStringOrNil("7051e796-8821-11ec-9b7d-d322b1036e7d"),
			uuid.FromStringOrNil("c5f1dffc-a866-11ec-be0a-a3c412cba4dc"),
			"+821021656521",
			"test name",
			"test detail",

			&number.Number{
				ID:                  uuid.FromStringOrNil("3a379dce-792a-11eb-a8e1-9f51cab620f8"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
				CallFlowID:          uuid.FromStringOrNil("7051e796-8821-11ec-9b7d-d322b1036e7d"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "",
				Status:              number.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
			},
			&sock.Request{
				URI:      "/v1/numbers",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "72f3b054-7ff4-11ec-9af9-0b8c5dbee258", "call_flow_id": "7051e796-8821-11ec-9b7d-d322b1036e7d", "message_flow_id": "c5f1dffc-a866-11ec-be0a-a3c412cba4dc", "number": "+821021656521", "name": "test name", "detail": "test detail"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3a379dce-792a-11eb-a8e1-9f51cab620f8","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","number":"+821021656521","call_flow_id":"7051e796-8821-11ec-9b7d-d322b1036e7d","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().Create(gomock.Any(), tt.customerID, tt.num, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail).Return(tt.createdNumber, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}

func Test_ProcessV1NumbersIDDelete(t *testing.T) {

	type test struct {
		name       string
		id         uuid.UUID
		resultData *number.Number

		request  *sock.Request
		response *sock.Response
	}

	tests := []test{
		{
			"1 number",
			uuid.FromStringOrNil("9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f"),
			&number.Number{
				ID:                  uuid.FromStringOrNil("9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "",
				Status:              number.StatusDeleted,
				T38Enabled:          false,
				EmergencyEnabled:    false,
			},
			&sock.Request{
				URI:    "/v1/numbers/9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f",
				Method: sock.RequestMethodDelete,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","number":"+821021656521","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"deleted","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.resultData, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}

func Test_ProcessV1NumbersIDGet(t *testing.T) {

	type test struct {
		name       string
		id         uuid.UUID
		resultData *number.Number

		request  *sock.Request
		response *sock.Response
	}

	tests := []test{
		{
			"1 number",
			uuid.FromStringOrNil("7b6f4caa-7a48-11eb-8b06-ff14cc60c8ad"),
			&number.Number{
				ID:                  uuid.FromStringOrNil("7b6f4caa-7a48-11eb-8b06-ff14cc60c8ad"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "",
				Status:              number.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
			},
			&sock.Request{
				URI:    "/v1/numbers/7b6f4caa-7a48-11eb-8b06-ff14cc60c8ad",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7b6f4caa-7a48-11eb-8b06-ff14cc60c8ad","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","number":"+821021656521","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().Get(gomock.Any(), tt.id).Return(tt.resultData, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}

func Test_processV1NumbersGet(t *testing.T) {

	type test struct {
		name string

		pageSize  uint64
		pageToken string

		responseFilters map[string]string
		responseNumbers []*number.Number

		request  *sock.Request
		response *sock.Response
	}

	tests := []test{
		{
			"normal",

			10,
			"2021-03-01 03:30:17.000000",

			map[string]string{
				"customer_id": "bfc9a3de-eca8-11ee-967c-87c3c0ddb3d2",
				"deleted":     "false",
			},
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("eeafd418-7a4e-11eb-8750-9bb0ca1d7926"),
					Number:              "+821021656521",
					CustomerID:          uuid.FromStringOrNil("bfc9a3de-eca8-11ee-967c-87c3c0ddb3d2"),
					Name:                "test name",
					Detail:              "test detail",
					ProviderName:        number.ProviderNameTelnyx,
					ProviderReferenceID: "",
					Status:              number.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
				},
			},
			&sock.Request{
				URI:    "/v1/numbers?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000&filter_customer_id=bfc9a3de-eca8-11ee-967c-87c3c0ddb3d2&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"eeafd418-7a4e-11eb-8750-9bb0ca1d7926","customer_id":"bfc9a3de-eca8-11ee-967c-87c3c0ddb3d2","number":"+821021656521","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 results",

			10,
			"2021-03-01 03:30:17.000000",

			map[string]string{
				"customer_id": "dcff90a8-eca8-11ee-8816-0fa58b162524",
				"deleted":     "false",
			},
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("5c18ee62-8800-11ec-bb8b-b74be365ebf2"),
					Number:              "+821100000001",
					CustomerID:          uuid.FromStringOrNil("dcff90a8-eca8-11ee-8816-0fa58b162524"),
					Name:                "test name",
					Detail:              "test detail",
					ProviderName:        number.ProviderNameTelnyx,
					ProviderReferenceID: "",
					Status:              number.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
				},
				{
					ID:                  uuid.FromStringOrNil("69dc1916-8800-11ec-8e68-e74880ae3121"),
					Number:              "+821100000002",
					CustomerID:          uuid.FromStringOrNil("dcff90a8-eca8-11ee-8816-0fa58b162524"),
					Name:                "test name",
					Detail:              "test detail",
					ProviderName:        number.ProviderNameTelnyx,
					ProviderReferenceID: "",
					Status:              number.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
				},
			},
			&sock.Request{
				URI:    "/v1/numbers?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000&filter_customer_id=dcff90a8-eca8-11ee-8816-0fa58b162524&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5c18ee62-8800-11ec-bb8b-b74be365ebf2","customer_id":"dcff90a8-eca8-11ee-8816-0fa58b162524","number":"+821100000001","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"69dc1916-8800-11ec-8e68-e74880ae3121","customer_id":"dcff90a8-eca8-11ee-8816-0fa58b162524","number":"+821100000002","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := &listenHandler{
				utilHandler:   mockUtil,
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockNumber.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseNumbers, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}

func Test_processV1NumbersIDPut(t *testing.T) {

	type test struct {
		name string

		id            uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		numberName    string
		detail        string

		resultData *number.Number

		request  *sock.Request
		response *sock.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("935190b4-7c58-11eb-8b90-f777a56fe90f"),
			uuid.FromStringOrNil("848dd8e8-20a3-11ee-bfaa-73da44e5a15c"),
			uuid.FromStringOrNil("84cbd580-20a3-11ee-81cd-b34190bda150"),
			"update name",
			"update detail",

			&number.Number{
				ID:                  uuid.FromStringOrNil("935190b4-7c58-11eb-8b90-f777a56fe90f"),
				CallFlowID:          uuid.FromStringOrNil("9394929c-7c58-11eb-8af3-13d1657955b6"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
				Name:                "update name",
				Detail:              "update detail",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "",
				Status:              number.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
			},
			&sock.Request{
				URI:      "/v1/numbers/935190b4-7c58-11eb-8b90-f777a56fe90f",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"call_flow_id":"848dd8e8-20a3-11ee-bfaa-73da44e5a15c", "message_flow_id":"84cbd580-20a3-11ee-81cd-b34190bda150","name": "update name", "detail": "update detail"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"935190b4-7c58-11eb-8b90-f777a56fe90f","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","number":"+821021656521","call_flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"update name","detail":"update detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().UpdateInfo(gomock.Any(), tt.id, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail).Return(tt.resultData, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}

func Test_processV1NumbersIDFlowIDPut(t *testing.T) {
	type test struct {
		name string

		id            uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID

		resultData *number.Number

		request  *sock.Request
		response *sock.Response
	}

	tests := []test{
		{
			"update call flow id",

			uuid.FromStringOrNil("935190b4-7c58-11eb-8b90-f777a56fe90f"),
			uuid.FromStringOrNil("9394929c-7c58-11eb-8af3-13d1657955b6"),
			uuid.Nil,

			&number.Number{
				ID:                  uuid.FromStringOrNil("935190b4-7c58-11eb-8b90-f777a56fe90f"),
				CallFlowID:          uuid.FromStringOrNil("9394929c-7c58-11eb-8af3-13d1657955b6"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
				Name:                "update name",
				Detail:              "update detail",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "",
				Status:              number.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
			},
			&sock.Request{
				URI:      "/v1/numbers/935190b4-7c58-11eb-8b90-f777a56fe90f/flow_ids",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"call_flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"935190b4-7c58-11eb-8b90-f777a56fe90f","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","number":"+821021656521","call_flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"update name","detail":"update detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().UpdateFlowID(gomock.Any(), tt.id, tt.callFlowID, tt.messageFlowID).Return(tt.resultData, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}

func Test_processV1NumbersRenewPost(t *testing.T) {
	type test struct {
		name string

		days    int
		hours   int
		tmRenew string

		request  *sock.Request
		response *sock.Response

		responseNumbers []*number.Number
	}

	tests := []test{
		{
			name: "normal",

			days:    3,
			hours:   10,
			tmRenew: "2023-06-26 18:26:49.000",

			request: &sock.Request{
				URI:      "/v1/numbers/renew",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"days":3,"hours":10,"tm_renew":"2023-06-26 18:26:49.000"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2725d60a-150f-11ee-bbb7-5394c1333278","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_name":"","provider_reference_id":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"2775644a-150f-11ee-b28b-5b82bb21aea0","customer_id":"00000000-0000-0000-0000-000000000000","number":"","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","provider_name":"","provider_reference_id":"","status":"","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_renew":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},

			responseNumbers: []*number.Number{
				{
					ID: uuid.FromStringOrNil("2725d60a-150f-11ee-bbb7-5394c1333278"),
				},
				{
					ID: uuid.FromStringOrNil("2775644a-150f-11ee-b28b-5b82bb21aea0"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().RenewNumbers(gomock.Any(), tt.days, tt.hours, tt.tmRenew).Return(tt.responseNumbers, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}
