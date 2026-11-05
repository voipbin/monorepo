package mcptoolhandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"

	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

const carrierSecret = "SENTINEL-ID:SENTINEL-SECRET"

func carrierTable() mcpserver.CarrierTable {
	return mcpserver.CarrierTable{"/fake": {Carrier: "fake", Credential: mcpserver.CredentialClientIDSecret}}
}

// fakeCarriers records what the handler gives it.
type fakeCarriers struct {
	tools    map[string][]mcpserver.CarrierToolDef
	calls    atomic.Int32
	lastCall mcpserver.CarrierCall
	read     func() (string, bool, error)

	text    string
	isError bool
	err     error
}

func (f *fakeCarriers) Tools(carrier string) ([]mcpserver.CarrierToolDef, bool) {
	defs, ok := f.tools[carrier]
	return defs, ok
}

func (f *fakeCarriers) Call(ctx context.Context, call mcpserver.CarrierCall) (string, bool, error) {
	f.calls.Add(1)
	f.lastCall = call
	if call.Secret != nil {
		secret, present, err := call.Secret()
		f.read = func() (string, bool, error) { return secret, present, err }
		if err != nil {
			return "", false, err
		}
	}
	return f.text, f.isError, f.err
}

// carrierHandler builds a handler whose network client fails the test when it
// is made: a reserved-host server must never reach the network layer.
func carrierHandler(t *testing.T, db dbhandler.DBHandler, table mcpserver.CarrierTable, carriers CarrierHandler) (*mcpToolHandler, *atomic.Int32) {
	t.Helper()
	var networkUsed atomic.Int32
	h := &mcpToolHandler{
		db:           db,
		crypto:       newCoverageCrypto(t),
		timeout:      time.Second,
		carrierTable: table,
		carriers:     carriers,
		newClient: func(time.Duration) *http.Client {
			networkUsed.Add(1)
			return &http.Client{Transport: failTransport{}}
		},
	}
	return h, &networkUsed
}

type failTransport struct{}

func (failTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("no network in this test")
}

func carrierServer(t *testing.T, url string, auth mcpserver.AuthType, secret string) *mcpserver.McpServer {
	t.Helper()
	m := &mcpserver.McpServer{
		Identity: identity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: uuid.Must(uuid.NewV4())},
		URL:      url,
		Status:   mcpserver.StatusActive,
		AuthType: auth,
	}
	if secret != "" {
		ct, nonce, version, err := newCoverageCrypto(t).Encrypt(secret)
		if err != nil {
			t.Fatal(err)
		}
		m.SecretCiphertext, m.SecretNonce, m.KeyVersion = ct, nonce, version
	}
	return m
}

func Test_ListTools_carrier(t *testing.T) {
	def := mcpserver.CarrierToolDef{Name: "track_fake", Description: "Look up a package.", InputSchema: json.RawMessage(`{"type":"object"}`)}

	tests := []struct {
		name     string
		url      string
		table    mcpserver.CarrierTable
		carriers CarrierHandler

		expectTools []string
		expectErr   bool
	}{
		{"known carrier", "https://carrier.invalid/fake", carrierTable(), &fakeCarriers{tools: map[string][]mcpserver.CarrierToolDef{"fake": {def}}}, []string{"track_fake"}, false},
		{"known carrier, odd spelling of the host", "https://CARRIER.invalid./fake", carrierTable(), &fakeCarriers{tools: map[string][]mcpserver.CarrierToolDef{"fake": {def}}}, []string{"track_fake"}, false},
		{"the table has it but no adapter is registered", "https://carrier.invalid/fake", carrierTable(), &fakeCarriers{}, nil, true},
		{"the table has it and no handler is wired", "https://carrier.invalid/fake", carrierTable(), nil, nil, true},
		{"the table lost the carrier", "https://carrier.invalid/fake", mcpserver.DefaultCarrierTable(), &fakeCarriers{tools: map[string][]mcpserver.CarrierToolDef{"fake": {def}}}, nil, true},
		{"an unknown path", "https://carrier.invalid/other", carrierTable(), &fakeCarriers{tools: map[string][]mcpserver.CarrierToolDef{"fake": {def}}}, nil, true},
		{"a query makes it not canonical", "https://carrier.invalid/fake?x=1", carrierTable(), &fakeCarriers{tools: map[string][]mcpserver.CarrierToolDef{"fake": {def}}}, nil, true},
		{"the ideographic stop variant is still the reserved host", "https://carrier\u3002invalid/fake", carrierTable(), &fakeCarriers{tools: map[string][]mcpserver.CarrierToolDef{"fake": {def}}}, []string{"track_fake"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)

			m := carrierServer(t, tt.url, mcpserver.AuthTypeBearer, carrierSecret)
			mockDB.EXPECT().McpServerGet(gomock.Any(), m.ID).Return(m, nil)

			h, networkUsed := carrierHandler(t, mockDB, tt.table, tt.carriers)
			tools, err := h.ListTools(context.Background(), m.ID)

			if networkUsed.Load() != 0 {
				t.Fatalf("a reserved-host server must never reach the network layer")
			}
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got tools %+v", tools)
				}
				if strings.Contains(err.Error(), "carrier.invalid") || strings.Contains(err.Error(), "SENTINEL") {
					t.Errorf("the error must not quote the url or the key: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tools) != len(tt.expectTools) || tools[0].Name != tt.expectTools[0] || tools[0].Description != def.Description || string(tools[0].InputSchema) != `{"type":"object"}` {
				t.Errorf("unexpected tools: %+v", tools)
			}
		})
	}
}

func Test_ListTools_otherInvalidHostsStillUseTheNetworkPath(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	m := carrierServer(t, "https://example.invalid/mcp", mcpserver.AuthTypeNone, "")
	mockDB.EXPECT().McpServerGet(gomock.Any(), m.ID).Return(m, nil)

	h, networkUsed := carrierHandler(t, mockDB, carrierTable(), &fakeCarriers{})
	if _, err := h.ListTools(context.Background(), m.ID); err == nil {
		t.Fatalf("the fake transport fails, so an error is expected")
	}
	if networkUsed.Load() == 0 {
		t.Errorf("a .invalid host that is not the reserved one must take the existing path")
	}
}

func Test_ListTools_deletedCarrierServerIsRefused(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	m := carrierServer(t, "https://carrier.invalid/fake", mcpserver.AuthTypeBearer, carrierSecret)
	now := time.Now()
	m.TMDelete = &now
	mockDB.EXPECT().McpServerGet(gomock.Any(), m.ID).Return(m, nil)

	f := &fakeCarriers{tools: map[string][]mcpserver.CarrierToolDef{"fake": {{Name: "track_fake"}}}}
	h, _ := carrierHandler(t, mockDB, carrierTable(), f)
	if _, err := h.ListTools(context.Background(), m.ID); err == nil {
		t.Errorf("a deleted server must be refused")
	}
}

func Test_CallTool_carrier(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	m := carrierServer(t, "https://carrier.invalid/fake", mcpserver.AuthTypeBearer, carrierSecret)
	mockDB.EXPECT().McpServerGet(gomock.Any(), m.ID).Return(m, nil)

	f := &fakeCarriers{text: `{"ok":true}`}
	h, networkUsed := carrierHandler(t, mockDB, carrierTable(), f)

	text, isError, err := h.CallTool(context.Background(), m.ID, "track_fake", `{"tracking_number":"X"}`)
	if err != nil || isError || text != `{"ok":true}` {
		t.Fatalf("got %q %v %v", text, isError, err)
	}
	if networkUsed.Load() != 0 {
		t.Errorf("the network layer must not be used")
	}

	c := f.lastCall
	if c.CustomerID != m.CustomerID || c.ServerID != m.ID || c.Carrier != "fake" || c.Credential != mcpserver.CredentialClientIDSecret ||
		c.ToolName != "track_fake" || c.ArgumentsJSON != `{"tracking_number":"X"}` {
		t.Errorf("the call was not built from the record: %+v", c)
	}
	secret, present, serr := f.read()
	if serr != nil || !present || secret != carrierSecret {
		t.Errorf("the closure must decrypt the saved secret, got %q %v %v", secret, present, serr)
	}
}

func Test_CallTool_carrierResultsPassThrough(t *testing.T) {
	boom := errors.New("system fault")
	tests := []struct {
		name    string
		f       *fakeCarriers
		expect  string
		isError bool
		err     error
	}{
		{"error result", &fakeCarriers{text: "fixed text", isError: true}, "fixed text", true, nil},
		{"system error", &fakeCarriers{err: boom}, "", false, boom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			m := carrierServer(t, "https://carrier.invalid/fake", mcpserver.AuthTypeBearer, carrierSecret)
			mockDB.EXPECT().McpServerGet(gomock.Any(), m.ID).Return(m, nil)

			h, _ := carrierHandler(t, mockDB, carrierTable(), tt.f)
			text, isError, err := h.CallTool(context.Background(), m.ID, "track_fake", "{}")
			if text != tt.expect || isError != tt.isError || !errors.Is(err, tt.err) {
				t.Errorf("got %q %v %v", text, isError, err)
			}
		})
	}
}

func Test_CallTool_carrierIsUnavailable(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		table    mcpserver.CarrierTable
		carriers func() *fakeCarriers
	}{
		{"the carrier was removed from the table", "https://carrier.invalid/fake", mcpserver.DefaultCarrierTable(), func() *fakeCarriers { return &fakeCarriers{text: "x"} }},
		{"production table does not list the path", "https://carrier.invalid/unlisted", mcpserver.DefaultCarrierTable(), func() *fakeCarriers { return &fakeCarriers{text: "x"} }},
		{"unknown path", "https://carrier.invalid/other", carrierTable(), func() *fakeCarriers { return &fakeCarriers{text: "x"} }},
		{"trailing dots", "https://carrier.invalid../ups", mcpserver.CarrierTable{}, func() *fakeCarriers { return &fakeCarriers{text: "x"} }},
		{"query", "https://carrier.invalid/fake?a=b", carrierTable(), func() *fakeCarriers { return &fakeCarriers{text: "x"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			m := carrierServer(t, tt.url, mcpserver.AuthTypeBearer, carrierSecret)
			mockDB.EXPECT().McpServerGet(gomock.Any(), m.ID).Return(m, nil)

			f := tt.carriers()
			h, networkUsed := carrierHandler(t, mockDB, tt.table, f)

			logs := logtest.New(t)
			text, isError, err := h.CallTool(context.Background(), m.ID, "track_fake", "{}")
			if err != nil || !isError || text != mcpserver.IntegrationUnavailableText {
				t.Fatalf("got %q %v %v", text, isError, err)
			}
			if networkUsed.Load() != 0 || f.calls.Load() != 0 {
				t.Errorf("neither the network nor the adapter may be used")
			}
			logs.AssertAbsent(t, "SENTINEL", "carrier.invalid")
		})
	}
}

func Test_CallTool_carrierWithoutAHandlerIsUnavailable(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)
	m := carrierServer(t, "https://carrier.invalid/fake", mcpserver.AuthTypeBearer, carrierSecret)
	mockDB.EXPECT().McpServerGet(gomock.Any(), m.ID).Return(m, nil)

	h, networkUsed := carrierHandler(t, mockDB, carrierTable(), nil)
	text, isError, err := h.CallTool(context.Background(), m.ID, "track_fake", "{}")
	if err != nil || !isError || text != mcpserver.IntegrationUnavailableText || networkUsed.Load() != 0 {
		t.Errorf("got %q %v %v (network %d)", text, isError, err, networkUsed.Load())
	}
}

func Test_CallTool_carrierSecretClosure(t *testing.T) {
	tests := []struct {
		name         string
		auth         mcpserver.AuthType
		secret       string
		corrupt      bool
		expectSecret string
		present      bool
		expectErr    bool
	}{
		{"bearer", mcpserver.AuthTypeBearer, carrierSecret, false, carrierSecret, true, false},
		{"api key", mcpserver.AuthTypeAPIKey, carrierSecret, false, carrierSecret, true, false},
		{"auth none ignores a saved secret", mcpserver.AuthTypeNone, carrierSecret, false, "", false, false},
		{"oauth is not read", mcpserver.AuthTypeOAuth, carrierSecret, false, "", false, false},
		{"no secret saved", mcpserver.AuthTypeBearer, "", false, "", false, false},
		{"a ciphertext that cannot be decrypted", mcpserver.AuthTypeBearer, carrierSecret, true, "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			m := carrierServer(t, "https://carrier.invalid/fake", tt.auth, tt.secret)
			if tt.corrupt {
				m.SecretCiphertext[0] ^= 0xff
			}
			mockDB.EXPECT().McpServerGet(gomock.Any(), m.ID).Return(m, nil)

			f := &fakeCarriers{text: "ok"}
			h, _ := carrierHandler(t, mockDB, carrierTable(), f)
			logs := logtest.New(t)

			_, _, err := h.CallTool(context.Background(), m.ID, "track_fake", "{}")
			secret, present, serr := f.read()
			if tt.expectErr {
				if err == nil || serr == nil {
					t.Fatalf("expected the decrypt error to be passed up, got %v / %v", err, serr)
				}
				if strings.Contains(err.Error(), "SENTINEL") {
					t.Errorf("the error must not carry the secret: %v", err)
				}
				return
			}
			if err != nil || serr != nil || secret != tt.expectSecret || present != tt.present {
				t.Errorf("got (%q, %v, %v) / %v", secret, present, serr, err)
			}
			if logs.Len() > 0 {
				logs.AssertAbsent(t, "SENTINEL")
			}
		})
	}
}

func Test_CallTool_carrierDeletedServerIsRefused(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)
	m := carrierServer(t, "https://carrier.invalid/fake", mcpserver.AuthTypeBearer, carrierSecret)
	now := time.Now()
	m.TMDelete = &now
	mockDB.EXPECT().McpServerGet(gomock.Any(), m.ID).Return(m, nil)

	f := &fakeCarriers{text: "x"}
	h, _ := carrierHandler(t, mockDB, carrierTable(), f)
	if _, _, err := h.CallTool(context.Background(), m.ID, "track_fake", "{}"); err == nil || f.calls.Load() != 0 {
		t.Errorf("a deleted server must be refused before the adapter, err = %v", err)
	}
}

func Test_NewMcpToolHandler_keepsTheCarrierWiring(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := &fakeCarriers{}
	res, err := NewMcpToolHandler(dbhandler.NewMockDBHandler(mc), coverageTestKeys, 5, nil, carrierTable(), f)
	if err != nil {
		t.Fatal(err)
	}
	impl := res.(*mcpToolHandler)
	if len(impl.carrierTable) != 1 || impl.carriers != CarrierHandler(f) {
		t.Errorf("the carrier table and handler were not stored")
	}
}
