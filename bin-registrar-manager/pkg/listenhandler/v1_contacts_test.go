package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/contacthandler"
)

func Test_processV1ContactsGet(t *testing.T) {

	type test struct {
		name             string
		customerID       uuid.UUID
		expectExtension  string
		request          *rabbitmqhandler.Request
		responseContacts []*astcontact.AstContact

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("2f905272-5653-11ee-b4df-f3faa1c18732"),
			"test",
			&rabbitmqhandler.Request{
				URI:    "/v1/contacts?customer_id=2f905272-5653-11ee-b4df-f3faa1c18732&extension=test",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			[]*astcontact.AstContact{
				{
					ID:                  "test11@2f905272-5653-11ee-b4df-f3faa1c18732.registrar.voipbin.net^3B@c21de7824c22185a665983170d7028b0",
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
					Endpoint:            "test@2f905272-5653-11ee-b4df-f3faa1c18732.registrar.sip.voipbin.net",
					PruneOnBoot:         "no",
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"test11@2f905272-5653-11ee-b4df-f3faa1c18732.registrar.voipbin.net^3B@c21de7824c22185a665983170d7028b0","uri":"sip:test11@211.178.226.108:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22","expiration_time":1613498199,"qualify_frequency":0,"outbound_proxy":"","path":"","user_agent":"Z 5.4.9 rv2.10.11.7-mod","qualify_timeout":3,"reg_server":"asterisk-registrar-b46bf4b67-j5rxz","authenticate_qualify":"no","via_addr":"192.168.0.20","via_port":35551,"call_id":"mX4vXXxJZ_gS4QpMapYfwA..","endpoint":"test@2f905272-5653-11ee-b4df-f3faa1c18732.registrar.sip.voipbin.net","prune_on_boot":"no"}]`),
			},
		},
		{
			"empty",
			uuid.FromStringOrNil("4962c82e-5653-11ee-96e1-4fca4502226b"),
			"test2",
			&rabbitmqhandler.Request{
				URI:      "/v1/contacts?customer_id=4962c82e-5653-11ee-96e1-4fca4502226b&extension=test2",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			[]*astcontact.AstContact{},
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
			mockContact := contacthandler.NewMockContactHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				reqHandler:     mockReq,
				contactHandler: mockContact,
			}

			mockContact.EXPECT().ContactGetsByExtension(gomock.Any(), tt.customerID, tt.expectExtension).Return(tt.responseContacts, nil)

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

func Test_processV1ContactsPut(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		extension  string
		request    *rabbitmqhandler.Request

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("883ce9bc-5707-11ee-b053-1ba4c0db8a30"),
			"test-extension",
			&rabbitmqhandler.Request{
				URI:      "/v1/contacts?customer_id=883ce9bc-5707-11ee-b053-1ba4c0db8a30&extension=test-extension",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockContact.EXPECT().ContactRefreshByEndpoint(gomock.Any(), tt.customerID, tt.extension).Return(nil)
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
