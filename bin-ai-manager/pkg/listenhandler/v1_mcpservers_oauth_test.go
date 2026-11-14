package listenhandler

import (
	stderrors "errors"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/mcpoauthhandler"
)

// Test_processV1McpServersOAuthStartPost pins wiring end to end: the
// switch dispatches POST /v1/mcp_servers/oauth/start to
// processV1McpServersOAuthStartPost, which unmarshals the request, calls
// mcpOAuthHandler.Start with the unwrapped fields, and marshals the
// authorize_url/link_token response.
func Test_processV1McpServersOAuthStartPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockOAuth := mcpoauthhandler.NewMockMcpOAuthHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		mcpOAuthHandler: mockOAuth,
	}

	customerID := uuid.FromStringOrNil("58e7502c-a770-11ed-9b86-7fabe2dba847")

	req := &sock.Request{
		URI:      "/v1/mcp_servers/oauth/start",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"customer_id":"58e7502c-a770-11ed-9b86-7fabe2dba847","vendor":"github"}`),
	}

	mockOAuth.EXPECT().Start(gomock.Any(), customerID, "github", nil).Return("https://github.com/login/oauth/authorize?...", "state-token-abc", nil)

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.StatusCode != 200 {
		t.Fatalf("expected 200, got: %v", res)
	}
	want := `{"authorize_url":"https://github.com/login/oauth/authorize?...","link_token":"state-token-abc"}`
	if string(res.Data) != want {
		t.Errorf("expected %s, got %s", want, res.Data)
	}
}

func Test_processV1McpServersOAuthStartPost_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockOAuth := mcpoauthhandler.NewMockMcpOAuthHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		mcpOAuthHandler: mockOAuth,
	}

	req := &sock.Request{
		URI:      "/v1/mcp_servers/oauth/start",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"customer_id":"58e7502c-a770-11ed-9b86-7fabe2dba847","vendor":"unknown-vendor"}`),
	}

	mockOAuth.EXPECT().Start(gomock.Any(), gomock.Any(), "unknown-vendor", nil).Return("", "", stderrors.New("unknown oauth vendor"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.StatusCode < 400 {
		t.Errorf("expected an error status code, got: %v", res)
	}
}

// Test_processV1McpServersOAuthCallbackGet pins that the public callback
// relay's existence check is dispatched correctly and never leaks more
// than a boolean.
func Test_processV1McpServersOAuthCallbackGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockOAuth := mcpoauthhandler.NewMockMcpOAuthHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		mcpOAuthHandler: mockOAuth,
	}

	req := &sock.Request{
		URI:    "/v1/mcp_servers/oauth/callback?state=abc123",
		Method: sock.RequestMethodGet,
	}

	mockOAuth.EXPECT().CallbackExists(gomock.Any(), "abc123").Return(true, nil)

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.StatusCode != 200 {
		t.Fatalf("expected 200, got: %v", res)
	}
	want := `{"exists":true}`
	if string(res.Data) != want {
		t.Errorf("expected %s, got %s", want, res.Data)
	}
}

// Test_processV1McpServersOAuthCompletePost pins that the complete
// handler unmarshals the request, calls mcpOAuthHandler.Complete, and
// marshals the resulting McpServer directly (style A).
func Test_processV1McpServersOAuthCompletePost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockOAuth := mcpoauthhandler.NewMockMcpOAuthHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		mcpOAuthHandler: mockOAuth,
	}

	customerID := uuid.FromStringOrNil("58e7502c-a770-11ed-9b86-7fabe2dba847")

	req := &sock.Request{
		URI:      "/v1/mcp_servers/oauth/complete",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"customer_id":"58e7502c-a770-11ed-9b86-7fabe2dba847","state":"abc123","code":"auth-code"}`),
	}

	mockOAuth.EXPECT().Complete(gomock.Any(), customerID, "abc123", "auth-code").Return(&mcpserver.McpServer{}, nil)

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.StatusCode != 200 {
		t.Fatalf("expected 200, got: %v", res)
	}
}

func Test_processV1McpServersOAuthCompletePost_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockOAuth := mcpoauthhandler.NewMockMcpOAuthHandler(mc)

	h := &listenHandler{
		sockHandler:     mockSock,
		mcpOAuthHandler: mockOAuth,
	}

	req := &sock.Request{
		URI:      "/v1/mcp_servers/oauth/complete",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{invalid`),
	}

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.StatusCode != 400 {
		t.Errorf("expected 400, got: %v", res)
	}
}

// A vendor whose app is not stored reaches the caller as a typed internal error: status 500, the
// voipbin error data type and the reason MCP_OAUTH_VENDOR_NOT_CONFIGURED, for the start and the
// completion of the flow alike.
func Test_processV1McpServersOAuth_notConfigured(t *testing.T) {
	tests := []struct {
		name  string
		uri   string
		data  string
		setup func(m *mcpoauthhandler.MockMcpOAuthHandler, err error)
	}{
		{
			name: "start",
			uri:  "/v1/mcp_servers/oauth/start",
			data: `{"customer_id":"58e7502c-a770-11ed-9b86-7fabe2dba847","vendor":"github"}`,
			setup: func(m *mcpoauthhandler.MockMcpOAuthHandler, err error) {
				m.EXPECT().Start(gomock.Any(), gomock.Any(), "github", nil).Return("", "", err)
			},
		},
		{
			name: "complete",
			uri:  "/v1/mcp_servers/oauth/complete",
			data: `{"customer_id":"58e7502c-a770-11ed-9b86-7fabe2dba847","state":"s1","code":"c1"}`,
			setup: func(m *mcpoauthhandler.MockMcpOAuthHandler, err error) {
				m.EXPECT().Complete(gomock.Any(), gomock.Any(), "s1", "c1").Return(nil, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockOAuth := mcpoauthhandler.NewMockMcpOAuthHandler(mc)
			h := &listenHandler{
				sockHandler:     sockhandler.NewMockSockHandler(mc),
				mcpOAuthHandler: mockOAuth,
			}

			tt.setup(mockOAuth, cerrors.Internal(commonoutline.ServiceNameAIManager, "MCP_OAUTH_VENDOR_NOT_CONFIGURED", "the oauth app of the vendor is not configured"))

			res, err := h.processRequest(&sock.Request{
				URI:      tt.uri,
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(tt.data),
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusInternalServerError || res.DataType != cerrors.DataTypeVoipbinError {
				t.Fatalf("Wrong match. expect: 500 with the voipbin error data type, got: %+v", res)
			}

			ve := cerrors.FromResponse(res)
			if ve == nil || ve.Reason != "MCP_OAUTH_VENDOR_NOT_CONFIGURED" || ve.Status != cerrors.StatusInternal {
				t.Errorf("Wrong match. expect: internal MCP_OAUTH_VENDOR_NOT_CONFIGURED, got: %+v", ve)
			}
		})
	}
}
