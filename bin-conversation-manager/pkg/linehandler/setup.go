package linehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/account"
)

// Setup sets up the given customer's line webhook
func (h *lineHandler) Setup(ctx context.Context, ac *account.Account) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Setup",
		"account_id": ac.ID,
	})

	c, err := h.getClient(ctx, ac)
	if err != nil {
		log.Errorf("Could not get client. err: %v", err)
		return err
	}

	// create webhook uri
	// "https://hook.voipbin.net/v1.0/conversation/accounts/e8f5795a-e6eb-11ec-bb81-c3cec34bd99c",
	uri := fmt.Sprintf("https://hook.voipbin.net/v1.0/conversation/accounts/%s", ac.ID)
	_, err = c.SetWebhookEndpointURL(uri).WithContext(ctx).Do()
	if err != nil {
		log.Errorf("Could not set webhook uri. err: %v", err)
		return err
	}

	return nil
}
