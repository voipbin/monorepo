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

	// CreateIPConnection calls POST /v2/ip_connections.
	// Returns the Telnyx connection ID.
	CreateIPConnection(ctx context.Context, name string, profileID string) (connID string, err error)
	// DeleteIPConnection calls DELETE /v2/ip_connections/{id}.
	DeleteIPConnection(ctx context.Context, connID string) error

	// RegisterIP calls POST /v2/ips to attach our SIP LB IP to the connection.
	// Returns the Telnyx IP resource ID.
	RegisterIP(ctx context.Context, connID string, ipAddress string, port int) (ipID string, err error)
	// DeleteIP calls DELETE /v2/ips/{id}.
	DeleteIP(ctx context.Context, ipID string) error
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
