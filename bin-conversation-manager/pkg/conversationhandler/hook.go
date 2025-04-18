package conversationhandler

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/account"
)

// Hook handles hooked event
func (h *conversationHandler) Hook(ctx context.Context, uri string, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Hook",
		"uri":  uri,
		"data": data,
	})
	log.Debugf("Hook detail. uri: %s", uri)

	// "https://hook.voipbin.net/v1.0/conversation/accounts/<account_id>",
	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	// /v1.0/conversation/accounts/<account_id>
	tmpVals := strings.Split(u.Path, "/")
	if len(tmpVals) < 4 {
		log.Debugf("Wrong hook request. Could not get customerID.")
		return fmt.Errorf("no customer info found")
	}
	accountID := uuid.FromStringOrNil(tmpVals[4])

	log.Debugf("Parsed data. customer_id: %s", accountID)

	// get account info
	ac, err := h.accountHandler.Get(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return errors.Wrap(err, "could not get account info")
	}

	switch ac.Type {
	case account.TypeLine:
		// line message
		if errEvent := h.hookLine(ctx, ac, data); errEvent != nil {
			log.Errorf("Could not handle the event type line. err: %v", errEvent)
			return errEvent
		}

	default:
		log.Errorf("Unsupported account type. account_type: %s", ac.Type)
		return fmt.Errorf("unsupported account type. account_type: %s", ac.Type)
	}

	return nil
}

// hookLine handle the line type of hook message
func (h *conversationHandler) hookLine(ctx context.Context, ac *account.Account, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "hookLine",
		"account_id": ac.ID,
	})

	// parse a messages
	if errHook := h.lineHandler.Hook(ctx, ac, data); errHook != nil {
		log.Errorf("Could not parse the message. err: %v", errHook)
		return errHook
	}

	return nil
}
