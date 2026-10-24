package identityverificationhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/customer"
)

func Test_noopProvider(t *testing.T) {
	ctx := context.Background()
	p := NewNoopProvider()
	customerID := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	t.Run("CreateSession binds the session to the customer", func(t *testing.T) {
		s, err := p.CreateSession(ctx, customerID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if s.ID != "noop-4cd23368-7cb7-11ec-9466-8318ef5a7125" {
			t.Errorf("Wrong session id: %s", s.ID)
		}
		if s.CustomerID != customerID {
			t.Errorf("Wrong customer id: %s", s.CustomerID)
		}
		if s.ProviderURL != "" {
			t.Errorf("Wrong provider url: %q", s.ProviderURL)
		}
	})

	t.Run("GetResult echoes the session id and reports verified", func(t *testing.T) {
		r, err := p.GetResult(ctx, "session-1")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if r.SessionID != "session-1" || r.Status != customer.IdentityVerificationStatusVerified || r.Reason != "" {
			t.Errorf("Wrong result: %+v", r)
		}
	})

	t.Run("HandleWebhook reports verified", func(t *testing.T) {
		r, err := p.HandleWebhook(ctx, []byte(`{"any":"payload"}`))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if r.Status != customer.IdentityVerificationStatusVerified || r.Reason != "" {
			t.Errorf("Wrong result: %+v", r)
		}
	})
}
