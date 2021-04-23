package requesthandler

import (
	"fmt"
	"net/url"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

func TestNMNumberCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:      "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		userID  uint64
		numbers string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *nmnumber.Number
	}

	tests := []test{
		{
			"normal",

			1,
			"+821021656521",

			"bin-manager.number-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/numbers",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"user_id":1,"number":"+821021656521"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3eda6a34-7b17-11eb-a2fa-8f4c0fd14c20","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("3eda6a34-7b17-11eb-a2fa-8f4c0fd14c20"),
				Number:              "+821021656521",
				UserID:              1,
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NMNumberCreate(tt.userID, tt.numbers)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func TestNMNumberGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:      "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		userID    uint64
		pageToken string
		pageSize  uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult []nmnumber.Number
	}

	tests := []test{
		{
			"normal",

			1,
			"2021-03-02 03:23:20.995000",
			10,

			"bin-manager.number-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/numbers?page_token=%s&page_size=10&user_id=1", url.QueryEscape("2021-03-02 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0e00bb78-7b19-11eb-a238-9f1154b2c92e","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
			[]nmnumber.Number{
				{
					ID:                  uuid.FromStringOrNil("0e00bb78-7b19-11eb-a238-9f1154b2c92e"),
					Number:              "+821021656521",
					UserID:              1,
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NMNumberGets(tt.userID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func TestNMNumberGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:      "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		numberID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *nmnumber.Number
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("74a2f4bc-7be2-11eb-bb71-c767ac6ed931"),

			"bin-manager.number-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/numbers/74a2f4bc-7be2-11eb-bb71-c767ac6ed931",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"74a2f4bc-7be2-11eb-bb71-c767ac6ed931","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("74a2f4bc-7be2-11eb-bb71-c767ac6ed931"),
				Number:              "+821021656521",
				UserID:              1,
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NMNumberGet(tt.numberID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func TestNMNumberDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:      "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		numberID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *nmnumber.Number
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("aa0b1c7e-7be2-11eb-89f2-a7882f79d5b5"),

			"bin-manager.number-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/numbers/aa0b1c7e-7be2-11eb-89f2-a7882f79d5b5",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"aa0b1c7e-7be2-11eb-89f2-a7882f79d5b5","number":"+821021656521","flow_id":"00000000-0000-0000-0000-000000000000","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"deleted","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("aa0b1c7e-7be2-11eb-89f2-a7882f79d5b5"),
				Number:              "+821021656521",
				UserID:              1,
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NMNumberDelete(tt.numberID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func TestNMNumberUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:      "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		number *nmnumber.Number

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *nmnumber.Number
	}

	tests := []test{
		{
			"normal",

			&nmnumber.Number{
				ID:     uuid.FromStringOrNil("d3877fec-7c5b-11eb-bb46-07fe08c74815"),
				FlowID: uuid.FromStringOrNil("d45aae76-7c5b-11eb-9542-eb46d11b1c1a"),
			},

			"bin-manager.number-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/numbers/d3877fec-7c5b-11eb-bb46-07fe08c74815",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"flow_id":"d45aae76-7c5b-11eb-9542-eb46d11b1c1a"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d3877fec-7c5b-11eb-bb46-07fe08c74815","number":"+821021656521","flow_id":"d45aae76-7c5b-11eb-9542-eb46d11b1c1a","user_id":1,"provider_name":"telnyx","provider_reference_id":"","status":"active","t38_enabled":false,"emergency_enabled":false,"tm_purchase":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
			&nmnumber.Number{
				ID:                  uuid.FromStringOrNil("d3877fec-7c5b-11eb-bb46-07fe08c74815"),
				FlowID:              uuid.FromStringOrNil("d45aae76-7c5b-11eb-9542-eb46d11b1c1a"),
				Number:              "+821021656521",
				UserID:              1,
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NMNumberUpdate(tt.number)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
