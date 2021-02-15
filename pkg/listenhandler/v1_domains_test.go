package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/domainhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/requesthandler"
)

func TestProcessV1DomainsPost(t *testing.T) {
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

	type test struct {
		name      string
		reqDomain *models.Domain
		resDomain *models.Domain
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			&models.Domain{
				UserID:     1,
				DomainName: "0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("1744ccb4-6e13-11eb-b08d-bb42431b2fb3"),
				UserID:     1,
				DomainName: "0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net",
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/domains",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"user_id": 1, "domain_name": "0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1744ccb4-6e13-11eb-b08d-bb42431b2fb3","user_id":1,"name":"","detail":"","domain_name":"0229f50c-6e13-11eb-90cd-e7faf83c6884.sip.voipbin.net","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDomain.EXPECT().DomainCreate(gomock.Any(), tt.reqDomain).Return(tt.resDomain, nil)
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

func TestV1DomainsGet(t *testing.T) {
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

	type test struct {
		name      string
		userID    uint64
		pageToken string
		pageSize  uint64
		request   *rabbitmqhandler.Request
		domains   []*models.Domain

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			2,
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/domains?page_token=2020-10-10T03:30:17.000000&page_size=10&user_id=2",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*models.Domain{
				{
					ID:         uuid.FromStringOrNil("abd3467a-6ee6-11eb-824f-c386fbaad128"),
					UserID:     2,
					DomainName: "abd3467a-6ee6-11eb-824f-c386fbaad128.sip.voipbin.net",
				},
				{
					ID:         uuid.FromStringOrNil("af6488da-6ee6-11eb-8d4d-0f848f8e1aee"),
					UserID:     2,
					DomainName: "af6488da-6ee6-11eb-8d4d-0f848f8e1aee.sip.voipbin.net",
				}},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"abd3467a-6ee6-11eb-824f-c386fbaad128","user_id":2,"name":"","detail":"","domain_name":"abd3467a-6ee6-11eb-824f-c386fbaad128.sip.voipbin.net","tm_create":"","tm_update":"","tm_delete":""},{"id":"af6488da-6ee6-11eb-8d4d-0f848f8e1aee","user_id":2,"name":"","detail":"","domain_name":"af6488da-6ee6-11eb-8d4d-0f848f8e1aee.sip.voipbin.net","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"empty",
			3,
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/domains?page_token=2020-10-10T03:30:17.000000&page_size=10&user_id=3",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*models.Domain{},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDomain.EXPECT().DomainGetsByUserID(gomock.Any(), tt.userID, tt.pageToken, tt.pageSize).Return(tt.domains, nil)

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

func TestProcessV1DomainsPut(t *testing.T) {
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

	type test struct {
		name      string
		reqDomain *models.Domain
		resDomain *models.Domain
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			&models.Domain{
				ID:     uuid.FromStringOrNil("f4f3c3f4-6eee-11eb-8463-cf5490689c2e"),
				Name:   "update name",
				Detail: "update detail",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("f4f3c3f4-6eee-11eb-8463-cf5490689c2e"),
				UserID:     1,
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
				Data:       []byte(`{"id":"f4f3c3f4-6eee-11eb-8463-cf5490689c2e","user_id":1,"name":"update name","detail":"update detail","domain_name":"f4f3c3f4-6eee-11eb-8463-cf5490689c2e.sip.voipbin.net","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDomain.EXPECT().DomainUpdate(gomock.Any(), tt.reqDomain).Return(tt.resDomain, nil)
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

func TestProcessV1DomainsDelete(t *testing.T) {
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

	type test struct {
		name      string
		domainID  uuid.UUID
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			uuid.FromStringOrNil("09e94cb4-6f32-11eb-af29-27dcd65a7064"),
			&rabbitmqhandler.Request{
				URI:    "/v1/domains/09e94cb4-6f32-11eb-af29-27dcd65a7064",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDomain.EXPECT().DomainDelete(gomock.Any(), tt.domainID).Return(nil)
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
