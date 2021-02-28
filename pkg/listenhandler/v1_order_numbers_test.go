package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
)

func TestProcessV1OrderNumbersPost(t *testing.T) {
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
		numbers       []string
		createdNumber []*models.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"1 number",
			1,
			[]string{"+821021656521"},
			[]*models.Number{
				{
					ID:                  uuid.FromStringOrNil("3a379dce-792a-11eb-a8e1-9f51cab620f8"),
					Number:              "+821021656521",
					UserID:              1,
					ProviderName:        models.NumberProviderNameTelnyx,
					ProviderReferenceID: "",
					Status:              models.NumberStatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
				},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/order_numbers",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "numbers": ["+821021656521"]}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"3a379dce-792a-11eb-a8e1-9f51cab620f8","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().CreateOrderNumbers(tt.userID, tt.numbers).Return(tt.createdNumber, nil)
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

func TestProcessV1OrderNumbersIDDelete(t *testing.T) {
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
		resultData *models.Number

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"1 number",
			uuid.FromStringOrNil("9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f"),
			&models.Number{
				ID:                  uuid.FromStringOrNil("9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        models.NumberProviderNameTelnyx,
				ProviderReferenceID: "",
				Status:              models.NumberStatusDeleted,
				T38Enabled:          false,
				EmergencyEnabled:    false,
			},
			&rabbitmqhandler.Request{
				URI:    "/v1/order_numbers/9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f",
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

			mockNumber.EXPECT().ReleaseOrderNumbers(gomock.Any(), tt.id).Return(tt.resultData, nil)
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
