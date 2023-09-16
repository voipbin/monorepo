package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	rmtrunk "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_RegistrarV1TrunkCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		trunkName  string
		detail     string
		domainName string
		authTypes  []rmtrunk.AuthType
		username   string
		password   string
		allowedIPs []string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *rmtrunk.Trunk
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("dbb730a8-549a-11ee-a7f3-4f1384f81f27"),
			trunkName:  "test name",
			detail:     "test detail",
			domainName: "test-domain",
			authTypes:  []rmtrunk.AuthType{rmtrunk.AuthTypeBasic, rmtrunk.AuthTypeIP},
			username:   "testusername",
			password:   "testpassword",
			allowedIPs: []string{"1.2.3.4"},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/trunks",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"dbb730a8-549a-11ee-a7f3-4f1384f81f27","name":"test name","detail":"test detail","domain_name":"test name","auth_types":["basic","ip"],"username":"testusername","password":"testpassword","allowed_ips":["1.2.3.4"]}`),
			},
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dc3b0e5a-549a-11ee-9469-abda3b219d1d"}`),
			},
			expectRes: &rmtrunk.Trunk{
				ID: uuid.FromStringOrNil("dc3b0e5a-549a-11ee-9469-abda3b219d1d"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1TrunkCreate(ctx, tt.customerID, tt.trunkName, tt.detail, tt.domainName, tt.authTypes, tt.username, tt.password, tt.allowedIPs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RegistrarV1TrunkGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     []rmtrunk.Trunk
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("b1e8f6c0-549b-11ee-a209-a7fe0fc532fa"),
			pageToken:  "2020-09-20 03:23:20.995000",
			pageSize:   10,

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				Data:       []byte(`[{"id":"b215904a-549b-11ee-874c-7f01e2fb3e8c"}]`),
			},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/trunks?page_token=%s&page_size=10&customer_id=b1e8f6c0-549b-11ee-a209-a7fe0fc532fa", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			expectRes: []rmtrunk.Trunk{
				{
					ID: uuid.FromStringOrNil("b215904a-549b-11ee-874c-7f01e2fb3e8c"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1TrunkGetsByCustomerID(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RegistrarV1TrunkGet(t *testing.T) {

	tests := []struct {
		name string

		trunkID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *rmtrunk.Trunk
	}{
		{
			name: "normal",

			trunkID: uuid.FromStringOrNil("f5547ab0-549b-11ee-a653-93228d9f8207"),
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f5547ab0-549b-11ee-a653-93228d9f8207"}`),
			},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/trunks/f5547ab0-549b-11ee-a653-93228d9f8207",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			expectRes: &rmtrunk.Trunk{
				ID: uuid.FromStringOrNil("f5547ab0-549b-11ee-a653-93228d9f8207"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1TrunkGet(ctx, tt.trunkID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RegistrarV1TrunkGetByDomainName(t *testing.T) {

	tests := []struct {
		name string

		domainName string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *rmtrunk.Trunk
	}{
		{
			name: "normal",

			domainName: "test-domain",
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4b766408-549c-11ee-a5ad-077c11ba6415"}`),
			},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/trunks/domain_name/test-domain",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			expectRes: &rmtrunk.Trunk{
				ID: uuid.FromStringOrNil("4b766408-549c-11ee-a5ad-077c11ba6415"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1TrunkGetByDomainName(ctx, tt.domainName)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_RegistrarV1TrunkDelete(t *testing.T) {

	tests := []struct {
		name string

		trunkID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *rmtrunk.Trunk
	}{
		{
			name: "normal",

			trunkID: uuid.FromStringOrNil("98fbcfba-549c-11ee-8a74-73230f51555d"),
			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"98fbcfba-549c-11ee-8a74-73230f51555d"}`),
			},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/trunks/98fbcfba-549c-11ee-8a74-73230f51555d",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeNone,
			},
			expectRes: &rmtrunk.Trunk{
				ID: uuid.FromStringOrNil("98fbcfba-549c-11ee-8a74-73230f51555d"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1TrunkDelete(ctx, tt.trunkID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_RegistrarV1TrunkUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		trunkID    uuid.UUID
		trunkName  string
		detail     string
		authTypes  []rmtrunk.AuthType
		username   string
		password   string
		allowedIPs []string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *rmtrunk.Trunk
	}{
		{
			name: "normal",

			trunkID:    uuid.FromStringOrNil("f27448ce-549c-11ee-b466-57162d71a670"),
			trunkName:  "update name",
			detail:     "update detail",
			authTypes:  []rmtrunk.AuthType{rmtrunk.AuthTypeBasic, rmtrunk.AuthTypeIP},
			username:   "updateusername",
			password:   "updatepassword",
			allowedIPs: []string{"1.2.3.4"},

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f27448ce-549c-11ee-b466-57162d71a670"}`),
			},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/trunks/f27448ce-549c-11ee-b466-57162d71a670",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail","auth_types":["basic","ip"],"username":"updateusername","password":"updatepassword","allowed_ips":["1.2.3.4"]}`),
			},
			expectRes: &rmtrunk.Trunk{
				ID: uuid.FromStringOrNil("f27448ce-549c-11ee-b466-57162d71a670"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1TrunkUpdateBasicInfo(ctx, tt.trunkID, tt.trunkName, tt.detail, tt.authTypes, tt.username, tt.password, tt.allowedIPs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRes, *res)
			}
		})
	}
}
