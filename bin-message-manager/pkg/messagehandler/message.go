package messagehandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

// Create creates a new message.
func (h *messageHandler) Create(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, targets []target.Target, providerName message.ProviderName, text string, direction message.Direction) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Create",
		"id":            id,
		"customer_id":   customerID,
		"source":        source,
		"targets":       targets,
		"provider_name": providerName,
		"text":          text,
		"direction":     direction,
	})

	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
	}
	m := &message.Message{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Type: message.TypeSMS,

		Source:  source,
		Targets: targets,

		ProviderName: providerName,

		Text:      text,
		Medias:    []string{},
		Direction: direction,
	}

	// create a message
	res, err := h.dbCreate(ctx, m)
	if err != nil {
		log.Errorf("Could not create a new message. err: %v", err)
		return nil, err
	}

	return res, nil
}

// List returns list of messges info with filters
func (h *messageHandler) List(ctx context.Context, token string, size uint64, filters map[message.Field]any) ([]*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "List",
		"filters": filters,
	})

	res, err := h.dbList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get messages. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes a message info of the given id
func (h *messageHandler) Delete(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"message_id": id,
	})
	log.Debugf("Get. message_id: %s", id)

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete message. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns message info of the given id
func (h *messageHandler) Get(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Get",
		"message_id": id,
	})

	res, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. message: %s, err:%v", id, err)
		return nil, err
	}

	return res, nil
}
