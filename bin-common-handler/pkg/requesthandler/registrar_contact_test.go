package requesthandler

import (
	"context"
	"reflect"
	"testing"

	astcontact "monorepo/bin-registrar-manager/models/astcontact"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_RegistrarV1ContactGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		extension  string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []astcontact.AstContact
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("390f34ba-57a4-11ee-a22c-d3dbf1f5af19"),
			extension:  "test_exten",

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/contacts?customer_id=390f34ba-57a4-11ee-a22c-d3dbf1f5af19&extension=test_exten",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"uri":"sip:test11@211.200.20.28:48540^3Btransport=udp^3Balias=211.200.20.28~48540~1"},{"uri":"sip:test11@223.38.28.126:48540^3Btransport=udp^3Balias=223.38.28.126~48540~1"},{"uri":"sip:test11@35.204.215.63^3Btransport=udp^3Balias=35.204.215.63~5060~1"}]`),
			},
			expectRes: []astcontact.AstContact{
				{
					URI: "sip:test11@211.200.20.28:48540^3Btransport=udp^3Balias=211.200.20.28~48540~1",
				},
				{
					URI: "sip:test11@223.38.28.126:48540^3Btransport=udp^3Balias=223.38.28.126~48540~1",
				},
				{
					URI: "sip:test11@35.204.215.63^3Btransport=udp^3Balias=35.204.215.63~5060~1",
				},
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

			res, err := reqHandler.RegistrarV1ContactGets(ctx, tt.customerID, tt.extension)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_RegistrarV1ContactRefresh(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		extension  string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []astcontact.AstContact
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("e168826a-57a4-11ee-818c-73dfee4986c0"),
			extension:  "test_exten",

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/contacts?customer_id=e168826a-57a4-11ee-818c-73dfee4986c0&extension=test_exten",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeNone,
			},
			response: &sock.Response{
				StatusCode: 200,
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

			if err := reqHandler.RegistrarV1ContactRefresh(ctx, tt.customerID, tt.extension); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
