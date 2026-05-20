package whatsapphandler

import (
	"context"

	"monorepo/bin-conversation-manager/models/account"
)

// Teardown is a no-op for WhatsApp — no server-side webhook deregistration is needed.
func (h *whatsappHandler) Teardown(_ context.Context, _ *account.Account) error {
	return nil
}
