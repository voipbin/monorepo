package whatsapphandler

import (
	"context"
	"fmt"

	"monorepo/bin-conversation-manager/models/account"
)

// VerifyWebhook handles the Meta webhook verification handshake.
// It returns the hub.challenge string when mode is "subscribe" and the verify_token matches ac.Secret.
func (h *whatsappHandler) VerifyWebhook(_ context.Context, ac *account.Account, mode string, verifyToken string, challenge string) (string, error) {
	if mode != "subscribe" {
		return "", fmt.Errorf("whatsapphandler: unexpected hub.mode: %q", mode)
	}
	if verifyToken != ac.Secret {
		return "", fmt.Errorf("whatsapphandler: hub.verify_token mismatch")
	}
	return challenge, nil
}
