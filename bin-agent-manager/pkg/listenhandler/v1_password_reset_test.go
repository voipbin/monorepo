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

func Test_ProcessV1PasswordResetPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/password-reset",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"token":"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890","password":"newpassword123"}`),
			},

			expectRes: &sock.Response{
				StatusCode: 200,
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

			mockAgent.EXPECT().PasswordReset(gomock.Any(), "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "newpassword123").Return(nil)

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

func Test_ProcessV1PasswordResetPost_ResetFailed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		sockHandler:  mockSock,
		agentHandler: mockAgent,
	}

	req := &sock.Request{
		URI:      "/v1/password-reset",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"token":"invalidtoken","password":"newpassword123"}`),
	}

	mockAgent.EXPECT().PasswordReset(gomock.Any(), "invalidtoken", "newpassword123").Return(fmt.Errorf("invalid or expired token"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != 400 {
		t.Errorf("Wrong status code. expect: 400, got: %d", res.StatusCode)
	}
}

func Test_ProcessV1PasswordResetPost_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	h := &listenHandler{
		sockHandler:  mockSock,
		agentHandler: mockAgent,
	}

	req := &sock.Request{
		URI:      "/v1/password-reset",
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
