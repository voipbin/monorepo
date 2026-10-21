package mcpoauthhandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpoauthstate"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_newVendorCatalog_HubSpot(t *testing.T) {
	vc, ok := newVendorCatalog("gh-id", "gh-secret", "li-id", "li-secret", "", "", "hs-id", "hs-secret")[VendorHubSpot]
	if !ok {
		t.Fatal("the hubspot vendor is missing from the catalog")
	}

	if VendorHubSpot != "hubspot" {
		t.Errorf("VendorHubSpot = %q, want hubspot", VendorHubSpot)
	}
	if vc.MCPServerURL != "https://mcp.hubspot.com" {
		t.Errorf("MCPServerURL = %q", vc.MCPServerURL)
	}
	if vc.AuthorizeURL != "https://mcp.hubspot.com/oauth/authorize/user" {
		t.Errorf("AuthorizeURL = %q", vc.AuthorizeURL)
	}
	if vc.TokenURL != "https://mcp.hubspot.com/oauth/v3/token" {
		t.Errorf("TokenURL = %q", vc.TokenURL)
	}
	if len(vc.Scopes) != 0 {
		t.Errorf("Scopes = %v, want none: the connector scope is fixed at install time", vc.Scopes)
	}
	if vc.ClientID != "hs-id" || vc.ClientSecret != "hs-secret" {
		t.Errorf("credentials were not injected: %q / %q", vc.ClientID, vc.ClientSecret)
	}
}

func Test_buildAuthorizeURL_HubSpot(t *testing.T) {
	vc := newVendorCatalog("", "", "", "", "", "", "hs-id", "hs-secret")[VendorHubSpot]

	raw := buildAuthorizeURL(vc, "the-state", "the-challenge")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("not a URL: %v", err)
	}
	if got := u.Scheme + "://" + u.Host + u.Path; got != "https://mcp.hubspot.com/oauth/authorize/user" {
		t.Errorf("authorize endpoint = %q", got)
	}

	q := u.Query()
	if _, has := q["scope"]; has {
		t.Errorf("scope must be omitted for hubspot, got %q", q.Get("scope"))
	}
	for k, want := range map[string]string{
		"client_id":             "hs-id",
		"redirect_uri":          oauthRedirectURI,
		"response_type":         "code",
		"state":                 "the-state",
		"code_challenge":        "the-challenge",
		"code_challenge_method": "S256",
	} {
		if got := q.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func Test_vendorDisplayName_HubSpot(t *testing.T) {
	if got := vendorDisplayName(VendorHubSpot); got != "HubSpot" {
		t.Errorf("vendorDisplayName = %q, want HubSpot", got)
	}
}

func Test_Start_HubSpot(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandler(t, mockDB)

	var created *mcpoauthstate.McpOAuthState
	mockDB.EXPECT().McpOAuthStateCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, row *mcpoauthstate.McpOAuthState) error {
			created = row
			return nil
		},
	)

	authorizeURL, linkToken, err := h.Start(context.Background(), uuid.Must(uuid.NewV4()), VendorHubSpot, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil || created.Vendor != VendorHubSpot {
		t.Fatalf("state row = %#v, want vendor hubspot", created)
	}
	if linkToken != created.State {
		t.Errorf("link token %q does not match the stored state %q", linkToken, created.State)
	}
	u, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("not a URL: %v", err)
	}
	if u.Host != "mcp.hubspot.com" || u.Query().Get("client_id") != "hubspot-client-id" {
		t.Errorf("unexpected authorize url: %s", authorizeURL)
	}
}

// Test_Complete_HubSpot_NewServer pins the whole code exchange for a new
// connection: the token request carries client_secret in the form body and no
// HTTP Basic header (the only client authentication method HubSpot supports),
// and the created server is named HubSpot with the exact vendor URL.
func Test_Complete_HubSpot_NewServer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := newTestHandler(t, mockDB)
	h.utilHandler = mockUtil

	var gotAuthHeader string
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthHeader = r.Header.Get("Authorization")
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "hubspot-access-token",
			"refresh_token": "hubspot-refresh-token",
			"token_type":    "bearer",
			"expires_in":    1800,
		})
	}))
	defer srv.Close()

	vc := h.vendors[VendorHubSpot]
	vc.TokenURL = srv.URL
	h.vendors[VendorHubSpot] = vc
	h.httpClient = &http.Client{Timeout: 2 * time.Second}

	state := "test-state-token"
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Minute)
	customerID := uuid.Must(uuid.NewV4())
	newID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().McpOAuthStateGet(gomock.Any(), state).Return(&mcpoauthstate.McpOAuthState{
		State:        state,
		CustomerID:   customerID,
		Vendor:       VendorHubSpot,
		PKCEVerifier: "the-verifier",
		TMExpire:     &future,
	}, nil)
	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newID)
	mockDB.EXPECT().McpOAuthStateDelete(gomock.Any(), state).Return(nil)

	var created *mcpserver.McpServer
	mockDB.EXPECT().McpServerCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *mcpserver.McpServer) error {
			created = m
			return nil
		},
	)
	mockDB.EXPECT().McpServerGet(gomock.Any(), newID).Return(&mcpserver.McpServer{}, nil)

	if _, err := h.Complete(context.Background(), customerID, state, "the-code"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuthHeader != "" {
		t.Errorf("token request must not use HTTP Basic, got Authorization header")
	}
	for k, want := range map[string]string{
		"client_id":     "hubspot-client-id",
		"client_secret": "hubspot-client-secret",
		"code":          "the-code",
		"code_verifier": "the-verifier",
		"grant_type":    "authorization_code",
		"redirect_uri":  oauthRedirectURI,
	} {
		if got := gotForm.Get(k); got != want {
			t.Errorf("token form %s = %q, want %q", k, got, want)
		}
	}

	if created == nil {
		t.Fatal("McpServerCreate was not called")
	}
	if created.Name != "HubSpot" || created.URL != "https://mcp.hubspot.com" {
		t.Errorf("created server name/url = %q / %q", created.Name, created.URL)
	}
	if created.AuthType != mcpserver.AuthTypeOAuth || created.OAuthVendor != VendorHubSpot {
		t.Errorf("created server auth = %v / %q", created.AuthType, created.OAuthVendor)
	}
}

// Test_HubSpot_UnconfiguredBehavesLikeOtherVendors pins that an empty
// client id/secret is not rejected by the catalog: hubspot stays a valid vendor
// and a self-hosting operator may leave its env empty, exactly as for GitHub
// and Linear. The vendor refuses the empty client_id at its own endpoint.
func Test_HubSpot_UnconfiguredBehavesLikeOtherVendors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h, err := NewMcpOAuthHandler(mockDB, "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=", "", "", "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("could not build handler: %v", err)
	}
	impl := h.(*mcpOAuthHandler)

	for _, v := range []string{VendorGitHub, VendorLinear, VendorHubSpot} {
		if !impl.isValidVendor(v) {
			t.Errorf("%s is not a valid vendor when unconfigured", v)
		}
	}
}
