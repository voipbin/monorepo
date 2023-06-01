package accounthandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
)

// setup sets up the account
func (h *accountHandler) setup(ctx context.Context, ac *account.Account) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "setup",
		"account_id": ac.ID,
	})

	var err error
	switch ac.Type {
	case account.TypeLine:
		err = h.lineHandler.Setup(ctx, ac)

	case account.TypeSMS:
		// nothing to do
		err = nil

	default:
		log.Errorf("Unsupported account type. account_type: %s", ac.Type)
		err = fmt.Errorf("unsupported account type. account_type: %s", ac.Type)
	}
	if err != nil {
		log.Errorf("Could not setup the account. err: %v", err)
		return errors.Wrap(err, "could not setup the account")
	}

	return nil
}
