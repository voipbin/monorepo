package listenhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_processV1AgentsCountByCustomerGet_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1d2e3f4-0001-0001-0001-000000000001")

	tests := []struct {
		name string

		data     []byte
		mockFunc func(m *agenthandler.MockAgentHandler)

		expectCode int
	}{
		{
			name: "invalid json body",

			data:     []byte(`{"customer_id":`),
			mockFunc: func(m *agenthandler.MockAgentHandler) {},

			expectCode: 400,
		},
		{
			name: "handler failure is an internal error",

			data: []byte(`{"customer_id":"c1d2e3f4-0001-0001-0001-000000000001"}`),
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().CountByCustomerID(gomock.Any(), customerID).Return(0, stderrors.New("boom"))
			},

			expectCode: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &listenHandler{
				sockHandler:  sockhandler.NewMockSockHandler(mc),
				agentHandler: mockAgent,
			}

			tt.mockFunc(mockAgent)

			res, err := h.processV1AgentsCountByCustomerGet(context.Background(), &sock.Request{
				URI:    "/v1/agents/count_by_customer",
				Method: sock.RequestMethodGet,
				Data:   tt.data,
			})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectCode, res.StatusCode)
			}
		})
	}
}

func Test_processV1Login_errors(t *testing.T) {
	tests := []struct {
		name string

		uri      string
		data     []byte
		mockFunc func(m *agenthandler.MockAgentHandler)

		expectCode int
	}{
		{
			name: "uri without path separator",

			uri:      "login",
			data:     []byte(`{}`),
			mockFunc: func(m *agenthandler.MockAgentHandler) {},

			expectCode: 400,
		},
		{
			name: "invalid json body",

			uri:      "/v1/login",
			data:     []byte(`{"username":`),
			mockFunc: func(m *agenthandler.MockAgentHandler) {},

			expectCode: 400,
		},
		{
			name: "login failure",

			uri:  "/v1/login",
			data: []byte(`{"username":"login@example.com","password":"password-for-test"}`),
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().Login(gomock.Any(), "login@example.com", "password-for-test").Return(nil, stderrors.New("wrong password"))
			},

			expectCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &listenHandler{
				sockHandler:  sockhandler.NewMockSockHandler(mc),
				agentHandler: mockAgent,
			}

			tt.mockFunc(mockAgent)

			res, err := h.processV1Login(context.Background(), &sock.Request{
				URI:    tt.uri,
				Method: sock.RequestMethodPost,
				Data:   tt.data,
			})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectCode, res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: empty body, got: %s", res.Data)
			}
		})
	}
}

func Test_processV1AgentsIDDirectHashRegenerate_cases(t *testing.T) {
	agentID := uuid.FromStringOrNil("b1c2d3e4-0001-0001-0001-000000000001")

	t.Run("uri with too few items", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &listenHandler{
			sockHandler:  sockhandler.NewMockSockHandler(mc),
			agentHandler: agenthandler.NewMockAgentHandler(mc),
		}

		res, err := h.processV1AgentsIDDirectHashRegenerate(context.Background(), &sock.Request{
			URI:    "/v1/agents",
			Method: sock.RequestMethodPost,
		})
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if res.StatusCode != 400 {
			t.Errorf("Wrong status code. expect: 400, got: %d", res.StatusCode)
		}
	})

	t.Run("success returns the regenerated agent", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAgent := agenthandler.NewMockAgentHandler(mc)
		h := &listenHandler{
			sockHandler:  sockhandler.NewMockSockHandler(mc),
			agentHandler: mockAgent,
		}

		mockAgent.EXPECT().DirectHashRegenerate(gomock.Any(), agentID).Return(newTestAgent(agentID), nil)

		res, err := h.processV1AgentsIDDirectHashRegenerate(context.Background(), &sock.Request{
			URI:    "/v1/agents/b1c2d3e4-0001-0001-0001-000000000001/direct-hash-regenerate",
			Method: sock.RequestMethodPost,
		})
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("Wrong status code. expect: 200, got: %d", res.StatusCode)
		}
		if res.DataType != "application/json" {
			t.Errorf("Wrong data type. expect: application/json, got: %s", res.DataType)
		}

		var got struct {
			ID       uuid.UUID `json:"id"`
			Username string    `json:"username"`
		}
		if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil {
			t.Fatalf("Could not unmarshal the response. err: %v", errUnmarshal)
		}
		if got.ID != agentID || got.Username != "direct@example.com" {
			t.Errorf("Wrong body. expect id: %s, username: direct@example.com, got: %s", agentID, res.Data)
		}
	})
}
