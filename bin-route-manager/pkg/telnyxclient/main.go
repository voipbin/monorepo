package telnyxclient

//go:generate mockgen -package telnyxclient -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// ErrInvalidKey is returned when the Telnyx API key is invalid or has insufficient permissions.
var ErrInvalidKey = errors.New("invalid or insufficient telnyx api key")

const (
	telnyxBaseURL = "https://api.telnyx.com/v2"
	telnyxTimeout = 10 * time.Second
)

// TelnyxClient performs Telnyx API operations needed for provider setup.
type TelnyxClient interface {
	// ValidateKey calls GET /v2/whoami. Returns ErrInvalidKey on 401/403.
	ValidateKey(ctx context.Context) error

	// CreateOutboundVoiceProfile calls POST /v2/outbound_voice_profiles.
	// Returns the Telnyx profile ID.
	CreateOutboundVoiceProfile(ctx context.Context, name string) (profileID string, err error)
	// DeleteOutboundVoiceProfile calls DELETE /v2/outbound_voice_profiles/{id}.
	DeleteOutboundVoiceProfile(ctx context.Context, profileID string) error

	// CreateFQDNConnection calls POST /v2/fqdn_connections. FQDN connections
	// (unlike IP connections) present inbound calls with the actual FQDN as
	// the SIP request-URI domain (e.g. "pstn.voipbin.net") instead of a raw
	// IP address, which Kamailio's domain validation requires. Telnyx
	// requires credential authentication to be configured before an
	// outbound_voice_profile can be attached to an FQDN connection, so
	// userName/password must be supplied. Returns the Telnyx connection ID.
	CreateFQDNConnection(ctx context.Context, name, profileID, userName, password string) (connID string, err error)
	// DeleteFQDNConnection calls DELETE /v2/fqdn_connections/{id}.
	DeleteFQDNConnection(ctx context.Context, connID string) error

	// RegisterFQDN calls POST /v2/fqdns to attach our public SIP domain to
	// the connection. DNS resolution of fqdn to our SIP LB address(es) is
	// managed outside of Telnyx (see EXTERNAL_SIP_GATEWAY_FQDN_FOR_PSTN). Returns the
	// Telnyx FQDN resource ID.
	RegisterFQDN(ctx context.Context, connID string, fqdn string, port int) (fqdnID string, err error)
	// DeleteFQDN calls DELETE /v2/fqdns/{id}.
	DeleteFQDN(ctx context.Context, fqdnID string) error
}

type telnyxClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// defaultHTTPClient is reused across calls to preserve TCP connection pooling.
var defaultHTTPClient = &http.Client{Timeout: telnyxTimeout}

// NewTelnyxClient creates a TelnyxClient for a single request. The API key is
// never persisted — it is used only for the duration of the setup call.
func NewTelnyxClient(apiKey string) TelnyxClient {
	return &telnyxClient{
		apiKey:     apiKey,
		baseURL:    telnyxBaseURL,
		httpClient: defaultHTTPClient,
	}
}

// newTelnyxClientWithBase is used in tests to inject a custom base URL.
func newTelnyxClientWithBase(apiKey, baseURL string) TelnyxClient {
	return &telnyxClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: telnyxTimeout},
	}
}
