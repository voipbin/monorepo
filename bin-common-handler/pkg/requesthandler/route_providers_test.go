package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	rmprovider "monorepo/bin-route-manager/models/provider"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_RouteV1ProviderCreate(t *testing.T) {

	tests := []struct {
		name string

		providerType rmprovider.Type
		hostname     string
		techPrefix   string
		techPostfix  string
		techHeaders  map[string]string
		providerName string
		detail       string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *rmprovider.Provider
	}{
		{
			"normal",

			rmprovider.TypeSIP,
			"test.com",
			"0001",
			"1000",
			map[string]string{
				"header1": "val1",
				"header2": "val2",
			},
			"test name",
			"test detail",

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/providers",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"type":"sip","hostname":"test.com","tech_prefix":"0001","tech_postfix":"1000","tech_headers":{"header1":"val1","header2":"val2"},"name":"test name","detail":"test detail"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9f5b518a-e882-49a2-8282-a9d4e95019cf"}`),
			},
			&rmprovider.Provider{
				ID: uuid.FromStringOrNil("9f5b518a-e882-49a2-8282-a9d4e95019cf"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RouteV1ProviderCreate(ctx, tt.providerType, tt.hostname, tt.techPrefix, tt.techPostfix, tt.techHeaders, tt.providerName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RouteV1ProviderGet(t *testing.T) {

	tests := []struct {
		name string

		providerID uuid.UUID

		responseRoute *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmprovider.Provider
	}{
		{
			"normal",

			uuid.FromStringOrNil("f15d8304-c13d-40e2-90e6-c602190dc0e1"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f15d8304-c13d-40e2-90e6-c602190dc0e1"}`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/providers/f15d8304-c13d-40e2-90e6-c602190dc0e1",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			&rmprovider.Provider{
				ID: uuid.FromStringOrNil("f15d8304-c13d-40e2-90e6-c602190dc0e1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.responseRoute, nil)

			res, err := reqHandler.RouteV1ProviderGet(ctx, tt.providerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RouteV1ProviderDelete(t *testing.T) {

	tests := []struct {
		name string

		routeID uuid.UUID

		responseRoute *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmprovider.Provider
	}{
		{
			"normal",

			uuid.FromStringOrNil("3f26d6c1-576c-46b3-91ad-1fbc1416b8d1"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3f26d6c1-576c-46b3-91ad-1fbc1416b8d1"}`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/providers/3f26d6c1-576c-46b3-91ad-1fbc1416b8d1",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeNone,
			},
			&rmprovider.Provider{
				ID: uuid.FromStringOrNil("3f26d6c1-576c-46b3-91ad-1fbc1416b8d1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.responseRoute, nil)

			res, err := reqHandler.RouteV1ProviderDelete(ctx, tt.routeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_RouteV1ProviderUpdate(t *testing.T) {

	tests := []struct {
		name string

		providerID   uuid.UUID
		providerType rmprovider.Type
		hostname     string
		techPrefix   string
		techPostfix  string
		techHeaders  map[string]string
		providerName string
		detail       string

		responseRoute *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmprovider.Provider
	}{
		{
			"normal",

			uuid.FromStringOrNil("c5e7f18c-fc5a-4520-8326-e534e2ca0b8f"),
			rmprovider.TypeSIP,
			"test.com",
			"0001",
			"1000",
			map[string]string{
				"header1": "val1",
				"header2": "val2",
			},
			"test name",
			"test detail",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c5e7f18c-fc5a-4520-8326-e534e2ca0b8f"}`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      "/v1/providers/c5e7f18c-fc5a-4520-8326-e534e2ca0b8f",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"type":"sip","hostname":"test.com","tech_prefix":"0001","tech_postfix":"1000","tech_headers":{"header1":"val1","header2":"val2"},"name":"test name","detail":"test detail"}`),
			},
			&rmprovider.Provider{
				ID: uuid.FromStringOrNil("c5e7f18c-fc5a-4520-8326-e534e2ca0b8f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.responseRoute, nil)

			res, err := reqHandler.RouteV1ProviderUpdate(ctx, tt.providerID, tt.providerType, tt.hostname, tt.techPrefix, tt.techPostfix, tt.techHeaders, tt.providerName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RouteV1ProviderGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []rmprovider.Provider
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"6eda3fb7-d99b-474a-af80-ebc11a6661cc"}]`),
			},

			"bin-manager.route-manager.request",
			&sock.Request{
				URI:      fmt.Sprintf("/v1/providers?page_token=%s&page_size=10", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			[]rmprovider.Provider{
				{
					ID: uuid.FromStringOrNil("6eda3fb7-d99b-474a-af80-ebc11a6661cc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RouteV1ProviderGets(ctx, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
