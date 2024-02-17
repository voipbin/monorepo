package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/trunkhandler"
)

func Test_processV1TrunksPost(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		trunkName  string
		detail     string
		domainName string
		authTypes  []sipauth.AuthType
		username   string
		password   string
		allowedIPs []string

		resTrunk  *trunk.Trunk
		request   *rabbitmqhandler.Request
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
			"test name",
			"test detail",
			"21b7ae32-5231-11ee-b7da-7f436158317b",
			[]sipauth.AuthType{sipauth.AuthTypeBasic},
			"testusername",
			"testpassword",
			[]string{
				"1.2.3.4",
			},

			&trunk.Trunk{
				ID: uuid.FromStringOrNil("1744ccb4-6e13-11eb-b08d-bb42431b2fb3"),
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/trunks",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "8c1f0206-7fed-11ec-bc4d-b75bc59a142c", "name": "test name", "detail": "test detail", "domain_name": "21b7ae32-5231-11ee-b7da-7f436158317b", "auth_types": ["basic"], "username": "testusername", "password": "testpassword", "allowed_ips": ["1.2.3.4"]}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1744ccb4-6e13-11eb-b08d-bb42431b2fb3","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				reqHandler:   mockReq,
				trunkHandler: mockTrunk,
			}

			mockTrunk.EXPECT().Create(gomock.Any(), tt.customerID, tt.trunkName, tt.detail, tt.domainName, tt.authTypes, tt.username, tt.password, tt.allowedIPs).Return(tt.resTrunk, nil)
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

func Test_processV1TrunksGet(t *testing.T) {

	type test struct {
		name       string
		customerID uuid.UUID
		pageToken  string
		pageSize   uint64
		request    *rabbitmqhandler.Request

		responseFilters map[string]string
		responseTrunks  []*trunk.Trunk

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/trunks?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=8c1f0206-7fed-11ec-bc4d-b75bc59a142c",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			map[string]string{
				"customer_id": "8c1f0206-7fed-11ec-bc4d-b75bc59a142c",
			},
			[]*trunk.Trunk{
				{
					ID:         uuid.FromStringOrNil("abd3467a-6ee6-11eb-824f-c386fbaad128"),
					CustomerID: uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
				},
				{
					ID:         uuid.FromStringOrNil("af6488da-6ee6-11eb-8d4d-0f848f8e1aee"),
					CustomerID: uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
				}},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"abd3467a-6ee6-11eb-824f-c386fbaad128","customer_id":"8c1f0206-7fed-11ec-bc4d-b75bc59a142c"},{"id":"af6488da-6ee6-11eb-8d4d-0f848f8e1aee","customer_id":"8c1f0206-7fed-11ec-bc4d-b75bc59a142c"}]`),
			},
		},
		{
			"empty",
			uuid.FromStringOrNil("8c1f0206-7fed-11ec-bc4d-b75bc59a142c"),
			"2020-10-10T03:30:17.000000",
			10,
			&rabbitmqhandler.Request{
				URI:      "/v1/trunks?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=8c1f0206-7fed-11ec-bc4d-b75bc59a142c",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			map[string]string{
				"customer_id": "8c1f0206-7fed-11ec-bc4d-b75bc59a142c",
			},
			[]*trunk.Trunk{},
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
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				reqHandler:   mockReq,
				utilHandler:  mockUtil,
				trunkHandler: mockTrunk,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockTrunk.EXPECT().Gets(gomock.Any(), tt.pageToken, tt.pageSize, tt.responseFilters).Return(tt.responseTrunks, nil)

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

func Test_processV1TrunksIDPut(t *testing.T) {

	type test struct {
		name string

		id         uuid.UUID
		trunkName  string
		detail     string
		authTypes  []sipauth.AuthType
		username   string
		password   string
		allowedIPs []string

		request   *rabbitmqhandler.Request
		resTrunk  *trunk.Trunk
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("a3e97272-5232-11ee-acd9-bbb3933eed48"),
			"update name",
			"update detail",
			[]sipauth.AuthType{
				sipauth.AuthTypeBasic,
			},
			"testusername",
			"testpassword",
			[]string{
				"1.2.3.4",
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/trunks/a3e97272-5232-11ee-acd9-bbb3933eed48",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name", "detail":"update detail", "auth_types": ["basic"], "username": "testusername", "password": "testpassword", "allowed_ips": ["1.2.3.4"]}`),
			},
			&trunk.Trunk{
				ID: uuid.FromStringOrNil("a3e97272-5232-11ee-acd9-bbb3933eed48"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a3e97272-5232-11ee-acd9-bbb3933eed48","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				reqHandler:   mockReq,
				trunkHandler: mockTrunk,
			}

			mockTrunk.EXPECT().Update(gomock.Any(), tt.id, tt.trunkName, tt.detail, tt.authTypes, tt.username, tt.password, tt.allowedIPs).Return(tt.resTrunk, nil)
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

func Test_processV1TrunksIDGet(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		request   *rabbitmqhandler.Request
		resTrunk  *trunk.Trunk
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("4e1f3c12-5234-11ee-ad7f-ef5be37113b2"),

			&rabbitmqhandler.Request{
				URI:    "/v1/trunks/4e1f3c12-5234-11ee-ad7f-ef5be37113b2",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&trunk.Trunk{
				ID: uuid.FromStringOrNil("4e1f3c12-5234-11ee-ad7f-ef5be37113b2"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4e1f3c12-5234-11ee-ad7f-ef5be37113b2","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				reqHandler:   mockReq,
				trunkHandler: mockTrunk,
			}

			mockTrunk.EXPECT().Get(gomock.Any(), tt.id).Return(tt.resTrunk, nil)
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

func Test_processV1TrunksTrunkNameTrunkNameGet(t *testing.T) {

	type test struct {
		name string

		request       *rabbitmqhandler.Request
		responseTrunk *trunk.Trunk

		expectDomainName string
		expectRes        *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/trunks/domain_name/testdomain",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&trunk.Trunk{
				ID: uuid.FromStringOrNil("d5829769-dacf-420e-9260-c8931560331e"),
			},

			"testdomain",
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d5829769-dacf-420e-9260-c8931560331e","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				reqHandler:   mockReq,
				trunkHandler: mockTrunk,
			}

			mockTrunk.EXPECT().GetByDomainName(gomock.Any(), tt.expectDomainName).Return(tt.responseTrunk, nil)
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

func Test_processV1TrunksDelete(t *testing.T) {

	type test struct {
		name    string
		trunkID uuid.UUID

		request       *rabbitmqhandler.Request
		responseTrunk *trunk.Trunk

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("09e94cb4-6f32-11eb-af29-27dcd65a7064"),
			&rabbitmqhandler.Request{
				URI:    "/v1/trunks/09e94cb4-6f32-11eb-af29-27dcd65a7064",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&trunk.Trunk{
				ID: uuid.FromStringOrNil("09e94cb4-6f32-11eb-af29-27dcd65a7064"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"09e94cb4-6f32-11eb-af29-27dcd65a7064","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)

			h := &listenHandler{
				rabbitSock:   mockSock,
				reqHandler:   mockReq,
				trunkHandler: mockTrunk,
			}

			mockTrunk.EXPECT().Delete(gomock.Any(), tt.trunkID).Return(tt.responseTrunk, nil)
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
