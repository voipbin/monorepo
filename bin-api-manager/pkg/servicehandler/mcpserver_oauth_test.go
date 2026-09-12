package servicehandler

import (
	"context"
	stderrors "errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	ammcpserver "monorepo/bin-ai-manager/models/mcpserver"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_McpOAuthStart_DirectAccessNotSupported(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}

	direct := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID: uuid.Must(uuid.NewV4()),
	})

	_, _, err := h.McpOAuthStart(context.Background(), direct, "github", nil)
	if err != serviceerrors.ErrDirectAccessNotSupported {
		t.Errorf("expected ErrDirectAccessNotSupported, got: %v", err)
	}
}

func Test_McpOAuthStart_PermissionDenied(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}

	agentNoPermission := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Permission: amagent.PermissionNone,
	})

	_, _, err := h.McpOAuthStart(context.Background(), agentNoPermission, "github", nil)
	if err != serviceerrors.ErrPermissionDenied {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

// Test_McpOAuthStart_PassesCallerCustomerID pins that the servicehandler
// passes the AUTHENTICATED caller's customer_id to the RPC, never a
// customer_id supplied by the request body -- the actual IDOR-prevention
// boundary for the create-new-server path (the reconnect-by-id path's
// ownership check lives at the ai-manager layer, design §9).
func Test_McpOAuthStart_PassesCallerCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}

	customerID := uuid.Must(uuid.NewV4())
	agentAdmin := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().AIV1McpOAuthStart(gomock.Any(), customerID, "github", (*uuid.UUID)(nil)).Return("https://authorize.example.com", "token123", nil)

	authorizeURL, linkToken, err := h.McpOAuthStart(context.Background(), agentAdmin, "github", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authorizeURL != "https://authorize.example.com" || linkToken != "token123" {
		t.Errorf("unexpected result: %s, %s", authorizeURL, linkToken)
	}
}

// Test_McpOAuthCallback_FailsClosedOnRPCError pins that a downstream RPC
// error is surfaced as an error (the server layer above this maps it to
// the same generic invalid_state redirect design §7a requires, never a
// 200 exists:true).
func Test_McpOAuthCallback_FailsClosedOnRPCError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}

	mockReq.EXPECT().AIV1McpOAuthCallback(gomock.Any(), "some-state").Return(false, stderrors.New("rpc failure"))

	exists, err := h.McpOAuthCallback(context.Background(), "some-state")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if exists {
		t.Error("expected exists=false on error")
	}
}

func Test_McpOAuthCallback_Passthrough(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}

	mockReq.EXPECT().AIV1McpOAuthCallback(gomock.Any(), "some-state").Return(true, nil)

	exists, err := h.McpOAuthCallback(context.Background(), "some-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected exists=true")
	}
}

func Test_McpOAuthComplete_DirectAccessNotSupported(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}

	direct := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID: uuid.Must(uuid.NewV4()),
	})

	_, err := h.McpOAuthComplete(context.Background(), direct, "state", "code")
	if err != serviceerrors.ErrDirectAccessNotSupported {
		t.Errorf("expected ErrDirectAccessNotSupported, got: %v", err)
	}
}

func Test_McpOAuthComplete_PermissionDenied(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}

	agentNoPermission := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Permission: amagent.PermissionNone,
	})

	_, err := h.McpOAuthComplete(context.Background(), agentNoPermission, "state", "code")
	if err != serviceerrors.ErrPermissionDenied {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

// Test_McpOAuthComplete_PassesCallerCustomerID pins that the caller's
// authenticated customer_id (not any client-supplied value) is what
// reaches the RPC -- the real security boundary lives at the ai-manager
// layer (design §7a), but this layer must not let a caller smuggle a
// different customer_id in.
func Test_McpOAuthComplete_PassesCallerCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}

	customerID := uuid.Must(uuid.NewV4())
	agentAdmin := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().AIV1McpOAuthComplete(gomock.Any(), customerID, "state123", "code456").Return(&ammcpserver.McpServer{}, nil)

	res, err := h.McpOAuthComplete(context.Background(), agentAdmin, "state123", "code456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected a non-nil result")
	}
}
