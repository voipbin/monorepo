package messagehandler

import (
	"context"
	stderrors "errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
)

// Create creates a new message.
//
// Create only persists the message; it does NOT perform identity-verification
// gating. Outbound sends must route through Send (which gates before creating),
// so any new outbound caller must call Send rather than Create directly. The
// inbound webhook path (DirectionInbound) is intentionally ungated.
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

// Get returns message info of the given id.
//
// When the underlying DB layer returns dbhandler.ErrNotFound, Get returns a
// typed *cerrors.VoipbinError (Status=NotFound) so the api-manager edge can
// recover the upstream domain/reason via errors.As.
func (h *messageHandler) Get(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Get",
		"message_id": id,
	})

	res, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. message: %s, err:%v", id, err)
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameMessageManager,
				"MESSAGE_NOT_FOUND",
				"The message was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	return res, nil
}
