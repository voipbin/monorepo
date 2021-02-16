package listenhandler

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/contacthandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/requesthandler"
)

func TestV1ContactsGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)

	h := &listenHandler{
		rabbitSock:     mockSock,
		reqHandler:     mockReq,
		contactHandler: mockContact,
	}

	type test struct {
		name     string
		endpoint string
		request  *rabbitmqhandler.Request
		contacts []*models.AstContact

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			"test@test.sip.voipbin.net",
			&rabbitmqhandler.Request{
				URI:      "/v1/contacts?endpoint=test%40test.sip.voipbin.net",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*models.AstContact{
				{
					ID:                  "test11@test.sip.voipbin.net^3B@c21de7824c22185a665983170d7028b0",
					URI:                 "sip:test11@211.178.226.108:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22",
					ExpirationTime:      1613498199,
					QualifyFrequency:    0,
					OutboundProxy:       "",
					Path:                "",
					UserAgent:           "Z 5.4.9 rv2.10.11.7-mod",
					QualifyTimeout:      3,
					RegServer:           "asterisk-registrar-b46bf4b67-j5rxz",
					AuthenticateQualify: "no",
					ViaAddr:             "192.168.0.20",
					ViaPort:             35551,
					CallID:              "mX4vXXxJZ_gS4QpMapYfwA..",
					Endpoint:            "test@test.sip.voipbin.net",
					PruneOnBoot:         "no",
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"test11@test.sip.voipbin.net^3B@c21de7824c22185a665983170d7028b0","uri":"sip:test11@211.178.226.108:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22","expiration_time":1613498199,"qualify_frequency":0,"outbound_proxy":"","path":"","user_agent":"Z 5.4.9 rv2.10.11.7-mod","qualify_timeout":3,"reg_server":"asterisk-registrar-b46bf4b67-j5rxz","authenticate_qualify":"no","via_addr":"192.168.0.20","via_port":35551,"call_id":"mX4vXXxJZ_gS4QpMapYfwA..","endpoint":"test@test.sip.voipbin.net","prune_on_boot":"no"}]`),
			},
		},
		{
			"empty",
			"test2@test.sip.voipbin.net",
			&rabbitmqhandler.Request{
				URI:      "/v1/contacts?endpoint=test2%40test.sip.voipbin.net",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*models.AstContact{},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockContact.EXPECT().ContactGetsByEndpoint(gomock.Any(), tt.endpoint).Return(tt.contacts, nil)

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
