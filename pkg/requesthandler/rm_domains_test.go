package requesthandler

import (
	"fmt"
	"net/url"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
)

func TestRMDomainCreate(t *testing.T) {
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

		userID          uint64
		domainName      string
		domainTmpName   string
		domainTmpDetail string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *rmdomain.Domain
	}

	tests := []test{
		{
			"normal",

			1,
			"test.sip.voipbin.net",
			"test name",
			"test detail",

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/domains",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"user_id":1,"domain_name":"test.sip.voipbin.net","name":"test name","detail":"test detail"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"68040bf2-6ed5-11eb-9924-9febe8425cbe","user_id":1,"domain_name":"test.sip.voipbin.net","name":"test name","detail":"test detail","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("68040bf2-6ed5-11eb-9924-9febe8425cbe"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "test name",
				Detail:     "test detail",
				TMCreate:   "2020-09-20 03:23:20.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMDomainCreate(tt.userID, tt.domainName, tt.domainTmpName, tt.domainTmpDetail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestRMDomainUpdate(t *testing.T) {
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

		requestDomain *rmdomain.Domain
		response      *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *rmdomain.Domain
	}

	tests := []test{
		{
			"normal",
			&rmdomain.Domain{
				ID:     uuid.FromStringOrNil("f4063a6c-6ed5-11eb-8835-23f57d9e419c"),
				Name:   "update name",
				Detail: "update detail",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f4063a6c-6ed5-11eb-8835-23f57d9e419c","user_id":1,"domain_name":"test.sip.voipbin.net","name":"update name","detail":"update detail","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/domains/f4063a6c-6ed5-11eb-8835-23f57d9e419c",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail"}`),
			},
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("f4063a6c-6ed5-11eb-8835-23f57d9e419c"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "update name",
				Detail:     "update detail",
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "",
				TMDelete:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMDomainUpdate(tt.requestDomain)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestRMDomainGet(t *testing.T) {
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

		domainID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *rmdomain.Domain
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("eb0e485e-6ed6-11eb-81bd-9365803e5d9f"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"eb0e485e-6ed6-11eb-81bd-9365803e5d9f","user_id":1,"domain_name":"test.sip.voipbin.net","name":"test domain","detail":"test domain detail","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/domains/eb0e485e-6ed6-11eb-81bd-9365803e5d9f",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("eb0e485e-6ed6-11eb-81bd-9365803e5d9f"),
				UserID:     1,
				DomainName: "test.sip.voipbin.net",
				Name:       "test domain",
				Detail:     "test domain detail",
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "",
				TMDelete:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMDomainGet(tt.domainID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestRMDomainDelete(t *testing.T) {
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

		domainID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("5980b2e4-6ed8-11eb-abc3-33f6180819c6"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/domains/5980b2e4-6ed8-11eb-abc3-33f6180819c6",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.RMDomainDelete(tt.domainID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestRMDomainsGets(t *testing.T) {
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

		userID    uint64
		pageToken string
		pageSize  uint64

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []rmdomain.Domain
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
				Data:       []byte(`[{"id":"d19c3956-6ed8-11eb-b971-fb12bc338aeb","user_id":1,"domain_name":"test.sip.voipbin.net","name":"test","detail":"test detail","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}]`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/domains?page_token=%s&page_size=10&user_id=1", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]rmdomain.Domain{
				{
					ID:         uuid.FromStringOrNil("d19c3956-6ed8-11eb-b971-fb12bc338aeb"),
					UserID:     1,
					DomainName: "test.sip.voipbin.net",
					Name:       "test",
					Detail:     "test detail",
					TMCreate:   "2020-09-20 03:23:20.995000",
					TMUpdate:   "",
					TMDelete:   "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMDomainGets(tt.userID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
