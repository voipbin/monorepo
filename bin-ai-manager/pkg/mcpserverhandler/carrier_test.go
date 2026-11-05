package mcpserverhandler

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func carrierTestTable() mcpserver.CarrierTable {
	return mcpserver.CarrierTable{
		"/fake":  {Carrier: "fake", Credential: mcpserver.CredentialClientIDSecret},
		"/fake2": {Carrier: "fake2", Credential: mcpserver.CredentialAPIKey},
	}
}

func newCarrierTestHandler(t *testing.T, db *dbhandler.MockDBHandler, notify *notifyhandler.MockNotifyHandler, table mcpserver.CarrierTable) *mcpServerHandler {
	t.Helper()
	h := newTestHandlerWithCrypto(t, db, notify)
	h.carrierTable = table
	return h
}

func assertReasonAndMessage(t *testing.T, err error, reason string) *cerrors.VoipbinError {
	t.Helper()
	assertInvalidArgument(t, err, reason)
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("expected a VoipbinError, got %v", err)
	}
	return ve
}

func Test_Create_carrierURL(t *testing.T) {
	customerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name  string
		table mcpserver.CarrierTable
		url   string

		expectReason  string
		expectMessage string
	}{
		{"unknown path on the reserved host", carrierTestTable(), "https://carrier.invalid/unknown", "INVALID_MCP_SERVER_URL", "url must be one of: https://carrier.invalid/fake, https://carrier.invalid/fake2"},
		{"query on the reserved url", carrierTestTable(), "https://carrier.invalid/fake?x=1", "INVALID_MCP_SERVER_URL", "url must be one of:"},
		{"port on the reserved url", carrierTestTable(), "https://carrier.invalid:443/fake", "INVALID_MCP_SERVER_URL", "url must be one of:"},
		{"upper case reserved host with a bad path", carrierTestTable(), "https://CARRIER.INVALID/x", "INVALID_MCP_SERVER_URL", "url must be one of:"},
		{"ideographic stop in the reserved host", carrierTestTable(), "https://carrier\u3002invalid/x", "INVALID_MCP_SERVER_URL", "url must be one of:"},
		{"empty production table refuses every reserved url", mcpserver.DefaultCarrierTable(), "https://carrier.invalid/ups", "INVALID_MCP_SERVER_URL", "no carrier integration is available"},
		{"empty production table, trailing dots", mcpserver.DefaultCarrierTable(), "https://carrier.invalid../fake", "INVALID_MCP_SERVER_URL", "no carrier integration is available"},
		{"nil table refuses every reserved url", nil, "https://carrier.invalid/fake", "INVALID_MCP_SERVER_URL", "no carrier integration is available"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := newCarrierTestHandler(t, dbhandler.NewMockDBHandler(mc), notifyhandler.NewMockNotifyHandler(mc), tt.table)

			res, err := h.Create(context.Background(), customerID, "n", "d", tt.url, mcpserver.StatusActive, mcpserver.AuthTypeBearer, "", "id1:sec1")
			if res != nil {
				t.Errorf("expected nil result, got %v", res)
			}
			ve := assertReasonAndMessage(t, err, tt.expectReason)
			if !strings.Contains(ve.Message, tt.expectMessage) {
				t.Errorf("message %q does not contain %q", ve.Message, tt.expectMessage)
			}
			if strings.Contains(ve.Message, "sec1") || strings.Contains(ve.Message, "/unknown") {
				t.Errorf("the message must not repeat the input: %q", ve.Message)
			}
		})
	}
}

func Test_Create_otherInvalidHostsAreStillRegistered(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newCarrierTestHandler(t, mockDB, mockNotify, mcpserver.DefaultCarrierTable())

	mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().McpServerGet(gomock.Any(), gomock.Any()).Return(&mcpserver.McpServer{}, nil)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), mcpserver.EventTypeCreated, gomock.Any())

	if _, err := h.Create(context.Background(), uuid.Must(uuid.NewV4()), "n", "d", "https://example.invalid/x", mcpserver.StatusActive, mcpserver.AuthTypeNone, "", ""); err != nil {
		t.Fatalf("a .invalid host other than the reserved one must still register: %v", err)
	}
}

func Test_Create_carrierSecret(t *testing.T) {
	customerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name     string
		url      string
		authType mcpserver.AuthType
		secret   string

		expectReject bool
	}{
		{"client id and secret accepted", "https://carrier.invalid/fake", mcpserver.AuthTypeBearer, "id1:sec1", false},
		{"missing colon rejected", "https://carrier.invalid/fake", mcpserver.AuthTypeBearer, "sec1", true},
		{"empty id rejected", "https://carrier.invalid/fake", mcpserver.AuthTypeBearer, ":sec1", true},
		{"api key accepted", "https://carrier.invalid/fake2", mcpserver.AuthTypeBearer, "key1", false},
		{"api key with inner space rejected", "https://carrier.invalid/fake2", mcpserver.AuthTypeAPIKey, "k 1", true},
		{"no secret in the request is not checked", "https://carrier.invalid/fake", mcpserver.AuthTypeBearer, "", false},
		{"auth none is not checked", "https://carrier.invalid/fake", mcpserver.AuthTypeNone, "sec1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := newCarrierTestHandler(t, mockDB, mockNotify, carrierTestTable())

			if !tt.expectReject {
				mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().McpServerGet(gomock.Any(), gomock.Any()).Return(&mcpserver.McpServer{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), mcpserver.EventTypeCreated, gomock.Any())
			}

			_, err := h.Create(context.Background(), customerID, "n", "d", tt.url, mcpserver.StatusActive, tt.authType, "", tt.secret)
			if !tt.expectReject {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			ve := assertReasonAndMessage(t, err, "INVALID_MCP_SERVER_SECRET")
			if strings.Contains(ve.Message, tt.secret) {
				t.Errorf("the message must not repeat the secret: %q", ve.Message)
			}
		})
	}
}

func Test_Update_carrierURLAndSecret(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	str := func(s string) *string { return &s }
	authBearer := mcpserver.AuthTypeBearer

	live := func(url string, auth mcpserver.AuthType) *mcpserver.McpServer {
		return &mcpserver.McpServer{Identity: identity.Identity{ID: id, CustomerID: customerID}, URL: url, AuthType: auth}
	}

	tests := []struct {
		name  string
		table mcpserver.CarrierTable
		live  *mcpserver.McpServer

		url      *string
		authType *mcpserver.AuthType
		secret   *string

		expectReason string // empty: the update goes through
	}{
		{"changing to an unknown reserved path is refused", carrierTestTable(), live("https://mcp.example.com/", ""), str("https://carrier.invalid/nope"), nil, nil, "INVALID_MCP_SERVER_URL"},
		{"changing to a known reserved url is accepted", carrierTestTable(), live("https://mcp.example.com/", ""), str("https://carrier.invalid/fake"), nil, nil, ""},
		{"an unchanged url is not checked again after the carrier was removed", mcpserver.DefaultCarrierTable(), live("https://carrier.invalid/fake", mcpserver.AuthTypeBearer), str("https://carrier.invalid/fake"), nil, nil, ""},
		{"a rename without a url is not checked", mcpserver.DefaultCarrierTable(), live("https://carrier.invalid/fake", mcpserver.AuthTypeBearer), nil, nil, nil, ""},
		{"changing to a reserved url while the table is empty is refused", mcpserver.DefaultCarrierTable(), live("https://mcp.example.com/", ""), str("https://carrier.invalid/fake"), nil, nil, "INVALID_MCP_SERVER_URL"},
		{"a bad secret for the stored carrier url is refused", carrierTestTable(), live("https://carrier.invalid/fake", mcpserver.AuthTypeBearer), nil, nil, str("nocolon"), "INVALID_MCP_SERVER_SECRET"},
		{"a good secret for the stored carrier url is accepted", carrierTestTable(), live("https://carrier.invalid/fake", mcpserver.AuthTypeBearer), nil, nil, str("id1:sec1"), ""},
		{"a bad secret together with a new carrier url is refused", carrierTestTable(), live("https://mcp.example.com/", ""), str("https://carrier.invalid/fake"), &authBearer, str("nocolon"), "INVALID_MCP_SERVER_SECRET"},
		{"the stored auth type none skips the secret check", carrierTestTable(), live("https://carrier.invalid/fake", ""), nil, nil, str("nocolon"), ""},
		{"clearing the secret is not a format error", carrierTestTable(), live("https://carrier.invalid/fake", mcpserver.AuthTypeBearer), nil, nil, str(""), ""},
		{"a url-only change keeps the stored secret unchecked (the call time check closes it)", carrierTestTable(), live("https://mcp.example.com/", mcpserver.AuthTypeBearer), str("https://carrier.invalid/fake"), nil, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := newCarrierTestHandler(t, mockDB, mockNotify, tt.table)

			mockDB.EXPECT().McpServerGet(gomock.Any(), id).Return(tt.live, nil).MinTimes(1)
			if tt.expectReason == "" {
				mockDB.EXPECT().McpServerUpdate(gomock.Any(), id, gomock.Any()).Return(nil).AnyTimes()
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), mcpserver.EventTypeUpdated, gomock.Any()).AnyTimes()
			}

			name := str("renamed")
			_, err := h.Update(context.Background(), id, name, nil, tt.url, nil, tt.authType, nil, tt.secret)
			if tt.expectReason == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			assertInvalidArgument(t, err, tt.expectReason)
		})
	}
}

func Test_guardInvalidDial(t *testing.T) {
	tests := []struct {
		name         string
		addr         string
		expectDialed bool
	}{
		{"reserved host", "carrier.invalid:443", false},
		{"upper case", "CARRIER.INVALID:443", false},
		{"trailing dots", "carrier.invalid..:443", false},
		{"ideographic stop", "carrier\u3002invalid:443", false},
		{"any other invalid host", "example.invalid:443", false},
		{"the invalid top level name itself", "invalid:443", false},
		{"a single trailing dot", "carrier.invalid.:443", false},
		{"unparseable address fails closed", "carrier.invalid", false},
		{"empty address fails closed", "", false},
		{"ipv6 literal passes to the dialer", "[2001:db8::1]:443", true},
		{"ordinary host", "mcp.example.com:443", true},
		{"a host that merely contains the word", "notinvalid.example.com:443", true},
		{"a host ending with the word without a dot", "xinvalid:443", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialed := false
			fake := func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialed = true
				return nil, errors.New("fake dial")
			}

			_, err := guardInvalidDial(fake)(context.Background(), "tcp", tt.addr)
			if dialed != tt.expectDialed {
				t.Fatalf("dialed = %v, expected %v (err: %v)", dialed, tt.expectDialed, err)
			}
			if !tt.expectDialed && !errors.Is(err, errReservedHostDial) {
				t.Errorf("expected errReservedHostDial, got %v", err)
			}
		})
	}
}

func Test_NewSSRFGuardedClient_neverDialsReservedHost(t *testing.T) {
	client := NewSSRFGuardedClient(2e9)

	for _, u := range []string{"https://carrier.invalid/fake", "https://CARRIER.invalid../fake", "https://carrier\u3002invalid/fake", "https://anything.invalid/x"} {
		resp, err := client.Get(u) //nolint:noctx // exercises the dial guard only; the dial is refused before any network use
		if resp != nil {
			_ = resp.Body.Close()
		}
		if err == nil || !errors.Is(err, errReservedHostDial) {
			t.Errorf("GET %q: expected errReservedHostDial, got %v", u, err)
		}
	}
}
