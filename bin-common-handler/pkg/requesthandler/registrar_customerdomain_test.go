package requesthandler

import (
	"context"
	reflect "reflect"
	"testing"

	rmcustomerdomain "monorepo/bin-registrar-manager/models/customerdomain"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_RegistrarV1CustomerDomainGetByRealm(t *testing.T) {

	tests := []struct {
		name string

		realm string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *rmcustomerdomain.CustomerDomain
	}{
		{
			name: "normal",

			realm: "ab12.reg.voipbin.net",
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","domain_label":"ab12","realm":"ab12.reg.voipbin.net"}`),
			},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/customer_domains/realm/ab12.reg.voipbin.net",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			expectRes: &rmcustomerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				DomainLabel: "ab12",
				Realm:       "ab12.reg.voipbin.net",
			},
		},
		{
			name: "legacy uuid realm",

			realm: "5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","domain_label":"5e4a0680-804e-11ec-8477-2fea5968d85b","realm":"5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net"}`),
			},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/customer_domains/realm/5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			expectRes: &rmcustomerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				DomainLabel: "5e4a0680-804e-11ec-8477-2fea5968d85b",
				Realm:       "5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1CustomerDomainGetByRealm(ctx, tt.realm)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RegistrarV1CustomerDomainGetByRealm_error(t *testing.T) {

	tests := []struct {
		name string

		realm string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			name: "not found",

			realm: "zzzz.reg.voipbin.net",
			response: &sock.Response{
				StatusCode: 404,
			},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/customer_domains/realm/zzzz.reg.voipbin.net",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
		},
		{
			name: "internal server error",

			realm: "ab12.reg.voipbin.net",
			response: &sock.Response{
				StatusCode: 500,
			},

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/customer_domains/realm/ab12.reg.voipbin.net",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1CustomerDomainGetByRealm(ctx, tt.realm)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok, res: %v", res)
			}
		})
	}
}
