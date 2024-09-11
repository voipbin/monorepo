package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
)

func Test_processV1ExtensionsPost(t *testing.T) {

	type test struct {
		name string

		request *sock.Request

		expectCustomerID uuid.UUID
		expectName       string
		expectDetail     string
		expectExtension  string
		expectPassword   string

		responseExtension *extension.Extension
		expectRes         *sock.Response
	}

	tests := []test{
		{
			"normal",

			&sock.Request{
				URI:      "/v1/extensions",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "2e341ffa-7fed-11ec-9667-1357b91d745d", "name": "test name", "detail": "test detail", "extension": "45eb6bac-6ebf-11eb-bcf3-3b9157826d22", "password": "4b1f7a6e-6ebf-11eb-a47e-5351700cd612"}`),
			},

			uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
			"test name",
			"test detail",
			"45eb6bac-6ebf-11eb-bcf3-3b9157826d22",
			"4b1f7a6e-6ebf-11eb-a47e-5351700cd612",

			&extension.Extension{
				ID:         uuid.FromStringOrNil("3f4bc63e-6ebf-11eb-b7de-df47266bf559"),
				CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),

				EndpointID: "45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net",
				AORID:      "45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net",
				AuthID:     "45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net",

				Extension: "45eb6bac-6ebf-11eb-bcf3-3b9157826d22",
				Password:  "4b1f7a6e-6ebf-11eb-a47e-5351700cd612",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3f4bc63e-6ebf-11eb-b7de-df47266bf559","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"","detail":"","endpoint_id":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net","aor_id":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net","auth_id":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22@test.sip.voipbin.net","extension":"45eb6bac-6ebf-11eb-bcf3-3b9157826d22","domain_name":"","realm":"","username":"","password":"4b1f7a6e-6ebf-11eb-a47e-5351700cd612","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)
			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().Create(
				gomock.Any(),
				tt.expectCustomerID,
				tt.expectName,
				tt.expectDetail,
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

func Test_processV1ExtensionsGet(t *testing.T) {

	type test struct {
		name string

		pageToken string
		pageSize  uint64
		request   *sock.Request

		responseFilters    map[string]string
		responseExtensions []*extension.Extension

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal",

			pageToken: "2020-10-10T03:30:17.000000",
			pageSize:  10,
			request: &sock.Request{
				URI:      "/v1/extensions?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=1b642fde-4ff1-11ee-8b2f-2f40ea091b7d",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			responseFilters: map[string]string{
				"filter_customer_id": "1b642fde-4ff1-11ee-8b2f-2f40ea091b7d",
			},
			responseExtensions: []*extension.Extension{
				{
					ID:         uuid.FromStringOrNil("c3bb89e8-6f4d-11eb-b0dc-2f9c1d06a8ec"),
					CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
				},
				{
					ID:         uuid.FromStringOrNil("c4fb2336-6f4d-11eb-b51d-b318fdb3e042"),
					CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
				}},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c3bb89e8-6f4d-11eb-b0dc-2f9c1d06a8ec","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"","detail":"","endpoint_id":"","aor_id":"","auth_id":"","extension":"","domain_name":"","realm":"","username":"","password":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"c4fb2336-6f4d-11eb-b51d-b318fdb3e042","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"","detail":"","endpoint_id":"","aor_id":"","auth_id":"","extension":"","domain_name":"","realm":"","username":"","password":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			name: "empty",

			pageToken: "2020-10-10T03:30:17.000000",
			pageSize:  10,
			request: &sock.Request{
				URI:      "/v1/extensions?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=1b991686-4ff1-11ee-89fd-2f283e362ada",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			responseFilters: map[string]string{
				"filter_customer_id": "1b991686-4ff1-11ee-89fd-2f283e362ada",
			},
			responseExtensions: []*extension.Extension{},
			expectRes: &sock.Response{
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
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				utilHandler:      mockUtil,
				extensionHandler: mockExtension,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockExtension.EXPECT().Gets(gomock.Any(), tt.pageToken, tt.pageSize, tt.responseFilters).Return(tt.responseExtensions, nil)

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
		name    string
		reqExt  *extension.Extension
		resExt  *extension.Extension
		request *sock.Request

		expectID       uuid.UUID
		expectName     string
		expectDetail   string
		expectPassword string
		expectRes      *sock.Response
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
			&sock.Request{
				URI:      "/v1/extensions/6dc9dd22-6f4e-11eb-8059-2fe116db7a2b",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name", "detail":"update detail", "password": "update password"}`),
			},

			uuid.FromStringOrNil("6dc9dd22-6f4e-11eb-8059-2fe116db7a2b"),
			"update name",
			"update detail",
			"update password",
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6dc9dd22-6f4e-11eb-8059-2fe116db7a2b","customer_id":"2e341ffa-7fed-11ec-9667-1357b91d745d","name":"update name","detail":"update detail","endpoint_id":"","aor_id":"","auth_id":"","extension":"","domain_name":"","realm":"","username":"","password":"update password","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().Update(gomock.Any(), tt.expectID, tt.expectName, tt.expectDetail, tt.expectPassword).Return(tt.resExt, nil)
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
		request     *sock.Request

		responseExtension *extension.Extension
		expectRes         *sock.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("adeea2b0-6f4f-11eb-acb7-13291c18927b"),
			&sock.Request{
				URI:    "/v1/extensions/adeea2b0-6f4f-11eb-acb7-13291c18927b",
				Method: sock.RequestMethodDelete,
			},

			&extension.Extension{
				ID: uuid.FromStringOrNil("adeea2b0-6f4f-11eb-acb7-13291c18927b"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"adeea2b0-6f4f-11eb-acb7-13291c18927b","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","endpoint_id":"","aor_id":"","auth_id":"","extension":"","domain_name":"","realm":"","username":"","password":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
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

func Test_processV1ExtensionsExtensionExtensionGet(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseExtension *extension.Extension

		expectCustomerID uuid.UUID
		expectExt        string
		expectRes        *sock.Response
	}

	tests := []test{
		{
			"normal",

			&sock.Request{
				URI:    `/v1/extensions/extension/test_ext?customer_id=14529572-5650-11ee-8bac-8f91175c7ceb`,
				Method: sock.RequestMethodGet,
			},

			&extension.Extension{
				ID: uuid.FromStringOrNil("922a32a2-5650-11ee-8341-cb03501d873e"),
			},

			uuid.FromStringOrNil("14529572-5650-11ee-8bac-8f91175c7ceb"),
			"test_ext",
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"922a32a2-5650-11ee-8341-cb03501d873e","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","endpoint_id":"","aor_id":"","auth_id":"","extension":"","domain_name":"","realm":"","username":"","password":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().GetByExtension(gomock.Any(), tt.expectCustomerID, tt.expectExt).Return(tt.responseExtension, nil)
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
