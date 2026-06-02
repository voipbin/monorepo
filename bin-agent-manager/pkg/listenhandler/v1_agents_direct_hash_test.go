package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

func Test_processV1AgentsIDDirectHashRegenerate_notFound(t *testing.T) {
	tests := []struct {
		name      string
		agentID   uuid.UUID
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			name:    "not found returns 404",
			agentID: uuid.FromStringOrNil("b1c2d3e4-0001-0001-0001-000000000001"),
			request: &sock.Request{
				URI:    "/v1/agents/b1c2d3e4-0001-0001-0001-000000000001/direct-hash-regenerate",
				Method: sock.RequestMethodPost,
			},
			expectRes: &sock.Response{
				StatusCode: 404,
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

			mockAgent.EXPECT().DirectHashRegenerate(gomock.Any(), tt.agentID).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectRes.StatusCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}
