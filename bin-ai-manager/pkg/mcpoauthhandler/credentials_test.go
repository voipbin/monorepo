package mcpoauthhandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpoauthstate"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	commonerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// countingCredentials is a CredentialsFunc that counts its calls, records the context it was
// called with, and answers from a fixed pair or a fixed error.
type countingCredentials struct {
	mu    sync.Mutex
	calls int
	ctxs  []context.Context

	pair Credentials
	err  error
}

func (c *countingCredentials) fn(ctx context.Context, _ string) (Credentials, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls++
	c.ctxs = append(c.ctxs, ctx)

	return c.pair, c.err
}

func (c *countingCredentials) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.calls
}

func newCountingHandler(t *testing.T, db dbhandler.DBHandler, cc *countingCredentials) *mcpOAuthHandler {
	t.Helper()

	h, err := NewMcpOAuthHandler(db, "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=", cc.fn)
	if err != nil {
		t.Fatalf("could not build the handler: %v", err)
	}

	return h.(*mcpOAuthHandler)
}

// isValidVendor only looks at the static catalog. It must not resolve anything, so an unreadable
// credential store can not turn a known vendor into an unknown one (or the other way round).
func Test_isValidVendor_neverResolves(t *testing.T) {
	cc := &countingCredentials{err: errors.New("the store is down")}
	h := newCountingHandler(t, nil, cc)

	tests := []struct {
		vendor string
		expect bool
	}{
		{VendorGitHub, true},
		{VendorLinear, true},
		{VendorAsana, true},
		{VendorHubSpot, true},
		{"not-a-vendor", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.vendor, func(t *testing.T) {
			if got := h.isValidVendor(tt.vendor); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}

	if cc.count() != 0 {
		t.Errorf("Wrong match. expect: 0 resolutions, got: %d", cc.count())
	}
}

// An unknown vendor is refused before anything is resolved or stored.
func Test_Start_unknownVendorDoesNotResolve(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	cc := &countingCredentials{}
	h := newCountingHandler(t, dbhandler.NewMockDBHandler(mc), cc)

	if _, _, err := h.Start(context.Background(), uuid.Must(uuid.NewV4()), "not-a-vendor", nil); err == nil {
		t.Fatal("Wrong match. expect: an error")
	}
	if cc.count() != 0 {
		t.Errorf("Wrong match. expect: 0 resolutions, got: %d", cc.count())
	}
}

// One Start resolves exactly once, and the pair it resolved is the one in the authorize URL.
func Test_Start_resolvesOnce(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	cc := &countingCredentials{pair: Credentials{ClientID: "resolved-id", ClientSecret: "resolved-secret"}}
	h := newCountingHandler(t, mockDB, cc)

	mockDB.EXPECT().McpOAuthStateCreate(gomock.Any(), gomock.Any()).Return(nil)

	authorizeURL, _, err := h.Start(context.Background(), uuid.Must(uuid.NewV4()), VendorLinear, nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	u, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("Wrong match. expect: a url, got: %v", err)
	}
	if got := u.Query().Get("client_id"); got != "resolved-id" {
		t.Errorf("Wrong match. expect: resolved-id, got: %q", got)
	}
	if strings.Contains(authorizeURL, "resolved-secret") {
		t.Error("Wrong match. the authorize URL holds the client_secret")
	}
	if cc.count() != 1 {
		t.Errorf("Wrong match. expect: 1 resolution, got: %d", cc.count())
	}
}

// assertNotConfigured checks that err is the internal error of a vendor whose app is not stored.
func assertNotConfigured(t *testing.T, err error) {
	t.Helper()

	var ve *commonerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: a VoipbinError, got: %T %v", err, err)
	}
	if ve.Status != commonerrors.StatusInternal || ve.Reason != "MCP_OAUTH_VENDOR_NOT_CONFIGURED" {
		t.Errorf("Wrong match. expect: internal MCP_OAUTH_VENDOR_NOT_CONFIGURED, got: %v %q", ve.Status, ve.Reason)
	}
}

// The pairs that are not usable: nothing, or one value alone (the contract of CredentialsFunc
// allows both, so the handler does not rely on the adapter to prevent them).
var unusablePairs = map[string]Credentials{
	"nothing":            {},
	"client_id only":     {ClientID: "the-id"},
	"client_secret only": {ClientSecret: "the-secret"},
}

// A vendor whose app is not stored fails closed: the error is the not configured reason and no
// state row is created, so there is no authorize URL with an empty client_id.
func Test_Start_notConfiguredFailsClosed(t *testing.T) {
	for name, pair := range unusablePairs {
		t.Run(name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockDB.EXPECT().McpOAuthStateCreate(gomock.Any(), gomock.Any()).Times(0)
			h := newCountingHandler(t, mockDB, &countingCredentials{pair: pair})

			authorizeURL, linkToken, err := h.Start(context.Background(), uuid.Must(uuid.NewV4()), VendorGitHub, nil)
			if err == nil || authorizeURL != "" || linkToken != "" {
				t.Fatalf("Wrong match. expect: an error and no URL or token, got: %q %q %v", authorizeURL, linkToken, err)
			}
			assertNotConfigured(t, err)
		})
	}
}

// A stored pair builds the authorize URL with its client_id (the guard does not touch this path).
func Test_Start_configuredBuildsTheURL(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockDB.EXPECT().McpOAuthStateCreate(gomock.Any(), gomock.Any()).Return(nil)
	h := newCountingHandler(t, mockDB, &countingCredentials{pair: Credentials{ClientID: "the-id", ClientSecret: "the-secret"}})

	authorizeURL, linkToken, err := h.Start(context.Background(), uuid.Must(uuid.NewV4()), VendorGitHub, nil)
	if err != nil || linkToken == "" {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	u, _ := url.Parse(authorizeURL)
	if got := u.Query().Get("client_id"); got != "the-id" {
		t.Errorf("Wrong match. expect: the-id, got: %q", got)
	}
}

// A failed resolution is an error and no state row is created.
func Test_Start_resolveErrorCreatesNoState(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockDB.EXPECT().McpOAuthStateCreate(gomock.Any(), gomock.Any()).Times(0)

	cc := &countingCredentials{err: errors.New("the store is down")}
	h := newCountingHandler(t, mockDB, cc)

	_, _, err := h.Start(context.Background(), uuid.Must(uuid.NewV4()), VendorGitHub, nil)
	if err == nil {
		t.Fatal("Wrong match. expect: an error")
	}
	if !strings.Contains(err.Error(), "MCP_OAUTH_CREDENTIALS_UNAVAILABLE") {
		t.Errorf("Wrong match. expect: the reason code, got: %v", err)
	}
	if cc.count() != 1 {
		t.Errorf("Wrong match. expect: 1 resolution, got: %d", cc.count())
	}
}

func completeFixture(t *testing.T, mc *gomock.Controller, cc *countingCredentials) (*mcpOAuthHandler, *dbhandler.MockDBHandler, uuid.UUID, string) {
	t.Helper()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newCountingHandler(t, mockDB, cc)
	h.utilHandler = mockUtil

	state := "test-state-token"
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Minute)
	customerID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().McpOAuthStateGet(gomock.Any(), state).Return(&mcpoauthstate.McpOAuthState{
		State:        state,
		CustomerID:   customerID,
		Vendor:       VendorLinear,
		PKCEVerifier: "the-verifier",
		TMExpire:     &future,
	}, nil)
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
	mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())).AnyTimes()

	return h, mockDB, customerID, state
}

// A failed resolution in Complete happens before the single-use delete: the state row is kept, no
// token exchange is made.
func Test_Complete_resolveErrorKeepsTheState(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	var vendorCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vendorCalls.Add(1)
	}))
	defer srv.Close()

	cc := &countingCredentials{err: errors.New("the store is down")}
	h, mockDB, customerID, state := completeFixture(t, mc, cc)
	h.vendors[VendorLinear] = vendorMeta{TokenURL: srv.URL}

	mockDB.EXPECT().McpOAuthStateDelete(gomock.Any(), gomock.Any()).Times(0)
	mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).Times(0)

	_, err := h.Complete(context.Background(), customerID, state, "the-code")
	if err == nil || !strings.Contains(err.Error(), "MCP_OAUTH_CREDENTIALS_UNAVAILABLE") {
		t.Fatalf("Wrong match. expect: the reason code, got: %v", err)
	}
	if vendorCalls.Load() != 0 {
		t.Errorf("Wrong match. expect: no vendor call, got: %d", vendorCalls.Load())
	}
	if cc.count() != 1 {
		t.Errorf("Wrong match. expect: 1 resolution, got: %d", cc.count())
	}
}

// A vendor whose app is not stored fails closed in Complete, before the single-use delete: the state
// row is kept, nothing is created and the vendor is not called.
func Test_Complete_notConfiguredKeepsTheState(t *testing.T) {
	for name, pair := range unusablePairs {
		t.Run(name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			var vendorCalls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				vendorCalls.Add(1)
			}))
			defer srv.Close()

			cc := &countingCredentials{pair: pair}
			h, mockDB, customerID, state := completeFixture(t, mc, cc)
			h.vendors[VendorLinear] = vendorMeta{TokenURL: srv.URL}

			mockDB.EXPECT().McpOAuthStateDelete(gomock.Any(), gomock.Any()).Times(0)
			mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).Times(0)

			res, err := h.Complete(context.Background(), customerID, state, "the-code")
			if err == nil || res != nil {
				t.Fatalf("Wrong match. expect: an error and no server, got: %v %v", res, err)
			}
			assertNotConfigured(t, err)
			if vendorCalls.Load() != 0 {
				t.Errorf("Wrong match. expect: no vendor call, got: %d", vendorCalls.Load())
			}
		})
	}
}

// The refresh path is not changed: an empty pair is not turned into the not configured error. It
// reaches the vendor (which refuses an empty client_id), the failure sets the backoff, as before.
func Test_GetValidAccessToken_emptyPairIsNotTheNotConfiguredError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	var vendorCalls atomic.Int32
	var emptyClientID atomic.Bool
	vendor := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		vendorCalls.Add(1)
		emptyClientID.Store(r.PostForm.Get("client_id") == "")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	})

	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")
	h.credentials = (&countingCredentials{}).fn

	_, err := h.GetValidAccessToken(context.Background(), m)
	if err == nil {
		t.Fatal("Wrong match. expect: an error")
	}
	if strings.Contains(err.Error(), "MCP_OAUTH_VENDOR_NOT_CONFIGURED") {
		t.Errorf("Wrong match. expect: not the not configured reason, got: %v", err)
	}
	if vendorCalls.Load() != 1 || !emptyClientID.Load() {
		t.Errorf("Wrong match. expect: one vendor call with an empty client_id, got: %d (empty: %v)", vendorCalls.Load(), emptyClientID.Load())
	}
	if !h.isInRefreshBackoff(m.ID.String()) {
		t.Error("Wrong match. expect: a backoff after the vendor refused")
	}
}

// A resolved pair is what the token request carries, and one Complete resolves once.
func Test_Complete_usesTheResolvedPair(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	cc := &countingCredentials{pair: Credentials{ClientID: "resolved-id", ClientSecret: "resolved-secret"}}
	h, mockDB, customerID, state := completeFixture(t, mc, cc)
	h.vendors[VendorLinear] = vendorMeta{TokenURL: srv.URL}
	h.httpClient = &http.Client{Timeout: 2 * time.Second}

	mockDB.EXPECT().McpOAuthStateDelete(gomock.Any(), state).Return(nil)
	mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().McpServerGet(gomock.Any(), gomock.Any()).Return(&mcpserver.McpServer{}, nil)

	if _, err := h.Complete(context.Background(), customerID, state, "the-code"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if gotForm.Get("client_id") != "resolved-id" || gotForm.Get("client_secret") != "resolved-secret" {
		t.Errorf("Wrong match. expect: the resolved pair in the token request, got: %q %q", gotForm.Get("client_id"), gotForm.Get("client_secret"))
	}
	if cc.count() != 1 {
		t.Errorf("Wrong match. expect: 1 resolution, got: %d", cc.count())
	}
}

// Eight callers that hold the same expired refresh token share one exchange, and the exchange
// resolves the pair once, not once per caller.
func Test_GetValidAccessToken_resolvesOncePerExchange(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("refresh-0")
	vendor.delay = 100 * time.Millisecond
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")
	cc := &countingCredentials{pair: Credentials{ClientID: "resolved-id", ClientSecret: "resolved-secret"}}
	h.credentials = cc.fn

	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m.ID, m.RefreshTokenCiphertext, gomock.Any()).Return(nil).Times(1)

	const callers = 8
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = h.GetValidAccessToken(context.Background(), m)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("Wrong match. caller %d expect: ok, got: %v", i, err)
		}
	}
	if cc.count() != 1 {
		t.Errorf("Wrong match. expect: 1 resolution, got: %d", cc.count())
	}
	if calls, _ := vendor.counts(); calls != 1 {
		t.Errorf("Wrong match. expect: 1 vendor call, got: %d", calls)
	}
}

// A failed resolution in the detached refresh never reaches the vendor, spends no refresh token,
// sets no backoff, and the next call resolves again.
func Test_GetValidAccessToken_resolveErrorIsNotABackoff(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("refresh-0")
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")
	cc := &countingCredentials{err: errors.New("the store is down")}
	h.credentials = cc.fn

	for i := 1; i <= 2; i++ {
		if _, err := h.GetValidAccessToken(context.Background(), m); err == nil {
			t.Fatalf("Wrong match. attempt %d expect: an error", i)
		}
		if h.isInRefreshBackoff(m.ID.String()) {
			t.Fatalf("Wrong match. attempt %d expect: no backoff after a failed resolution", i)
		}
		if cc.count() != i {
			t.Fatalf("Wrong match. attempt %d expect: %d resolutions, got: %d", i, i, cc.count())
		}
	}
	if calls, _ := vendor.counts(); calls != 0 {
		t.Errorf("Wrong match. expect: no vendor call, got: %d", calls)
	}

	// The store is back: the same refresh token is still unspent and works.
	cc.mu.Lock()
	cc.err = nil
	cc.pair = Credentials{ClientID: "resolved-id", ClientSecret: "resolved-secret"}
	cc.mu.Unlock()
	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m.ID, m.RefreshTokenCiphertext, gomock.Any()).Return(nil).Times(1)

	if _, err := h.GetValidAccessToken(context.Background(), m); err != nil {
		t.Fatalf("Wrong match. expect: ok after the store is back, got: %v", err)
	}
	if calls, reused := vendor.counts(); calls != 1 || reused != 0 {
		t.Errorf("Wrong match. expect: 1 accepted vendor call, got: %d (%d refused)", calls, reused)
	}
}

// The resolution of the detached exchange survives the end of the caller's context, and is bounded
// by its own deadline.
func Test_GetValidAccessToken_resolveIsDetachedAndBounded(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)

	vendor := newRotatingVendor("refresh-0")
	h, m := expiredOAuthServer(t, mockDB, vendor, "refresh-0")

	started := make(chan struct{})
	release := make(chan struct{})
	var gotCtx context.Context
	var ctxErrAtResolve atomic.Value
	h.credentials = func(ctx context.Context, _ string) (Credentials, error) {
		gotCtx = ctx
		close(started)
		<-release
		ctxErrAtResolve.Store(ctx.Err() == nil)

		return Credentials{ClientID: "resolved-id", ClientSecret: "resolved-secret"}, nil
	}

	stored := make(chan struct{})
	mockDB.EXPECT().McpServerUpdateOAuthTokensIfCurrent(gomock.Any(), m.ID, m.RefreshTokenCiphertext, gomock.Any()).DoAndReturn(
		func(context.Context, uuid.UUID, []byte, map[mcpserver.Field]any) error {
			close(stored)
			return nil
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	returned := make(chan error, 1)
	go func() {
		_, err := h.GetValidAccessToken(ctx, m)
		returned <- err
	}()

	<-started
	deadline, ok := gotCtx.Deadline()
	if !ok || time.Until(deadline) > credentialResolveTimeout || time.Until(deadline) <= 0 {
		t.Errorf("Wrong match. expect: a deadline of at most %v, got: %v %v", credentialResolveTimeout, deadline, ok)
	}

	// The caller leaves while the pair is being read. The exchange goes on.
	cancel()
	if err := <-returned; err == nil {
		t.Error("Wrong match. expect: the caller to give up")
	}
	close(release)

	select {
	case <-stored:
	case <-time.After(5 * time.Second):
		t.Fatal("Wrong match. the exchange did not finish after the caller left")
	}
	if v, _ := ctxErrAtResolve.Load().(bool); !v {
		t.Error("Wrong match. the resolution context ended with the caller's context")
	}
}

// A handler without a credentials function is a wiring mistake and is refused at construction.
func Test_NewMcpOAuthHandler_requiresCredentials(t *testing.T) {
	if _, err := NewMcpOAuthHandler(nil, "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=", nil); err == nil {
		t.Error("Wrong match. expect: an error for nil credentials")
	}
}
