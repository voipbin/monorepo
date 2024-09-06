package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/providerhandler"
	"monorepo/bin-route-manager/pkg/routehandler"
)

func Test_v1ProvidersPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		providerType   provider.Type
		hostname       string
		techPrefix     string
		techPostfix    string
		techHeaders    map[string]string
		provierName    string
		providerDetail string

		responseRoute *provider.Provider
		expectRes     *rabbitmqhandler.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/providers",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type": "sip", "hostname": "test.com", "tech_prefix":"0001", "tech_postfix":"1000", "tech_headers":{"HEADER1":"val1", "HEADER2":"val2"}, "name":"test name", "detail": "test detail"}`),
			},

			provider.TypeSIP,
			"test.com",
			"0001",
			"1000",
			map[string]string{
				"HEADER1": "val1",
				"HEADER2": "val2",
			},
			"test name",
			"test detail",

			&provider.Provider{
				ID: uuid.FromStringOrNil("997a7752-4872-11ed-be7a-5783111a9092"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"997a7752-4872-11ed-be7a-5783111a9092","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockProvider.EXPECT().Create(gomock.Any(), tt.providerType, tt.hostname, tt.techPrefix, tt.techPostfix, tt.techHeaders, tt.provierName, tt.providerDetail).Return(tt.responseRoute, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1ProvidersGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken string
		pageSize  uint64

		responseProviders []*provider.Provider

		expectRes *rabbitmqhandler.Response
	}{
		{
			"1 item",
			&sock.Request{
				URI:      "/v1/providers?page_token=2020-10-10T03:30:17.000000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10T03:30:17.000000",
			10,

			[]*provider.Provider{
				{
					ID: uuid.FromStringOrNil("104eef98-7492-473d-b058-579364d20e6b"),
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"104eef98-7492-473d-b058-579364d20e6b","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",
			&sock.Request{
				URI:      "/v1/providers?page_token=2020-10-10T03:30:17.000000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10T03:30:17.000000",
			10,

			[]*provider.Provider{
				{
					ID: uuid.FromStringOrNil("df5c4b4d-a75d-45d3-a27c-ec6686dcd467"),
				},
				{
					ID: uuid.FromStringOrNil("eac421c0-a0b4-4d33-8184-ffcbe80a92fb"),
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"df5c4b4d-a75d-45d3-a27c-ec6686dcd467","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"eac421c0-a0b4-4d33-8184-ffcbe80a92fb","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"empty response",
			&sock.Request{
				URI:      "/v1/providers?page_token=2020-10-10T03:30:17.000000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10T03:30:17.000000",
			10,

			[]*provider.Provider{},

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
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockProvider.EXPECT().Gets(gomock.Any(), tt.pageToken, tt.pageSize).Return(tt.responseProviders, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1ProvidersIDGet(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		providerID uuid.UUID

		responseProvider *provider.Provider

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/providers/30bc4952-efcc-4944-95d8-df8e7f571479",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},
			uuid.FromStringOrNil("30bc4952-efcc-4944-95d8-df8e7f571479"),

			&provider.Provider{
				ID: uuid.FromStringOrNil("30bc4952-efcc-4944-95d8-df8e7f571479"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"30bc4952-efcc-4944-95d8-df8e7f571479","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockProvider.EXPECT().Get(gomock.Any(), tt.providerID).Return(tt.responseProvider, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1ProvidersIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID

		providerType provider.Type
		hostname     string

		techPrefix   string
		techPostfix  string
		techHeaders  map[string]string
		providerName string
		detail       string

		responseRoute *provider.Provider
		expectRes     *rabbitmqhandler.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/providers/83cfba90-d8a4-48e2-a9d0-dae964937163",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"type":"sip", "hostname":"test.com", "tech_prefix":"0001", "tech_postfix":"1000","tech_headers":{"header1":"val1","header2":"val2"},"name":"test name","detail":"test detail"}`),
			},
			uuid.FromStringOrNil("83cfba90-d8a4-48e2-a9d0-dae964937163"),

			provider.TypeSIP,
			"test.com",
			"0001",
			"1000",
			map[string]string{
				"header1": "val1",
				"header2": "val2",
			},
			"test name",
			"test detail",

			&provider.Provider{
				ID: uuid.FromStringOrNil("83cfba90-d8a4-48e2-a9d0-dae964937163"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"83cfba90-d8a4-48e2-a9d0-dae964937163","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockProvider.EXPECT().Update(gomock.Any(), tt.id, tt.providerType, tt.hostname, tt.techPrefix, tt.techPostfix, tt.techHeaders, tt.providerName, tt.detail).Return(tt.responseRoute, nil)

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

func Test_v1ProvidersIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID

		responseProvider *provider.Provider
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/providers/be3be98f-d434-4ce9-9374-71b3932de735",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},
			uuid.FromStringOrNil("be3be98f-d434-4ce9-9374-71b3932de735"),

			&provider.Provider{
				ID: uuid.FromStringOrNil("be3be98f-d434-4ce9-9374-71b3932de735"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"be3be98f-d434-4ce9-9374-71b3932de735","type":"","hostname":"","tech_prefix":"","tech_postfix":"","tech_headers":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockRoute := routehandler.NewMockRouteHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				routeHandler:    mockRoute,
				providerHandler: mockProvider,
			}

			mockProvider.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseProvider, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
