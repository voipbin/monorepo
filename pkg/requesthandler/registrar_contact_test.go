package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	astcontact "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
)

func Test_RegistrarV1ContactGets(t *testing.T) {

	tests := []struct {
		name string

		endpoint string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes []astcontact.AstContact
	}{
		{
			name: "normal",

			endpoint: "test_exten@test_domain",

			expectTarget: "bin-manager.registrar-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/contacts?endpoint=test_exten%40test_domain",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			response: &rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.RegistrarV1ContactGets(ctx, tt.endpoint)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
