package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestRMV1DomainCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		customerID      uuid.UUID
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

			uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
			"test.sip.voipbin.net",
			"test name",
			"test detail",

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/domains",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_name":"test.sip.voipbin.net","name":"test name","detail":"test detail"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"68040bf2-6ed5-11eb-9924-9febe8425cbe","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_name":"test.sip.voipbin.net","name":"test name","detail":"test detail","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("68040bf2-6ed5-11eb-9924-9febe8425cbe"),
				CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
				DomainName: "test.sip.voipbin.net",
				Name:       "test name",
				Detail:     "test detail",
				TMCreate:   "2020-09-20 03:23:20.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMV1DomainCreate(ctx, tt.customerID, tt.domainName, tt.domainTmpName, tt.domainTmpDetail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestRMV1DomainUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		id      uuid.UUID
		domainN string
		detail  string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *rmdomain.Domain
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("f4063a6c-6ed5-11eb-8835-23f57d9e419c"),
			"update name",
			"update detail",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f4063a6c-6ed5-11eb-8835-23f57d9e419c","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_name":"test.sip.voipbin.net","name":"update name","detail":"update detail","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
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
				CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
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
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMV1DomainUpdate(ctx, tt.id, tt.domainN, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestRMV1DomainGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
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
				Data:       []byte(`{"id":"eb0e485e-6ed6-11eb-81bd-9365803e5d9f","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_name":"test.sip.voipbin.net","name":"test domain","detail":"test domain detail","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/domains/eb0e485e-6ed6-11eb-81bd-9365803e5d9f",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rmdomain.Domain{
				ID:         uuid.FromStringOrNil("eb0e485e-6ed6-11eb-81bd-9365803e5d9f"),
				CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
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
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMV1DomainGet(ctx, tt.domainID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func TestRMV1DomainDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		domainID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *rmdomain.Domain
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("5980b2e4-6ed8-11eb-abc3-33f6180819c6"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5980b2e4-6ed8-11eb-abc3-33f6180819c6"}`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/domains/5980b2e4-6ed8-11eb-abc3-33f6180819c6",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},
			&rmdomain.Domain{
				ID: uuid.FromStringOrNil("5980b2e4-6ed8-11eb-abc3-33f6180819c6"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMV1DomainDelete(ctx, tt.domainID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestRMV1DomainsGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	type test struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []rmdomain.Domain
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
			"2020-09-20 03:23:20.995000",
			10,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d19c3956-6ed8-11eb-b971-fb12bc338aeb","customer_id":"324cf776-7ff0-11ec-a0ea-e30825a4224f","domain_name":"test.sip.voipbin.net","name":"test","detail":"test detail","tm_create":"2020-09-20 03:23:20.995000","tm_update":"","tm_delete":""}]`),
			},

			"bin-manager.registrar-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/domains?page_token=%s&page_size=10&customer_id=324cf776-7ff0-11ec-a0ea-e30825a4224f", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]rmdomain.Domain{
				{
					ID:         uuid.FromStringOrNil("d19c3956-6ed8-11eb-b971-fb12bc338aeb"),
					CustomerID: uuid.FromStringOrNil("324cf776-7ff0-11ec-a0ea-e30825a4224f"),
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
			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RMV1DomainGets(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
