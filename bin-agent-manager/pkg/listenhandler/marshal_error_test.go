package listenhandler

import (
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
	cerrors "monorepo/bin-common-handler/models/errors"
)

// Test_processRequest_unserializableResponse covers the response paths whose
// handler result cannot be encoded as JSON. The agent carries a timestamp
// outside the year range that JSON can represent. Every path must answer 500
// with no body, never a partial or successful response.
func Test_processRequest_unserializableResponse(t *testing.T) {
	const (
		agentID    = "f1ba04b0-951e-11ee-a0a2-7b8600a1ee45"
		customerID = "5e4a0680-804e-11ec-8477-2fea5968d85b"
	)

	tests := []struct {
		name string

		request *sock.Request
		expect  func(m *agenthandler.MockAgentHandler, bad *agent.Agent)
	}{
		{
			name: "list",
			request: &sock.Request{
				URI:    "/v1/agents?page_size=10&page_token=",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().List(gomock.Any(), uint64(10), "", gomock.Any()).Return([]*agent.Agent{bad}, nil)
			},
		},
		{
			name: "create",
			request: &sock.Request{
				URI:    "/v1/agents",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"customer_id":"` + customerID + `","username":"user-a"}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Create(gomock.Any(), uuid.FromStringOrNil(customerID), "user-a", "", "", "", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name: "get",
			request: &sock.Request{
				URI:    "/v1/agents/" + agentID,
				Method: sock.RequestMethodGet,
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(agentID)).Return(bad, nil)
			},
		},
		{
			name: "delete",
			request: &sock.Request{
				URI:    "/v1/agents/" + agentID,
				Method: sock.RequestMethodDelete,
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(agentID)).Return(bad, nil)
			},
		},
		{
			name: "get by customer id and address",
			request: &sock.Request{
				URI:    "/v1/agents/get_by_customer_id_address",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"customer_id":"` + customerID + `","address":{"type":"extension","target":"1000"}}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().GetByCustomerIDAndAddress(gomock.Any(), uuid.FromStringOrNil(customerID), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name: "login by username",
			request: &sock.Request{
				URI:    "/v1/agents/user-a/login",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"password":"secret-a"}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Login(gomock.Any(), "user-a", "secret-a").Return(bad, nil)
			},
		},
		{
			name: "login",
			request: &sock.Request{
				URI:    "/v1/login",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"username":"user-a","password":"secret-a"}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Login(gomock.Any(), "user-a", "secret-a").Return(bad, nil)
			},
		},
		{
			name: "update addresses",
			request: &sock.Request{
				URI:    "/v1/agents/" + agentID + "/addresses",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"addresses":[]}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdateAddresses(gomock.Any(), uuid.FromStringOrNil(agentID), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name: "update basic info",
			request: &sock.Request{
				URI:    "/v1/agents/" + agentID,
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"name":"name-a"}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdateBasicInfo(gomock.Any(), uuid.FromStringOrNil(agentID), gomock.Any(), nil, nil).Return(bad, nil)
			},
		},
		{
			name: "update status",
			request: &sock.Request{
				URI:    "/v1/agents/" + agentID + "/status",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"status":"available"}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdateStatus(gomock.Any(), uuid.FromStringOrNil(agentID), agent.StatusAvailable).Return(bad, nil)
			},
		},
		{
			name: "update password",
			request: &sock.Request{
				URI:    "/v1/agents/" + agentID + "/password",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"password":"secret-a"}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdatePassword(gomock.Any(), uuid.FromStringOrNil(agentID), "secret-a").Return(bad, nil)
			},
		},
		{
			name: "update tag ids",
			request: &sock.Request{
				URI:    "/v1/agents/" + agentID + "/tag_ids",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"tag_ids":[]}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdateTagIDs(gomock.Any(), uuid.FromStringOrNil(agentID), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name: "update permission",
			request: &sock.Request{
				URI:    "/v1/agents/" + agentID + "/permission",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"permission":0}`),
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdatePermission(gomock.Any(), uuid.FromStringOrNil(agentID), agent.PermissionNone).Return(bad, nil)
			},
		},
		{
			name: "direct hash regenerate",
			request: &sock.Request{
				URI:    "/v1/agents/" + agentID + "/direct-hash-regenerate",
				Method: sock.RequestMethodPost,
			},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().DirectHashRegenerate(gomock.Any(), uuid.FromStringOrNil(agentID)).Return(bad, nil)
			},
		},
	}

	// year 10000 cannot be encoded as RFC 3339 by the standard JSON encoder.
	outOfRange := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

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

			bad := &agent.Agent{
				Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(agentID)},
				TMCreate: &outOfRange,
			}
			tt.expect(mockAgent, bad)

			tt.request.DataType = "application/json"
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 500 {
				t.Errorf("Wrong status. expect: 500, got: %d", res.StatusCode)
			}
			if len(res.Data) != 0 || res.DataType != "" {
				t.Errorf("Wrong body. expect an empty body, got: %s %s", res.DataType, res.Data)
			}
		})
	}
}

// Test_errorResponse_typedNil checks that a typed nil *VoipbinError, which
// matches errors.As but cannot be turned into a response, is still answered
// with a plain 500.
func Test_errorResponse_typedNil(t *testing.T) {
	var ve *cerrors.VoipbinError

	res := errorResponse(ve)
	if res == nil {
		t.Fatalf("Wrong match. expect a response, got nil")
	}
	if res.StatusCode != 500 {
		t.Errorf("Wrong status. expect: 500, got: %d", res.StatusCode)
	}
	if len(res.Data) != 0 {
		t.Errorf("Wrong body. expect an empty body, got: %s", res.Data)
	}
}
