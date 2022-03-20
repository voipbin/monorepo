package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandler"
)

func TestProcessV1NumbersPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		numberHandler: mockNumber,
	}

	type test struct {
		name string

		customerID    uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		num           string
		numberName    string
		detail        string

		createdNumber *number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
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
			&rabbitmqhandler.Request{
				URI:      "/v1/numbers",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "72f3b054-7ff4-11ec-9af9-0b8c5dbee258", "call_flow_id": "7051e796-8821-11ec-9b7d-d322b1036e7d", "message_flow_id": "c5f1dffc-a866-11ec-be0a-a3c412cba4dc", "number": "+821021656521", "name": "test name", "detail": "test detail"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3a379dce-792a-11eb-a8e1-9f51cab620f8","number":"+821021656521","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","call_flow_id":"7051e796-8821-11ec-9b7d-d322b1036e7d","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().CreateNumber(gomock.Any(), tt.customerID, tt.num, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail).Return(tt.createdNumber, nil)
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

func TestProcessV1NumbersIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		numberHandler: mockNumber,
	}

	type test struct {
		name       string
		id         uuid.UUID
		resultData *number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
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
			&rabbitmqhandler.Request{
				URI:    "/v1/numbers/9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f","number":"+821021656521","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"deleted","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().ReleaseNumber(gomock.Any(), tt.id).Return(tt.resultData, nil)
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

func TestProcessV1NumbersIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		numberHandler: mockNumber,
	}

	type test struct {
		name       string
		id         uuid.UUID
		resultData *number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
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
			&rabbitmqhandler.Request{
				URI:    "/v1/numbers/7b6f4caa-7a48-11eb-8b06-ff14cc60c8ad",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7b6f4caa-7a48-11eb-8b06-ff14cc60c8ad","number":"+821021656521","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().GetNumber(gomock.Any(), tt.id).Return(tt.resultData, nil)
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

func TestProcessV1NumbersNumberGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		numberHandler: mockNumber,
	}

	type test struct {
		name       string
		num        string
		resultData *number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			"+821021656521",
			&number.Number{
				ID:                  uuid.FromStringOrNil("52f48d94-7a57-11eb-bda1-57eb6d071e62"),
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
			&rabbitmqhandler.Request{
				URI:    "/v1/numbers/%2B821021656521",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"52f48d94-7a57-11eb-bda1-57eb6d071e62","number":"+821021656521","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().GetNumberByNumber(gomock.Any(), tt.num).Return(tt.resultData, nil)
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

func TestProcessV1NumbersGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		numberHandler: mockNumber,
	}

	type test struct {
		name string

		customerID uuid.UUID
		pageSize   uint64
		pageToken  string
		resultData []*number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
			10,
			"2021-03-01 03:30:17.000000",
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("eeafd418-7a4e-11eb-8750-9bb0ca1d7926"),
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
			},
			&rabbitmqhandler.Request{
				URI:    "/v1/numbers?customer_id=72f3b054-7ff4-11ec-9af9-0b8c5dbee258&page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"eeafd418-7a4e-11eb-8750-9bb0ca1d7926","number":"+821021656521","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 results",

			uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
			10,
			"2021-03-01 03:30:17.000000",
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("5c18ee62-8800-11ec-bb8b-b74be365ebf2"),
					Number:              "+821100000001",
					CustomerID:          uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
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
					CustomerID:          uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258"),
					Name:                "test name",
					Detail:              "test detail",
					ProviderName:        number.ProviderNameTelnyx,
					ProviderReferenceID: "",
					Status:              number.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
				},
			},
			&rabbitmqhandler.Request{
				URI:    "/v1/numbers?customer_id=72f3b054-7ff4-11ec-9af9-0b8c5dbee258&page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5c18ee62-8800-11ec-bb8b-b74be365ebf2","number":"+821100000001","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"69dc1916-8800-11ec-8e68-e74880ae3121","number":"+821100000002","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","call_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().GetNumbers(gomock.Any(), tt.customerID, tt.pageSize, tt.pageToken).Return(tt.resultData, nil)
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

func TestProcessV1NumbersIDPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		numberHandler: mockNumber,
	}

	type test struct {
		name string

		id         uuid.UUID
		numberName string
		detail     string

		resultData *number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("935190b4-7c58-11eb-8b90-f777a56fe90f"),
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
			&rabbitmqhandler.Request{
				URI:      "/v1/numbers/935190b4-7c58-11eb-8b90-f777a56fe90f",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6", "name": "update name", "detail": "update detail"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"935190b4-7c58-11eb-8b90-f777a56fe90f","number":"+821021656521","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","call_flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"update name","detail":"update detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().UpdateBasicInfo(gomock.Any(), tt.id, tt.numberName, tt.detail).Return(tt.resultData, nil)
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

func TestProcessV1NumbersIDFlowIDPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		numberHandler: mockNumber,
	}

	type test struct {
		name string

		id            uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID

		resultData *number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
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
			&rabbitmqhandler.Request{
				URI:      "/v1/numbers/935190b4-7c58-11eb-8b90-f777a56fe90f/flow_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"call_flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"935190b4-7c58-11eb-8b90-f777a56fe90f","number":"+821021656521","customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258","call_flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6","message_flow_id":"00000000-0000-0000-0000-000000000000","name":"update name","detail":"update detail","provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

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
