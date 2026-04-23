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
	// CreateCredentialConnection calls POST /v2/credential_connections.
	// Returns the Telnyx connection ID (held in memory for compensating cleanup only).
	CreateCredentialConnection(ctx context.Context, name string) (connID string, err error)
	// DeleteCredentialConnection calls DELETE /v2/credential_connections/{id}.
	DeleteCredentialConnection(ctx context.Context, connID string) error
}

type telnyxClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewTelnyxClient creates a TelnyxClient for a single request. The API key is
// never persisted — it is used only for the duration of the setup call.
func NewTelnyxClient(apiKey string) TelnyxClient {
	return &telnyxClient{
		apiKey:  apiKey,
		baseURL: telnyxBaseURL,
		httpClient: &http.Client{Timeout: telnyxTimeout},
	}
}

// newTelnyxClientWithBase is used in tests to inject a custom base URL.
func newTelnyxClientWithBase(apiKey, baseURL string) TelnyxClient {
	return &telnyxClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: telnyxTimeout},
	}
}
