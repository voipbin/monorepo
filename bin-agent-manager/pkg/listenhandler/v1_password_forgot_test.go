package listenhandler

import (
	"fmt"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_ProcessV1PasswordForgotPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseToken    string
		responseUsername string
		responseErr      error

		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/password-forgot",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"username":"test@voipbin.net"}`),
			},

			responseToken:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			responseUsername: "test@voipbin.net",
			responseErr:      nil,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"token":"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890","username":"test@voipbin.net"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				agentHandler: mockAgent,
			}

			mockAgent.EXPECT().PasswordForgot(gomock.Any(), "test@voipbin.net").Return(tt.responseToken, tt.responseUsername, tt.responseErr)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ProcessV1PasswordForgotPost_AgentNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		sockHandler:  mockSock,
		agentHandler: mockAgent,
	}

	req := &sock.Request{
		URI:      "/v1/password-forgot",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"username":"unknown@voipbin.net"}`),
	}

	mockAgent.EXPECT().PasswordForgot(gomock.Any(), "unknown@voipbin.net").Return("", "", fmt.Errorf("agent not found"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 404 {
		t.Errorf("Wrong status code. expect: 404, got: %d", res.StatusCode)
	}
}

func Test_ProcessV1PasswordForgotPost_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		sockHandler:  mockSock,
		agentHandler: mockAgent,
	}

	req := &sock.Request{
		URI:      "/v1/password-forgot",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`invalid json`),
	}

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 400 {
		t.Errorf("Wrong status code. expect: 400, got: %d", res.StatusCode)
	}
}
