package messagehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
)

// Create creates a new message.
func (h *messageHandler) Create(ctx context.Context, m *message.Message) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Create",
	})

	if errCreate := h.db.MessageCreate(ctx, m); errCreate != nil {
		log.Errorf("Could not create the message. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.MessageGet(ctx, m.ID)
	if err != nil {
		log.Errorf("Could not get created message. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageCreated, res)

	return res, nil
}

// UpdateTargets updates the targets.
func (h *messageHandler) UpdateTargets(ctx context.Context, id uuid.UUID, targets []target.Target) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "UpdateTargets",
	})

	if errCreate := h.db.MessageUpdateTargets(ctx, id, targets); errCreate != nil {
		log.Errorf("Could not update the message targets. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated message. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageUpdated, res)

	return res, nil
}

// Get returns message info of the given id
func (h *messageHandler) Get(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Get",
			"message_id": id,
		},
	)
	log.Debugf("Get. message_id: %s", id)

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. message: %s, err:%v", id, err)
		return nil, err
	}

	return res, nil
}

// Gets returns list of messges info of the given customer_id
func (h *messageHandler) Gets(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*message.Message, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Gets",
			"customer_id": customerID,
		},
	)
	log.Debugf("Gets the messages. customer_id: %s", customerID)

	if pageToken == "" {
		pageToken = dbhandler.GetCurTime()
	}

	res, err := h.db.MessageGets(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get messages. customer_id: %s, err:%v", customerID, err)
		return nil, err
	}
	log.WithField("messages", res).Debugf("Found messages info. count: %d", len(res))

	return res, nil
}
