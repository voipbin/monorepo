package identityverificationhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/customer"
)

// Session represents a verification session initiated with a provider.
type Session struct {
	ID          string    // Provider-assigned session ID
	CustomerID  uuid.UUID // Customer being verified
	ProviderURL string    // URL to redirect user for verification
}

// Result represents the outcome of a verification session.
type Result struct {
	SessionID  string                              // Provider-assigned session ID
	CustomerID uuid.UUID                           // Customer being verified
	Status     customer.IdentityVerificationStatus // Resulting verification status
	Reason     string                              // Rejection reason, empty if verified
}

// IdentityVerificationProvider defines the interface for identity verification providers.
type IdentityVerificationProvider interface {
	// CreateSession initiates a verification session for a customer.
	CreateSession(ctx context.Context, customerID uuid.UUID) (*Session, error)

	// GetResult retrieves the verification result for a session.
	GetResult(ctx context.Context, sessionID string) (*Result, error)

	// HandleWebhook processes a callback from the verification provider.
	HandleWebhook(ctx context.Context, payload []byte) (*Result, error)
}
