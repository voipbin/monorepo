package requesthandler

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_AstAMIRedirect(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		context  string
		exten    string
		priority string

		expectQueue  string
		expectURI    string
		expectMethod sock.RequestMethod
		expectData   []byte
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"6b79ae28-e3f1-11ea-bf62-7f539c6300fc",

			"svc-echo",
			"s",
			"1",

			"asterisk.00:11:22:33:44:55.request",
			"/ami/action",
			sock.RequestMethodPost,
			[]byte(`{"Action":"Redirect","Channel":"6b79ae28-e3f1-11ea-bf62-7f539c6300fc","Context":"svc-echo","Exten":"s","Priority":"1"}`),
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

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				&sock.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: "application/json",
					Data:     tt.expectData,
				},
			).Return(&sock.Response{StatusCode: 200, Data: []byte(`{"Response":"Success","Message":"Redirect successful"}`)}, nil)

			err := reqHandler.AstAMIRedirect(context.Background(), tt.asteriskID, tt.channelID, tt.context, tt.exten, tt.priority)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AstAMIRedirect_response(t *testing.T) {

	tests := []struct {
		name string

		response *sock.Response

		expectErr       bool
		expectErrSubstr string
	}{
		{
			name:     "success",
			response: &sock.Response{StatusCode: 200, Data: []byte(`{"Response":"Success","Message":"Redirect successful"}`)},
		},
		{
			name:            "not found without body",
			response:        &sock.Response{StatusCode: 404},
			expectErr:       true,
			expectErrSubstr: "status code: 404",
		},
		{
			name:            "bad request without body",
			response:        &sock.Response{StatusCode: 400},
			expectErr:       true,
			expectErrSubstr: "status code: 400",
		},
		{
			name:            "zero status code",
			response:        &sock.Response{StatusCode: 0},
			expectErr:       true,
			expectErrSubstr: "status code: 0",
		},
		{
			name:            "nil response",
			response:        nil,
			expectErr:       true,
			expectErrSubstr: "status code: 0",
		},
		{
			name:            "ami error response",
			response:        &sock.Response{StatusCode: 200, Data: []byte(`{"Response":"Error","Message":"No such channel"}`)},
			expectErr:       true,
			expectErrSubstr: "response: Error",
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "asterisk.00:11:22:33:44:55.request", gomock.Any()).Return(tt.response, nil)

			err := reqHandler.AstAMIRedirect(context.Background(), "00:11:22:33:44:55", "6b79ae28-e3f1-11ea-bf62-7f539c6300fc", "svc-stasis", "s", "1")
			if !tt.expectErr {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if !strings.Contains(err.Error(), tt.expectErrSubstr) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectErrSubstr, err)
			}
		})
	}
}
