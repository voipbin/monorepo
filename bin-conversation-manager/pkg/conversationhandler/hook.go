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
func (h *conversationHandler) Hook(ctx context.Context, uri string, method string, signature string, data []byte) error {
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

	case account.TypeWhatsApp:
		if errHook := h.hookWhatsApp(ctx, ac, data, signature); errHook != nil {
			log.Errorf("Could not handle WhatsApp hook. err: %v", errHook)
			return errHook
		}

	default:
		log.Errorf("Unsupported account type. account_type: %s", ac.Type)
		return fmt.Errorf("unsupported account type. account_type: %s", ac.Type)
	}

	return nil
}

// hookWhatsApp handles the WhatsApp type of hook message
func (h *conversationHandler) hookWhatsApp(ctx context.Context, ac *account.Account, data []byte, signature string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "hookWhatsApp",
		"account_id": ac.ID,
	})

	results, err := h.whatsappHandler.Hook(ctx, ac, data, signature)
	if err != nil {
		log.Errorf("Could not parse WhatsApp message. err: %v", err)
		return err
	}

	for _, r := range results {
		if r.Conversation == nil || r.Message == nil {
			continue
		}
		mode := h.getExecuteMode(r.Conversation)
		switch mode {
		case ExecuteModeAgent:
			if errAgent := h.runExecuteModeAgent(ctx, r.Conversation, r.Message); errAgent != nil {
				return errors.Wrapf(errAgent, "could not run agent mode. account_id: %s", ac.ID)
			}
		case ExecuteModeFlow:
			if errFlow := h.runExecuteModeFlow(ctx, r.Conversation, r.Message); errFlow != nil {
				return errors.Wrapf(errFlow, "could not run flow mode. account_id: %s", ac.ID)
			}
		case ExecuteModeNone:
			// no-op
		default:
			return fmt.Errorf("unknown execute mode: %s", mode)
		}
	}
	return nil
}

// HookVerify handles WhatsApp webhook verification challenge
func (h *conversationHandler) HookVerify(ctx context.Context, uri string, mode string, verifyToken string, challenge string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "HookVerify",
		"uri":  uri,
	})

	// Parse account_id from URI path: /v1.0/conversation/accounts/<account_id>
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	tmpVals := strings.Split(u.Path, "/")
	if len(tmpVals) < 5 {
		log.Debugf("Wrong hook verify request. Could not get accountID.")
		return "", fmt.Errorf("could not parse account_id from uri: %s", uri)
	}
	accountID := uuid.FromStringOrNil(tmpVals[4])

	ac, err := h.accountHandler.Get(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return "", errors.Wrap(err, "could not get account")
	}

	if ac.Type != account.TypeWhatsApp {
		return "", fmt.Errorf("unsupported account type for webhook verification: %s", ac.Type)
	}

	return h.whatsappHandler.VerifyWebhook(ctx, ac, mode, verifyToken, challenge)
}

// hookLine handle the line type of hook message
func (h *conversationHandler) hookLine(ctx context.Context, ac *account.Account, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "hookLine",
		"account_id": ac.ID,
	})

	// parse messages and get results back
	results, err := h.lineHandler.Hook(ctx, ac, data)
	if err != nil {
		log.Errorf("Could not parse the message. err: %v", err)
		return err
	}

	for _, r := range results {
		if r.Conversation == nil || r.Message == nil {
			continue
		}

		mode := h.getExecuteMode(r.Conversation)
		switch mode {
		case ExecuteModeAgent:
			if errAgent := h.runExecuteModeAgent(ctx, r.Conversation, r.Message); errAgent != nil {
				return errors.Wrapf(errAgent, "could not run agent mode. account_id: %s", ac.ID)
			}
		case ExecuteModeFlow:
			if errFlow := h.runExecuteModeFlow(ctx, r.Conversation, r.Message); errFlow != nil {
				return errors.Wrapf(errFlow, "could not run flow mode. account_id: %s", ac.ID)
			}
		case ExecuteModeNone:
			// reserved; no-op
		default:
			return fmt.Errorf("unknown execute mode: %s", mode)
		}
	}

	return nil
}
