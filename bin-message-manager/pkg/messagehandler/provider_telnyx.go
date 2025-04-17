package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_provider_telnyx.go -source provider_telnyx.go -build_flags=-mod=mod

import (
	"context"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/requestexternal"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// MessageHandlerMessagebird is interface for service handle
type MessageHandlerTelnyx interface {
	SendMessage(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string) ([]target.Target, error)
}

// messageHandlerMessagebird structure for service handle
type messageHandlerTelnyx struct {
	requestExternal requestexternal.RequestExternal
}

// NewMessageHandlerMessagebird returns new service handler
func NewMessageHandlerTelnyx(reqExternal requestexternal.RequestExternal) MessageHandlerTelnyx {
	h := &messageHandlerTelnyx{
		requestExternal: reqExternal,
	}

	return h
}

// SendMessage sends the message.
func (h *messageHandlerTelnyx) SendMessage(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string) ([]target.Target, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendMessage",
		"message_id": messageID,
		"source":     source,
		"targets":    targets,
		"text":       text,
	})

	var (
		mu   sync.Mutex
		res  []target.Target
		wg   sync.WaitGroup
		errs []error
	)

	for _, t := range targets {
		wg.Add(1)
		go func(sendTarget target.Target) {
			defer wg.Done()

			log.WithField("destination", sendTarget).Debugf("Sending a message by telnyx. message_id: %s, sender: %s", messageID, source.Target)

			// send a request to messaging providers
			m, err := h.requestExternal.TelnyxSendMessage(ctx, source.Target, sendTarget.Destination.Target, text)
			if err != nil {
				log.Errorf("Could not send message correctly to telnyx. err: %v", err)
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			log.WithField("message", m).Debugf("Received message sending response. message_id: %s", m.Data.ID)

			mu.Lock()
			res = append(res, sendTarget)
			mu.Unlock()
		}(t)
	}
	wg.Wait()

	if len(errs) > 0 {
		return nil, errors.Wrapf(errs[0], "could not send message to telnyx")
	}
	promTelnyxSendTotal.WithLabelValues("sms").Add(float64(len(res)))

	return res, nil
}
