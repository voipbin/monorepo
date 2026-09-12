package listenhandler

import (
	stderrors "errors"
	"testing"

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
