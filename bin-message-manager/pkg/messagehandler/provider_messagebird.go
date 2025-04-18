package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_provider_messagebird.go -source provider_messagebird.go -build_flags=-mod=mod

import (
	"context"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/requestexternal"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// MessageHandlerMessagebird is interface for service handle
type MessageHandlerMessagebird interface {
	SendMessage(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string) ([]target.Target, error)
}

// messageHandlerMessagebird structure for service handle
type messageHandlerMessagebird struct {
	requestExternal requestexternal.RequestExternal
}

// NewMessageHandlerMessagebird returns new service handler
func NewMessageHandlerMessagebird(reqExternal requestexternal.RequestExternal) MessageHandlerMessagebird {
	h := &messageHandlerMessagebird{
		requestExternal: reqExternal,
	}

	return h
}

// SendMessage sends the message.
func (h *messageHandlerMessagebird) SendMessage(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string) ([]target.Target, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendMessage",
		"message_id": messageID,
		"source":     source,
		"targets":    targets,
		"text":       text,
	})

	sender := source.Target
	receivers := []string{}
	for _, target := range targets {
		receivers = append(receivers, target.Destination.Target)
	}
	log.WithField("receivers", receivers).Debugf("Sending a messages by messagebird. message_id: %s, sender: %s", messageID, sender)

	// send a request to messaging providers
	m, err := h.requestExternal.MessagebirdSendMessage(ctx, sender, receivers, text)
	if err != nil {
		log.Errorf("Could not send message correctly to the messagebird. err: %v", err)
		return nil, err
	}
	log.WithField("message", m).Debugf("Received message sending response. message_id: %s", m.ID)

	res := m.GetTargets()

	// currently, the only sms type we send
	promMessagebirdSendTotal.WithLabelValues("sms").Add(float64(len(res)))

	return res, nil
}
