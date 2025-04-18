package messagehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

// dbGets returns list of messges info of the given customer_id
func (h *messageHandler) dbGets(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbGets",
		"customer_id": customerID,
	})

	if pageToken == "" {
		pageToken = h.utilHandler.TimeGetCurTime()
	}

	res, err := h.db.MessageGets(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get messages. customer_id: %s, err:%v", customerID, err)
		return nil, err
	}

	return res, nil
}

// dbCreate creates a new message.
func (h *messageHandler) dbCreate(ctx context.Context, m *message.Message) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "dbCreate",
		"message": m,
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

// dbUpdateTargets updates the targets.
func (h *messageHandler) dbUpdateTargets(ctx context.Context, id uuid.UUID, providerName message.ProviderName, targets []target.Target) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "dbUpdateTargets",
		"id":      id,
		"targets": targets,
	})

	if errCreate := h.db.MessageUpdateTargets(ctx, id, providerName, targets); errCreate != nil {
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

// dbGet returns message info of the given id
func (h *messageHandler) dbGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "dbGet",
		"message_id": id,
	})

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. message: %s, err:%v", id, err)
		return nil, err
	}

	return res, nil
}

// dbDelete deletes a message info of the given id
func (h *messageHandler) dbDelete(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "dbDelete",
		"message_id": id,
	})

	err := h.db.MessageDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. message_id: %s, err:%v", id, err)
		return nil, err
	}

	res, err := h.db.MessageGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted message info. message_id: %s, err: %v", id, err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, message.EventTypeMessageDeleted, res)

	return res, nil
}
