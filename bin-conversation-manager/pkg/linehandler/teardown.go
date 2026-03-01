package linehandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/account"
)

// Teardown removes the LINE webhook for the given account
func (h *lineHandler) Teardown(ctx context.Context, ac *account.Account) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Teardown",
		"account_id": ac.ID,
	})

	c, err := h.getClient(ctx, ac)
	if err != nil {
		log.Errorf("Could not get client. err: %v", err)
		return err
	}

	_, err = c.SetWebhookEndpointURL("").WithContext(ctx).Do()
	if err != nil {
		log.Errorf("Could not remove webhook uri. err: %v", err)
		return err
	}

	return nil
}
