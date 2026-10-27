package mcpserverhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

const testCoverageKeys = "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

func Test_Create_validationErrors(t *testing.T) {
	customerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name string

		url      string
		status   mcpserver.Status
		authType mcpserver.AuthType

		expectReason string
	}{
		{name: "plain http url", url: "http://mcp.example.com/", status: mcpserver.StatusActive, expectReason: "INVALID_MCP_SERVER_URL"},
		{name: "unknown status", url: "https://mcp.example.com/", status: mcpserver.Status("paused"), expectReason: "INVALID_MCP_SERVER_STATUS"},
		{name: "unknown auth type", url: "https://mcp.example.com/", status: mcpserver.StatusActive, authType: mcpserver.AuthType("basic"), expectReason: "INVALID_MCP_SERVER_AUTH_TYPE"},
		{name: "oauth cannot be set directly", url: "https://mcp.example.com/", status: mcpserver.StatusActive, authType: mcpserver.AuthTypeOAuth, expectReason: "INVALID_MCP_SERVER_AUTH_TYPE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, mockNotify)

			res, err := h.Create(context.Background(), customerID, "n", "d", tt.url, tt.status, tt.authType, "", "")
			if res != nil {
				t.Errorf("expected nil result, got %v", res)
			}
			assertInvalidArgument(t, err, tt.expectReason)
		})
	}
}

func Test_Create_defaultsStatusToActive(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newTestHandlerWithCrypto(t, mockDB, mockNotify)
	h.utilHandler = mockUtil

	customerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	newID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	mockUtil.EXPECT().UUIDCreate().Return(newID)

	var created *mcpserver.McpServer
	mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, m *mcpserver.McpServer) error {
		created = m
		return nil
	})
	mockDB.EXPECT().McpServerGet(gomock.Any(), newID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*mcpserver.McpServer, error) {
		return created, nil
	})
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, mcpserver.EventTypeCreated, gomock.Any())

	res, err := h.Create(context.Background(), customerID, "name1", "detail1", "https://mcp.example.com/", "", mcpserver.AuthTypeAPIKey, "X-Key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != mcpserver.StatusActive {
		t.Errorf("Status = %q, expect: %q", res.Status, mcpserver.StatusActive)
	}
	if res.ID != newID || res.CustomerID != customerID || res.Name != "name1" || res.Detail != "detail1" || res.APIKeyHeader != "X-Key" || res.AuthType != mcpserver.AuthTypeAPIKey {
		t.Errorf("unexpected stored row: %+v", res)
	}
}

func Test_Create_dependencyErrors(t *testing.T) {
	customerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	newID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	errBoom := errors.New("boom")

	tests := []struct {
		name string

		secret    string
		noKeys    bool
		createErr error
		getErr    error
		// which db calls are expected
		expectCreate bool
		expectGet    bool

		expectMsg string
	}{
		{name: "encrypt fails without keys", secret: "s", noKeys: true, expectMsg: "could not encrypt secret"},
		{name: "db create fails", createErr: errBoom, expectCreate: true, expectMsg: "could not create mcp server"},
		{name: "db get fails", getErr: errBoom, expectCreate: true, expectGet: true, expectMsg: "could not get created mcp server"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, mockNotify)
			h.utilHandler = mockUtil
			if tt.noKeys {
				h.crypto, _ = NewSecretCrypto("")
			}

			mockUtil.EXPECT().UUIDCreate().Return(newID)
			if tt.expectCreate {
				mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).Return(tt.createErr)
			}
			if tt.expectGet {
				mockDB.EXPECT().McpServerGet(gomock.Any(), newID).Return(nil, tt.getErr)
			}

			res, err := h.Create(context.Background(), customerID, "n", "d", "https://mcp.example.com/", mcpserver.StatusActive, mcpserver.AuthTypeBearer, "", tt.secret)
			if res != nil {
				t.Errorf("expected nil result, got %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("error = %v, expect to contain %q", err, tt.expectMsg)
			}
		})
	}
}

func Test_Get(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	errBoom := errors.New("boom")
	row := &mcpserver.McpServer{Identity: identityFor(id, uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")), Name: "n1"}

	tests := []struct {
		name string

		dbRes *mcpserver.McpServer
		dbErr error

		expectRes      *mcpserver.McpServer
		expectNotFound bool
		expectMsg      string
	}{
		{name: "found", dbRes: row, expectRes: row},
		{name: "not found maps to NotFound", dbErr: dbhandler.ErrNotFound, expectNotFound: true},
		{name: "wrapped not found maps to NotFound", dbErr: errors.Join(errBoom, dbhandler.ErrNotFound), expectNotFound: true},
		{name: "other error is wrapped", dbErr: errBoom, expectMsg: "could not get mcp server"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, notifyhandler.NewMockNotifyHandler(mc))

			mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(tt.dbRes, tt.dbErr)

			res, err := h.Get(context.Background(), id)
			if res != tt.expectRes {
				t.Errorf("result = %v, expect: %v", res, tt.expectRes)
			}
			switch {
			case tt.expectNotFound:
				if !isNotFound(err) {
					t.Errorf("expected NotFound error, got %v", err)
				}
			case tt.expectMsg != "":
				if err == nil || !strings.Contains(err.Error(), tt.expectMsg) || isNotFound(err) {
					t.Errorf("error = %v, expect plain error containing %q", err, tt.expectMsg)
				}
			default:
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func Test_List(t *testing.T) {
	errBoom := errors.New("boom")
	filters := map[mcpserver.Field]any{mcpserver.FieldCustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")}
	rows := []*mcpserver.McpServer{{Name: "a"}, {Name: "b"}}

	tests := []struct {
		name string

		dbRes []*mcpserver.McpServer
		dbErr error

		expectLen int
		expectErr bool
	}{
		{name: "two rows", dbRes: rows, expectLen: 2},
		{name: "empty", dbRes: []*mcpserver.McpServer{}, expectLen: 0},
		{name: "db error", dbErr: errBoom, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, notifyhandler.NewMockNotifyHandler(mc))

			mockDB.EXPECT().McpServerList(gomock.Any(), uint64(25), "2026-01-01T00:00:00Z", filters).Return(tt.dbRes, tt.dbErr)

			res, err := h.List(context.Background(), 25, "2026-01-01T00:00:00Z", filters)
			if tt.expectErr {
				if err == nil || !errors.Is(err, errBoom) || !strings.Contains(err.Error(), "could not list mcp servers") {
					t.Errorf("error = %v", err)
				}
				if res != nil {
					t.Errorf("expected nil result, got %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(res) != tt.expectLen {
				t.Errorf("len = %d, expect: %d", len(res), tt.expectLen)
			}
		})
	}
}

func Test_Update_getLiveErrors(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name  string
		dbErr error

		expectNotFound bool
	}{
		{name: "row missing", dbErr: dbhandler.ErrNotFound, expectNotFound: true},
		{name: "db failure", dbErr: errors.New("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, notifyhandler.NewMockNotifyHandler(mc))
			mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(nil, tt.dbErr)

			res, err := h.Update(context.Background(), id, strPtr("x"), nil, nil, nil, nil, nil, nil)
			if res != nil {
				t.Errorf("expected nil result, got %v", res)
			}
			if isNotFound(err) != tt.expectNotFound {
				t.Errorf("isNotFound = %v, expect: %v (err: %v)", isNotFound(err), tt.expectNotFound, err)
			}
		})
	}
}

func Test_Update_validationErrors(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	customerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	badStatus := mcpserver.Status("paused")
	badAuth := mcpserver.AuthType("basic")

	tests := []struct {
		name string

		url      *string
		status   *mcpserver.Status
		authType *mcpserver.AuthType

		expectReason string
	}{
		{name: "http url", url: strPtr("http://mcp.example.com/"), expectReason: "INVALID_MCP_SERVER_URL"},
		{name: "loopback literal url", url: strPtr("https://127.0.0.1/"), expectReason: "INVALID_MCP_SERVER_URL"},
		{name: "unknown status", status: &badStatus, expectReason: "INVALID_MCP_SERVER_STATUS"},
		{name: "unknown auth type", authType: &badAuth, expectReason: "INVALID_MCP_SERVER_AUTH_TYPE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, notifyhandler.NewMockNotifyHandler(mc))
			mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(&mcpserver.McpServer{Identity: identityFor(id, customerID), URL: "https://mcp.example.com/"}, nil)

			res, err := h.Update(context.Background(), id, nil, nil, tt.url, tt.status, tt.authType, nil, nil)
			if res != nil {
				t.Errorf("expected nil result, got %v", res)
			}
			assertInvalidArgument(t, err, tt.expectReason)
		})
	}
}

func Test_Update_dependencyErrors(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	customerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	errBoom := errors.New("boom")
	live := &mcpserver.McpServer{Identity: identityFor(id, customerID), URL: "https://mcp.example.com/"}

	tests := []struct {
		name string

		secret    *string
		noKeys    bool
		updateErr error
		getErr    error

		expectUpdate bool
		expectGet    bool
		expectMsg    string
	}{
		{name: "encrypt fails without keys", secret: strPtr("s"), noKeys: true, expectMsg: "could not encrypt secret"},
		{name: "db update fails", updateErr: errBoom, expectUpdate: true, expectMsg: "could not update mcp server"},
		{name: "read back fails", getErr: errBoom, expectUpdate: true, expectGet: true, expectMsg: "could not get updated mcp server"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := newTestHandlerWithCrypto(t, mockDB, notifyhandler.NewMockNotifyHandler(mc))
			if tt.noKeys {
				h.crypto, _ = NewSecretCrypto("")
			}

			// first read is the existence gate
			mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(live, nil)
			if tt.expectUpdate {
				mockDB.EXPECT().McpServerUpdate(gomock.Any(), id, map[mcpserver.Field]any{mcpserver.FieldName: "renamed"}).Return(tt.updateErr)
			}
			if tt.expectGet {
				mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(nil, tt.getErr)
			}

			res, err := h.Update(context.Background(), id, strPtr("renamed"), nil, nil, nil, nil, nil, tt.secret)
			if res != nil {
				t.Errorf("expected nil result, got %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("error = %v, expect to contain %q", err, tt.expectMsg)
			}
		})
	}
}

func Test_Update_emptyPutReturnsCurrentRow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	live := &mcpserver.McpServer{Identity: identityFor(id, uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")), Name: "current"}

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandlerWithCrypto(t, mockDB, notifyhandler.NewMockNotifyHandler(mc))
	// gate read plus the read that returns the current state; no update, no event
	mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(live, nil).Times(2)

	res, err := h.Update(context.Background(), id, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != live {
		t.Errorf("expected the current row, got %v", res)
	}
}

func Test_NewMcpServerHandler(t *testing.T) {
	tests := []struct {
		name string
		keys string

		expectErr bool
	}{
		{name: "valid key", keys: testCoverageKeys},
		{name: "empty keys are accepted", keys: ""},
		{name: "malformed keys are refused", keys: "not-a-key", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res, err := NewMcpServerHandler(notifyhandler.NewMockNotifyHandler(mc), dbhandler.NewMockDBHandler(mc), tt.keys)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Errorf("expected error and nil handler, got %v / %v", res, err)
				}
				return
			}
			if err != nil || res == nil {
				t.Fatalf("unexpected result: %v / %v", res, err)
			}
			impl, ok := res.(*mcpServerHandler)
			if !ok || impl.crypto == nil || impl.utilHandler == nil {
				t.Errorf("handler not fully wired: %#v", res)
			}
		})
	}
}
