package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
)

func TestProcessV1NumbersPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		db:            mockDB,
		reqHandler:    mockReq,
		numberHandler: mockNumber,
	}

	type test struct {
		name          string
		userID        uint64
		number        string
		createdNumber *number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"1 number",
			1,
			"+821021656521",
			&number.Number{

				ID:                  uuid.FromStringOrNil("3a379dce-792a-11eb-a8e1-9f51cab620f8"),
				Number:              "+821021656521",
				UserID:              1,
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
				Data:     []byte(`{"user_id": 1, "number": "+821021656521"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3a379dce-792a-11eb-a8e1-9f51cab620f8","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().CreateNumber(tt.userID, tt.number).Return(tt.createdNumber, nil)
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
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		db:            mockDB,
		reqHandler:    mockReq,
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
				UserID:              1,
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
				Data:       []byte(`{"id":"9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"deleted","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
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
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		db:            mockDB,
		reqHandler:    mockReq,
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
				UserID:              1,
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
				Data:       []byte(`{"id":"7b6f4caa-7a48-11eb-8b06-ff14cc60c8ad","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
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
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		db:            mockDB,
		reqHandler:    mockReq,
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
			"1 number",
			"+821021656521",
			&number.Number{
				ID:                  uuid.FromStringOrNil("52f48d94-7a57-11eb-bda1-57eb6d071e62"),
				Number:              "+821021656521",
				UserID:              1,
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
				Data:       []byte(`{"id":"52f48d94-7a57-11eb-bda1-57eb6d071e62","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
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
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		db:            mockDB,
		reqHandler:    mockReq,
		numberHandler: mockNumber,
	}

	type test struct {
		name       string
		userID     uint64
		pageSize   uint64
		pageToken  string
		resultData []*number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"1 number",
			1,
			10,
			"2021-03-01 03:30:17.000000",
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("eeafd418-7a4e-11eb-8750-9bb0ca1d7926"),
					Number:              "+821021656521",
					UserID:              1,
					ProviderName:        number.ProviderNameTelnyx,
					ProviderReferenceID: "",
					Status:              number.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
				},
			},
			&rabbitmqhandler.Request{
				URI:    "/v1/numbers?user_id=1&page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"eeafd418-7a4e-11eb-8750-9bb0ca1d7926","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().GetNumbers(gomock.Any(), tt.userID, tt.pageSize, tt.pageToken).Return(tt.resultData, nil)
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
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		db:            mockDB,
		reqHandler:    mockReq,
		numberHandler: mockNumber,
	}

	type test struct {
		name       string
		updateInfo *number.Number
		resultData *number.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&number.Number{
				ID:     uuid.FromStringOrNil("935190b4-7c58-11eb-8b90-f777a56fe90f"),
				FlowID: uuid.FromStringOrNil("9394929c-7c58-11eb-8af3-13d1657955b6"),
			},
			&number.Number{
				ID:                  uuid.FromStringOrNil("935190b4-7c58-11eb-8b90-f777a56fe90f"),
				FlowID:              uuid.FromStringOrNil("9394929c-7c58-11eb-8af3-13d1657955b6"),
				Number:              "+821021656521",
				UserID:              1,
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
				Data:     []byte(`{"flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"935190b4-7c58-11eb-8b90-f777a56fe90f","number":"+821021656521","flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().UpdateNumber(gomock.Any(), tt.updateInfo).Return(tt.resultData, nil)
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
