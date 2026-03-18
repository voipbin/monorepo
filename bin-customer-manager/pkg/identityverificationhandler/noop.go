package identityverificationhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/customer"
)

// noopProvider is a no-op implementation that immediately returns verified status.
// Useful for development and testing environments.
type noopProvider struct{}

// NewNoopProvider returns an IdentityVerificationProvider that always succeeds.
func NewNoopProvider() IdentityVerificationProvider {
	return &noopProvider{}
}

func (p *noopProvider) CreateSession(_ context.Context, customerID uuid.UUID) (*Session, error) {
	return &Session{
		ID:          "noop-" + customerID.String(),
		CustomerID:  customerID,
		ProviderURL: "",
	}, nil
}

func (p *noopProvider) GetResult(_ context.Context, sessionID string) (*Result, error) {
	return &Result{
		SessionID:  sessionID,
		CustomerID: uuid.Nil,
		Status:     customer.IdentityVerificationStatusVerified,
		Reason:     "",
	}, nil
}

func (p *noopProvider) HandleWebhook(_ context.Context, _ []byte) (*Result, error) {
	return &Result{
		Status: customer.IdentityVerificationStatusVerified,
		Reason: "",
	}, nil
}
