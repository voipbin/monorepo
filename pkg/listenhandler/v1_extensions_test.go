package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/domainhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
)

func Test_processV1ExtensionsPost(t *testing.T) {

	type test struct {
		name string

		request *rabbitmqhandler.Request

		expectCustomerID uuid.UUID
		expectName       string
		expectDetail     string
		expectDomainID   uuid.UUID
		expectExtension  string
		expectPassword   string

		responseExtension *extension.Extension
		expectRes         *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			&rabbitmqhandler.Request{
				URI:      "/v1/extensions",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "2e341ffa-7fed-11ec-9667-1357b91d745d", "name": "test name", "detail": "test detail", "domain_id": "42dd6424-6ebf-11eb-8630-6b91b6089dc4", "extension": "45eb6bac-6ebf-11eb-bcf3-3b9157826d22", "password": "4b1f7a6e-6ebf-11eb-a47e-5351700cd612"}`),
			},

			uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
			"test name",
			"test detail",
			uuid.FromStringOrNil("42dd6424-6ebf-11eb-8630-6b91b6089dc4"),
			"45eb6bac-6ebf-11eb-bcf3-3b9157826d22",
			"4b1f7a6e-6ebf-11eb-a47e-5351700cd612",

			&extension.Extension{
				ID:         uuid.FromStringOrNil("3f4bc63e-6ebf-11eb-b7de-df47266bf559"),
				CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),

				DomainID: uuid.FromStringOrNil("42dd6424-6ebf-11eb-8630-6b91b6089dc4"),

				EndpointID: "45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net",
				AORID:      "45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net",
				AuthID:     "45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net",

				Extension: "45eb6bac-6ebf-11eb-bcf3-3b9157826d22",
				Password:  "4b1f7a6e-6ebf-11eb-a47e-5351700cd612",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3f4bc63e-6ebf-11eb-b7de-df47266bf559","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"","detail":"","domain_id":"42dd6424-6ebf-11eb-8630-6b91b6089dc4","endpoint_id":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net","aor_id":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net","auth_id":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net","extension":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22","password":"4b1f7a6e-6ebf-11eb-a47e-5351700cd612","tm_create":"","tm_update":"","tm_delete":""}`),
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
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)
			h := &listenHandler{
				rabbitSock:       mockSock,
				reqHandler:       mockReq,
				domainHandler:    mockDomain,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().Create(
				gomock.Any(),
				tt.expectCustomerID,
				tt.expectName,
				tt.expectDetail,
				tt.expectDomainID,
				tt.expectExtension,
				tt.expectPassword,
			).Return(tt.responseExtension, nil)
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

func Test_processV1ExtensionsGet_domain_id(t *testing.T) {

	type test struct {
		name       string
		customerID uuid.UUID
		domainID   uuid.UUID
		pageToken  string
		pageSize   uint64
		request    *rabbitmqhandler.Request
		exts       []*extension.Extension

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name:      "normal",
			domainID:  uuid.FromStringOrNil("a4b2db1e-6f4d-11eb-9df6-5793191d903c"),
			pageToken: "2020-10-10T03:30:17.000000",
			pageSize:  10,
			request: &rabbitmqhandler.Request{
				URI:      "/v1/extensions?page_token=2020-10-10T03:30:17.000000&page_size=10&domain_id=a4b2db1e-6f4d-11eb-9df6-5793191d903c",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			exts: []*extension.Extension{
				{
					ID:         uuid.FromStringOrNil("c3bb89e8-6f4d-11eb-b0dc-2f9c1d06a8ec"),
					CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
					DomainID:   uuid.FromStringOrNil("a4b2db1e-6f4d-11eb-9df6-5793191d903c"),
				},
				{
					ID:         uuid.FromStringOrNil("c4fb2336-6f4d-11eb-b51d-b318fdb3e042"),
					CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
					DomainID:   uuid.FromStringOrNil("a4b2db1e-6f4d-11eb-9df6-5793191d903c"),
				}},
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c3bb89e8-6f4d-11eb-b0dc-2f9c1d06a8ec","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"","detail":"","domain_id":"a4b2db1e-6f4d-11eb-9df6-5793191d903c","endpoint_id":"","aor_id":"","auth_id":"","extension":"","password":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"c4fb2336-6f4d-11eb-b51d-b318fdb3e042","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"","detail":"","domain_id":"a4b2db1e-6f4d-11eb-9df6-5793191d903c","endpoint_id":"","aor_id":"","auth_id":"","extension":"","password":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name:      "empty",
			domainID:  uuid.FromStringOrNil("c5231cce-6f4d-11eb-8a7f-3f6cf1546343"),
			pageToken: "2020-10-10T03:30:17.000000",
			pageSize:  10,
			request: &rabbitmqhandler.Request{
				URI:      "/v1/extensions?page_token=2020-10-10T03:30:17.000000&page_size=10&domain_id=c5231cce-6f4d-11eb-8a7f-3f6cf1546343",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			exts: []*extension.Extension{},
			expectRes: &rabbitmqhandler.Response{
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
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				reqHandler:       mockReq,
				domainHandler:    mockDomain,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().GetsByDomainID(gomock.Any(), tt.domainID, tt.pageToken, tt.pageSize).Return(tt.exts, nil)

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

func Test_processV1ExtensionsGet_customer_id(t *testing.T) {

	type test struct {
		name       string
		customerID uuid.UUID
		pageToken  string
		pageSize   uint64
		request    *rabbitmqhandler.Request
		exts       []*extension.Extension

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name:       "normal customer id",
			customerID: uuid.FromStringOrNil("1b642fde-4ff1-11ee-8b2f-2f40ea091b7d"),
			pageToken:  "2020-10-10T03:30:17.000000",
			pageSize:   10,
			request: &rabbitmqhandler.Request{
				URI:      "/v1/extensions?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=1b642fde-4ff1-11ee-8b2f-2f40ea091b7d",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			exts: []*extension.Extension{
				{
					ID:         uuid.FromStringOrNil("c3bb89e8-6f4d-11eb-b0dc-2f9c1d06a8ec"),
					CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
					DomainID:   uuid.FromStringOrNil("a4b2db1e-6f4d-11eb-9df6-5793191d903c"),
				},
				{
					ID:         uuid.FromStringOrNil("c4fb2336-6f4d-11eb-b51d-b318fdb3e042"),
					CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
					DomainID:   uuid.FromStringOrNil("a4b2db1e-6f4d-11eb-9df6-5793191d903c"),
				}},
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c3bb89e8-6f4d-11eb-b0dc-2f9c1d06a8ec","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"","detail":"","domain_id":"a4b2db1e-6f4d-11eb-9df6-5793191d903c","endpoint_id":"","aor_id":"","auth_id":"","extension":"","password":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"c4fb2336-6f4d-11eb-b51d-b318fdb3e042","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"","detail":"","domain_id":"a4b2db1e-6f4d-11eb-9df6-5793191d903c","endpoint_id":"","aor_id":"","auth_id":"","extension":"","password":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name:       "empty",
			customerID: uuid.FromStringOrNil("1b991686-4ff1-11ee-89fd-2f283e362ada"),
			pageToken:  "2020-10-10T03:30:17.000000",
			pageSize:   10,
			request: &rabbitmqhandler.Request{
				URI:      "/v1/extensions?page_token=2020-10-10T03:30:17.000000&page_size=10&customer_id=1b991686-4ff1-11ee-89fd-2f283e362ada",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			exts: []*extension.Extension{},
			expectRes: &rabbitmqhandler.Response{
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
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				reqHandler:       mockReq,
				domainHandler:    mockDomain,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().GetsByCustomerID(gomock.Any(), tt.customerID, tt.pageToken, tt.pageSize).Return(tt.exts, nil)

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

func Test_processV1ExtensionsPut(t *testing.T) {

	type test struct {
		name      string
		reqExt    *extension.Extension
		resExt    *extension.Extension
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"empty addresses",
			&extension.Extension{
				ID:       uuid.FromStringOrNil("6dc9dd22-6f4e-11eb-8059-2fe116db7a2b"),
				Name:     "update name",
				Detail:   "update detail",
				Password: "update password",
			},
			&extension.Extension{
				ID:         uuid.FromStringOrNil("6dc9dd22-6f4e-11eb-8059-2fe116db7a2b"),
				CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
				Name:       "update name",
				Detail:     "update detail",
				Password:   "update password",
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/extensions/6dc9dd22-6f4e-11eb-8059-2fe116db7a2b",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name", "detail":"update detail", "password": "update password"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6dc9dd22-6f4e-11eb-8059-2fe116db7a2b","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"update name","detail":"update detail","domain_id":"00000000-0000-0000-0000-000000000000","endpoint_id":"","aor_id":"","auth_id":"","extension":"","password":"update password","tm_create":"","tm_update":"","tm_delete":""}`),
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
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				reqHandler:       mockReq,
				domainHandler:    mockDomain,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().Update(gomock.Any(), tt.reqExt).Return(tt.resExt, nil)
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

func Test_processV1ExtensionsIDDelete(t *testing.T) {

	type test struct {
		name        string
		extensionID uuid.UUID
		request     *rabbitmqhandler.Request

		responseExtension *extension.Extension
		expectRes         *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("adeea2b0-6f4f-11eb-acb7-13291c18927b"),
			&rabbitmqhandler.Request{
				URI:    "/v1/extensions/adeea2b0-6f4f-11eb-acb7-13291c18927b",
				Method: rabbitmqhandler.RequestMethodDelete,
			},

			&extension.Extension{
				ID: uuid.FromStringOrNil("adeea2b0-6f4f-11eb-acb7-13291c18927b"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"adeea2b0-6f4f-11eb-acb7-13291c18927b","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","domain_id":"00000000-0000-0000-0000-000000000000","endpoint_id":"","aor_id":"","auth_id":"","extension":"","password":"","tm_create":"","tm_update":"","tm_delete":""}`),
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
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				reqHandler:       mockReq,
				domainHandler:    mockDomain,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().Delete(gomock.Any(), tt.extensionID).Return(tt.responseExtension, nil)
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

func Test_processV1ExtensionsExtensionEndpointGet(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseExtension *extension.Extension

		expectEndpoint string
		expectRes      *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			&rabbitmqhandler.Request{
				URI:    "/v1/extensions/endpoint/test_ext@test_domain",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			&extension.Extension{
				ID: uuid.FromStringOrNil("7e0dc99c-e4e3-491a-8f66-9ccb2cea44c6"),
			},

			"test_ext@test_domain",
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7e0dc99c-e4e3-491a-8f66-9ccb2cea44c6","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","domain_id":"00000000-0000-0000-0000-000000000000","endpoint_id":"","aor_id":"","auth_id":"","extension":"","password":"","tm_create":"","tm_update":"","tm_delete":""}`),
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
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				reqHandler:       mockReq,
				domainHandler:    mockDomain,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().GetByEndpoint(gomock.Any(), tt.expectEndpoint).Return(tt.responseExtension, nil)
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
