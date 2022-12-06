package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/domainhandler"
)

func Test_processV1DomainsPost(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		domainName string
		domainN    string
		detail     string

		resDomain *domain.Domain
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",

			uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
			"0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net",
			"test name",
			"test detail",

			&domain.Domain{
				ID:         uuid.FromStringOrNil("1744ccb4-6e13-11eb-b08d-bb42431b2fb3"),
				CustomerID: uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
				DomainName: "0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net",
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/domains",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "8c1f0206-7fed-11ec-bc4d-b75bc59a142c", "name": "test name", "detail": "test detail", "domain_name": "0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1744ccb4-6e13-11eb-b08d-bb42431b2fb3","customer_id":"8c1f0206-7fed-11ec-bc4d-b75bc59a142c","name":"","detail":"","domain_name":"0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDomain := domainhandler.NewMockDomainHandler(mc)

			h := &listenHandler{
				rabbitSock:    mockSock,
				reqHandler:    mockReq,
				domainHandler: mockDomain,
			}

			mockDomain.EXPECT().Create(gomock.Any(), tt.customerID, tt.domainName, tt.domainN, tt.detail).Return(tt.resDomain, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1DomainsGet(t *testing.T) {

	type test struct {
		name       string
		customerID uuid.UUID
		pageToken  string
		pageSize   uint64
		request    *rabbitmqhandler.Request
		domains    []*domain.Domain

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/domains?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=8c1f0206-7fed-11ec-bc4d-b75bc59a142c",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*domain.Domain{
				{
					ID:         uuid.FromStringOrNil("abd3467a-6ee6-11eb-824f-c386fbaad128"),
					CustomerID: uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
					DomainName: "abd3467a-6ee6-11eb-824f-c386fbaad128.sip.voipbin.net",
				},
				{
					ID:         uuid.FromStringOrNil("af6488da-6ee6-11eb-8d4d-0f848f8e1aee"),
					CustomerID: uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
					DomainName: "af6488da-6ee6-11eb-8d4d-0f848f8e1aee.sip.voipbin.net",
				}},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"abd3467a-6ee6-11eb-824f-c386fbaad128","customer_id":"8c1f0206-7fed-11ec-bc4d-b75bc59a142c","name":"","detail":"","domain_name":"abd3467a-6ee6-11eb-824f-c386fbaad128.sip.voipbin.net","tm_create":"","tm_update":"","tm_delete":""},{"id":"af6488da-6ee6-11eb-8d4d-0f848f8e1aee","customer_id":"8c1f0206-7fed-11ec-bc4d-b75bc59a142c","name":"","detail":"","domain_name":"af6488da-6ee6-11eb-8d4d-0f848f8e1aee.sip.voipbin.net","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"empty",
			uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/domains?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=8c1f0206-7fed-11ec-bc4d-b75bc59a142c",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*domain.Domain{},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDomain := domainhandler.NewMockDomainHandler(mc)

			h := &listenHandler{
				rabbitSock:    mockSock,
				reqHandler:    mockReq,
				domainHandler: mockDomain,
			}

			mockDomain.EXPECT().Gets(gomock.Any(), tt.customerID, tt.pageToken, tt.pageSize).Return(tt.domains, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1DomainsPut(t *testing.T) {

	type test struct {
		name      string
		reqDomain *domain.Domain

		id      uuid.UUID
		domainN string
		detail  string

		resDomain *domain.Domain
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			&domain.Domain{
				ID:     uuid.FromStringOrNil("f4f3c3f4-6eee-11eb-8463-cf5490689c2e"),
				Name:   "update name",
				Detail: "update detail",
			},

			uuid.FromStringOrNil("f4f3c3f4-6eee-11eb-8463-cf5490689c2e"),
			"update name",
			"update detail",

			&domain.Domain{
				ID:         uuid.FromStringOrNil("f4f3c3f4-6eee-11eb-8463-cf5490689c2e"),
				CustomerID: uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
				Name:       "update name",
				Detail:     "update detail",
				DomainName: "f4f3c3f4-6eee-11eb-8463-cf5490689c2e.sip.voipbin.net",
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/domains/f4f3c3f4-6eee-11eb-8463-cf5490689c2e",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name", "detail":"update detail"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f4f3c3f4-6eee-11eb-8463-cf5490689c2e","customer_id":"8c1f0206-7fed-11ec-bc4d-b75bc59a142c","name":"update name","detail":"update detail","domain_name":"f4f3c3f4-6eee-11eb-8463-cf5490689c2e.sip.voipbin.net","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDomain := domainhandler.NewMockDomainHandler(mc)

			h := &listenHandler{
				rabbitSock:    mockSock,
				reqHandler:    mockReq,
				domainHandler: mockDomain,
			}
			mockDomain.EXPECT().Update(gomock.Any(), tt.id, tt.domainN, tt.detail).Return(tt.resDomain, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1DomainsDelete(t *testing.T) {

	type test struct {
		name     string
		domainID uuid.UUID

		request        *rabbitmqhandler.Request
		responseDomain *domain.Domain

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("09e94cb4-6f32-11eb-af29-27dcd65a7064"),
			&rabbitmqhandler.Request{
				URI:    "/v1/domains/09e94cb4-6f32-11eb-af29-27dcd65a7064",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&domain.Domain{
				ID: uuid.FromStringOrNil("09e94cb4-6f32-11eb-af29-27dcd65a7064"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"09e94cb4-6f32-11eb-af29-27dcd65a7064","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","domain_name":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDomain := domainhandler.NewMockDomainHandler(mc)

			h := &listenHandler{
				rabbitSock:    mockSock,
				reqHandler:    mockReq,
				domainHandler: mockDomain,
			}

			mockDomain.EXPECT().Delete(gomock.Any(), tt.domainID).Return(tt.responseDomain, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
