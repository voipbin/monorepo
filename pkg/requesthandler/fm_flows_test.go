package requesthandler

import (
	"fmt"
	"net/url"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
)

func TestFMFlowCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueStorage:   "bin-manager.storage-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
		queueNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		flow *fmflow.Flow

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectResult  *fmflow.Flow
	}

	tests := []test{
		{
			"normal",

			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0"),
				UserID:     1,
				Name:       "test flow",
				Detail:     "test flow detail",
				Actions:    []fmaction.Action{},
				Persist:    true,
				WebhookURI: "",
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"persist":true,"webhook_uri":""}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"persist":true,"webhook_uri":"","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			&fmflow.Flow{
				ID:       uuid.FromStringOrNil("5d205ffa-f2ee-11ea-9ae3-cf94fb96c9f0"),
				UserID:   1,
				Name:     "test flow",
				Detail:   "test flow detail",
				Actions:  []fmaction.Action{},
				Persist:  true,
				TMCreate: "2020-09-20 03:23:20.995000",
				TMUpdate: "",
				TMDelete: "",
			},
		},
		{
			"webhook",

			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("c409a736-82f3-11eb-839a-ebe51df950d4"),
				UserID:     1,
				Name:       "test flow",
				Detail:     "test flow detail",
				Actions:    []fmaction.Action{},
				Persist:    true,
				WebhookURI: "https://test.com/webhook",
			},
			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"c409a736-82f3-11eb-839a-ebe51df950d4","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"persist":true,"webhook_uri":"https://test.com/webhook"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c409a736-82f3-11eb-839a-ebe51df950d4","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"persist":true,"webhook_uri":"https://test.com/webhook","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			&fmflow.Flow{
				ID:         uuid.FromStringOrNil("c409a736-82f3-11eb-839a-ebe51df950d4"),
				UserID:     1,
				Name:       "test flow",
				Detail:     "test flow detail",
				Actions:    []fmaction.Action{},
				Persist:    true,
				WebhookURI: "https://test.com/webhook",
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "",
				TMDelete:   "",
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			// res, err := reqHandler.FMFlowCreate(tt.userID, tt.flowID, tt.flowName, tt.flowDetail, tt.actions, tt.persist)
			res, err := reqHandler.FMFlowCreate(tt.flow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestFMFlowUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueStorage:   "bin-manager.storage-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
		queueNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		requestFlow *fmflow.Flow
		response    *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *fmflow.Flow
	}

	tests := []test{
		{
			"empty action",
			&fmflow.Flow{
				ID:      uuid.FromStringOrNil("7dc3a1b2-6789-11eb-9f30-1b1cc6d13e51"),
				Name:    "update name",
				Detail:  "update detail",
				Actions: []fmaction.Action{},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7dc3a1b2-6789-11eb-9f30-1b1cc6d13e51","user_id":1,"name":"update name","detail":"update detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/7dc3a1b2-6789-11eb-9f30-1b1cc6d13e51",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail","actions":[]}`),
			},
			&fmflow.Flow{
				ID:       uuid.FromStringOrNil("7dc3a1b2-6789-11eb-9f30-1b1cc6d13e51"),
				UserID:   1,
				Name:     "update name",
				Detail:   "update detail",
				Actions:  []fmaction.Action{},
				TMCreate: "2020-09-20 03:23:20.995000",
				TMUpdate: "",
				TMDelete: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FMFlowUpdate(tt.requestFlow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestFMFlowGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueStorage:   "bin-manager.storage-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
		queueNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		userID uint64
		flowID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *fmflow.Flow
	}

	tests := []test{
		{
			"normal",

			1,
			uuid.FromStringOrNil("be66d9a6-6ed6-11eb-8152-0bb66bad7293"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"be66d9a6-6ed6-11eb-8152-0bb66bad7293","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/be66d9a6-6ed6-11eb-8152-0bb66bad7293",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&fmflow.Flow{
				ID:       uuid.FromStringOrNil("be66d9a6-6ed6-11eb-8152-0bb66bad7293"),
				UserID:   1,
				Name:     "test flow",
				Detail:   "test flow detail",
				Actions:  []fmaction.Action{},
				TMCreate: "2020-09-20 03:23:20.995000",
				TMUpdate: "",
				TMDelete: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FMFlowGet(tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestFMFlowDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueStorage:   "bin-manager.storage-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
		queueNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		flowID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("4193c3a2-67ca-11eb-a892-0b6d18cda91a"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/flows/4193c3a2-67ca-11eb-a892-0b6d18cda91a",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.FMFlowDelete(tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestFMFlowGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueStorage:   "bin-manager.storage-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
		queueNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		userID    uint64
		pageToken string
		pageSize  uint64

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []fmflow.Flow
	}

	tests := []test{
		{
			"normal",

			1,
			"2020-09-20 03:23:20.995000",
			10,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"158e4b2c-0c55-11eb-b4f2-37c93a78a6a0","user_id":1,"name":"test flow","detail":"test flow detail","actions":[],"tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}]`),
			},

			"bin-manager.flow-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/flows?page_token=%s&page_size=10&user_id=1", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]fmflow.Flow{
				{
					ID:       uuid.FromStringOrNil("158e4b2c-0c55-11eb-b4f2-37c93a78a6a0"),
					UserID:   1,
					Name:     "test flow",
					Detail:   "test flow detail",
					Actions:  []fmaction.Action{},
					TMCreate: "2020-09-20 03:23:20.995000",
					TMUpdate: "",
					TMDelete: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FMFlowGets(tt.userID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
