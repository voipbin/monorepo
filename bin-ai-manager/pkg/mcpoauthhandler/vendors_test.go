package mcpoauthhandler

import (
	"context"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_newVendorCatalog_Credentials(t *testing.T) {
	catalog := newVendorCatalog("gh-id", "gh-secret", "ln-id", "ln-secret", "as-id", "as-secret", "", "")

	tests := []struct {
		vendor     string
		wantID     string
		wantSecret string
	}{
		{VendorGitHub, "gh-id", "gh-secret"},
		{VendorLinear, "ln-id", "ln-secret"},
		{VendorAsana, "as-id", "as-secret"},
	}

	for _, tt := range tests {
		t.Run(tt.vendor, func(t *testing.T) {
			vc, ok := catalog[tt.vendor]
			if !ok {
				t.Fatalf("vendor %q missing from catalog", tt.vendor)
			}
			if vc.ClientID != tt.wantID || vc.ClientSecret != tt.wantSecret {
				t.Errorf("credentials = (%q, %q), want (%q, %q)", vc.ClientID, vc.ClientSecret, tt.wantID, tt.wantSecret)
			}
		})
	}
}

func Test_newVendorCatalog_Asana(t *testing.T) {
	vc := newVendorCatalog("", "", "", "", "as-id", "as-secret", "", "")[VendorAsana]

	if vc.MCPServerURL != "https://mcp.asana.com/v2/mcp" {
		t.Errorf("MCPServerURL = %q", vc.MCPServerURL)
	}
	if vc.AuthorizeURL != "https://app.asana.com/-/oauth_authorize" {
		t.Errorf("AuthorizeURL = %q", vc.AuthorizeURL)
	}
	if vc.TokenURL != "https://app.asana.com/-/oauth_token" {
		t.Errorf("TokenURL = %q", vc.TokenURL)
	}
	if len(vc.Scopes) != 0 {
		t.Errorf("Scopes = %v, want none (Asana rejects an explicit scope)", vc.Scopes)
	}
}

func Test_buildAuthorizeURL_Asana(t *testing.T) {
	vc := newVendorCatalog("", "", "", "", "as-id", "as-secret", "", "")[VendorAsana]

	raw := buildAuthorizeURL(vc, "state-1", "challenge-1")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("could not parse authorize url: %v", err)
	}
	q := u.Query()

	if got := u.Scheme + "://" + u.Host + u.Path; got != vc.AuthorizeURL {
		t.Errorf("authorize endpoint = %q, want %q", got, vc.AuthorizeURL)
	}
	if q.Get("client_id") != "as-id" {
		t.Errorf("client_id = %q", q.Get("client_id"))
	}
	if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") != "challenge-1" {
		t.Errorf("pkce params = (%q, %q)", q.Get("code_challenge_method"), q.Get("code_challenge"))
	}
	if q.Get("response_type") != "code" || q.Get("state") != "state-1" || q.Get("redirect_uri") != oauthRedirectURI {
		t.Errorf("unexpected core params: %v", q)
	}
	if _, ok := q["scope"]; ok {
		t.Errorf("scope must be omitted for Asana, got %q", q.Get("scope"))
	}
	if _, ok := q["resource"]; ok {
		t.Errorf("resource must not be sent, got %q", q.Get("resource"))
	}
}

func Test_vendorDisplayName(t *testing.T) {
	tests := []struct {
		vendor string
		want   string
	}{
		{VendorGitHub, "GitHub"},
		{VendorLinear, "Linear"},
		{VendorAsana, "Asana"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.vendor, func(t *testing.T) {
			if got := vendorDisplayName(tt.vendor); got != tt.want {
				t.Errorf("vendorDisplayName(%q) = %q, want %q", tt.vendor, got, tt.want)
			}
		})
	}
}

func Test_Start_Asana(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := newTestHandler(t, mockDB)

	mockDB.EXPECT().McpOAuthStateCreate(gomock.Any(), gomock.Any()).Return(nil)

	authorizeURL, linkToken, err := h.Start(context.Background(), uuid.Must(uuid.NewV4()), VendorAsana, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if linkToken == "" {
		t.Error("expected a non-empty link_token")
	}

	u, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("could not parse authorize url: %v", err)
	}
	if u.Host != "app.asana.com" || u.Query().Get("client_id") != "asana-client-id" {
		t.Errorf("unexpected authorize url host/client_id: %s", u.Host)
	}
}
