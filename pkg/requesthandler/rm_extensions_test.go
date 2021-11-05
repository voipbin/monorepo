package requesthandler

import (
	"fmt"
	"net/url"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
)

func TestRMExtensionCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:                  mockSock,
		exchangeDelay:         "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:     "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		extension *rmextension.Extension

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *rmextension.Extension
	}

	tests := []test{
		{
			"normal",

			&rmextension.Extension{
				UserID:    1,
				Name:      "test name",
				Detail:    "test detail",
				DomainID:  uuid.FromStringOrNil("22de2e58-6f9e-11eb-8fee-ef16005005d7"),
				Extension: "4c98b74a-6f9e-11eb-a82f-37575ab16881",
				Password:  "53710356-6f9e-11eb-8a91-43345d98682a",
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"user_id":1,"domain_id":"22de2e58-6f9e-11eb-8fee-ef16005005d7","extension":"4c98b74a-6f9e-11eb-a82f-37575ab16881","password":"53710356-6f9e-11eb-8a91-43345d98682a","name":"test name","detail":"test detail"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"68040bf2-6ed5-11eb-9924-9febe8425cbe","user_id":1,"domain_id":"22de2e58-6f9e-11eb-8fee-ef16005005d7","name":"test name","detail":"test detail","extension":"4c98b74a-6f9e-11eb-a82f-37575ab16881","password":"53710356-6f9e-11eb-8a91-43345d98682a","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			&rmextension.Extension{
				ID:        uuid.FromStringOrNil("68040bf2-6ed5-11eb-9924-9febe8425cbe"),
				UserID:    1,
				Name:      "test name",
				Detail:    "test detail",
				DomainID:  uuid.FromStringOrNil("22de2e58-6f9e-11eb-8fee-ef16005005d7"),
				Extension: "4c98b74a-6f9e-11eb-a82f-37575ab16881",
				Password:  "53710356-6f9e-11eb-8a91-43345d98682a",
				TMCreate:  "2020-09-20 03:23:20.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMExtensionCreate(tt.extension)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestRMExtensionUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:                  mockSock,
		exchangeDelay:         "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:     "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		requestExtension *rmextension.Extension
		response         *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *rmextension.Extension
	}

	tests := []test{
		{
			"normal",
			&rmextension.Extension{
				ID:       uuid.FromStringOrNil("0be5298a-6f9f-11eb-bb77-f71f5b5f95f7"),
				Name:     "update name",
				Detail:   "update detail",
				Password: "update password",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0be5298a-6f9f-11eb-bb77-f71f5b5f95f7","user_id":1,"name":"update name","detail":"update detail","password":"update password"}`),
			},
			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions/0be5298a-6f9f-11eb-bb77-f71f5b5f95f7",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail","password":"update password"}`),
			},
			&rmextension.Extension{
				ID:       uuid.FromStringOrNil("0be5298a-6f9f-11eb-bb77-f71f5b5f95f7"),
				UserID:   1,
				Name:     "update name",
				Detail:   "update detail",
				Password: "update password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMExtensionUpdate(tt.requestExtension)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestRMExtensionGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:                  mockSock,
		exchangeDelay:         "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:     "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		extensionID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *rmextension.Extension
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("342f9734-6fa1-11eb-a937-17d537105d6a"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"342f9734-6fa1-11eb-a937-17d537105d6a","user_id":1,"domain_id":"4351e596-6fa1-11eb-b086-db7f03792b30","name":"test domain","detail":"test domain detail","extension":"test","password":"password","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions/342f9734-6fa1-11eb-a937-17d537105d6a",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rmextension.Extension{
				ID:        uuid.FromStringOrNil("342f9734-6fa1-11eb-a937-17d537105d6a"),
				UserID:    1,
				DomainID:  uuid.FromStringOrNil("4351e596-6fa1-11eb-b086-db7f03792b30"),
				Extension: "test",
				Password:  "password",
				Name:      "test domain",
				Detail:    "test domain detail",
				TMCreate:  "2020-09-20 03:23:20.995000",
				TMUpdate:  "",
				TMDelete:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMExtensionGet(tt.extensionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestRMExtensionDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:                  mockSock,
		exchangeDelay:         "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:     "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		extesnionID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("b2ca6024-6fa1-11eb-aa5a-738c234d2ee1"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions/b2ca6024-6fa1-11eb-aa5a-738c234d2ee1",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.RMExtensionDelete(tt.extesnionID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestRMExtensionsGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:                  mockSock,
		exchangeDelay:         "bin-manager.delay",
		queueRequestCall:      "bin-manager.call-manager.request",
		queueRequesstFlow:     "bin-manager.flow-manager.request",
		queueRequestStorage:   "bin-manager.storage-manager.request",
		queueRequestRegistrar: "bin-manager.registrar-manager.request",
		queueRequestNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		domainID  uuid.UUID
		pageToken string
		pageSize  uint64

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []rmextension.Extension
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("e45dafce-6fa1-11eb-9e87-7ba8b7ae10f0"),
			"2020-09-20 03:23:20.995000",
			10,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d19c3956-6ed8-11eb-b971-fb12bc338aeb","user_id":1,"domain_id":"e45dafce-6fa1-11eb-9e87-7ba8b7ae10f0","name":"test","detail":"test detail","extension":"test","password":"password","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}]`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/extensions?page_token=%s&page_size=10&domain_id=e45dafce-6fa1-11eb-9e87-7ba8b7ae10f0", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]rmextension.Extension{
				{
					ID:        uuid.FromStringOrNil("d19c3956-6ed8-11eb-b971-fb12bc338aeb"),
					UserID:    1,
					DomainID:  uuid.FromStringOrNil("e45dafce-6fa1-11eb-9e87-7ba8b7ae10f0"),
					Name:      "test",
					Detail:    "test detail",
					Extension: "test",
					Password:  "password",
					TMCreate:  "2020-09-20 03:23:20.995000",
					TMUpdate:  "",
					TMDelete:  "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMExtensionGets(tt.domainID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
