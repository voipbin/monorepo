package talkhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-talk-manager/models/talk"
)

// TalkCreate creates a new talk
func (h *talkHandler) TalkCreate(ctx context.Context, customerID uuid.UUID, talkType talk.Type) (*talk.Talk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TalkCreate",
		"customer_id": customerID,
		"type":        talkType,
	})
	log.Debug("Creating a new talk")

	// Validate input
	if customerID == uuid.Nil {
		log.Error("Invalid customer ID: nil UUID")
		return nil, errors.New("customer ID cannot be nil")
	}

	// Validate talk type
	if talkType != talk.TypeNormal && talkType != talk.TypeGroup {
		log.Errorf("Invalid talk type: %s", talkType)
		return nil, errors.Errorf("invalid talk type: %s", talkType)
	}

	// Create talk object
	t := &talk.Talk{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: customerID,
		},
		Type: talkType,
	}

	// Save to database
	err := h.dbHandler.TalkCreate(ctx, t)
	if err != nil {
		log.Errorf("Failed to create talk in database. err: %v", err)
		return nil, errors.Wrap(err, "failed to create talk in database")
	}

	// Publish webhook event
	h.notifyHandler.PublishWebhookEvent(ctx, t.CustomerID, talk.EventTypeTalkCreated, t)

	log.WithField("talk_id", t.ID).Debug("Talk created successfully")
	return t, nil
}

// TalkGet retrieves a talk by ID
func (h *talkHandler) TalkGet(ctx context.Context, id uuid.UUID) (*talk.Talk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "TalkGet",
		"talk_id": id,
	})

	t, err := h.dbHandler.TalkGet(ctx, id)
	if err != nil {
		log.Errorf("Failed to get talk. err: %v", err)
		return nil, errors.Wrap(err, "failed to get talk")
	}

	return t, nil
}

// TalkList retrieves talks with filters and pagination
func (h *talkHandler) TalkList(ctx context.Context, filters map[talk.Field]any, token string, size uint64) ([]*talk.Talk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "TalkList",
		"filters": filters,
		"token":   token,
		"size":    size,
	})

	talks, err := h.dbHandler.TalkList(ctx, filters, token, size)
	if err != nil {
		log.Errorf("Failed to list talks. err: %v", err)
		return nil, errors.Wrap(err, "failed to list talks")
	}

	return talks, nil
}

// TalkDelete soft deletes a talk
func (h *talkHandler) TalkDelete(ctx context.Context, id uuid.UUID) (*talk.Talk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "TalkDelete",
		"talk_id": id,
	})
	log.Debug("Deleting talk")

	// Get talk before deletion for webhook
	t, err := h.dbHandler.TalkGet(ctx, id)
	if err != nil {
		log.Errorf("Failed to get talk before deletion. err: %v", err)
		return nil, errors.Wrap(err, "failed to get talk before deletion")
	}

	// Soft delete in database
	err = h.dbHandler.TalkDelete(ctx, id)
	if err != nil {
		log.Errorf("Failed to delete talk. err: %v", err)
		return nil, errors.Wrap(err, "failed to delete talk")
	}

	// Publish webhook event
	h.notifyHandler.PublishWebhookEvent(ctx, t.CustomerID, talk.EventTypeTalkDeleted, t)

	log.Debug("Talk deleted successfully")
	return t, nil
}
