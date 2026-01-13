package listenhandler

import (
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-email-manager/pkg/emailhandler"
	reflect "reflect"
	"testing"

	gomock "go.uber.org/mock/gomock"
)

func Test_processV1HooksPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		expectReceivedURI  string
		expectReceivedData []byte
		expectRes          *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/hooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"https://hook.voipbin.net/v1.0/conversation/customers/a92e60ea-e85b-11ec-a173-0b1cf8c9d3e9/line","received_data":"eyJkZXN0aW5hdGlvbiI6IlUxMTI5ODIxNDExNmUzYWZiYWQ0MzJiNTc5NGE2ZDNhMCIsImV2ZW50cyI6W119"}`),
			},

			expectReceivedURI:  "https://hook.voipbin.net/v1.0/conversation/customers/a92e60ea-e85b-11ec-a173-0b1cf8c9d3e9/line",
			expectReceivedData: []byte(`{"destination":"U11298214116e3afbad432b5794a6d3a0","events":[]}`),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				emailHandler: mockEmail,
			}

			mockEmail.EXPECT().Hook(gomock.Any(), tt.expectReceivedURI, tt.expectReceivedData.Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
