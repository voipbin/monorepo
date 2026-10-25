package requesthandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/models/sock"
)

const agentQueue = "bin-manager.agent-manager.request"

func Test_AgentV1PasswordWrappers(t *testing.T) {
	cases := []rpcCase{
		{
			name:           "forgot",
			queue:          agentQueue,
			expectURI:      "/v1/password-forgot",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"username":"agent-a@example.com"}`,
			responseData:   `{}`,
			call: func(h *requestHandler) (any, error) {
				return nil, h.AgentV1PasswordForgot(context.Background(), 3000, "agent-a@example.com")
			},
		},
		{
			name:           "reset",
			queue:          agentQueue,
			expectURI:      "/v1/password-reset",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"token":"reset-token-a","password":"synthetic-pass-a"}`,
			responseData:   `{}`,
			call: func(h *requestHandler) (any, error) {
				return nil, h.AgentV1PasswordReset(context.Background(), 3000, "reset-token-a", "synthetic-pass-a")
			},
		},
	}

	runRPCCases(t, cases)
}
