package whatsapphandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-conversation-manager/models/account"
)

// Setup validates that the account's provider_data contains a non-empty phone_number_id.
func (h *whatsappHandler) Setup(_ context.Context, ac *account.Account) error {
	var pd account.WhatsAppProviderData
	if err := json.Unmarshal(ac.ProviderData, &pd); err != nil {
		return fmt.Errorf("whatsapphandler: invalid provider_data: %w", err)
	}
	if pd.PhoneNumberID == "" {
		return fmt.Errorf("whatsapphandler: provider_data.phone_number_id is required")
	}
	return nil
}
