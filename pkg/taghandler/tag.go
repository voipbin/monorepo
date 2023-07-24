package taghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
)

// Gets returns tags
func (h *tagHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
		"size":        size,
		"token":       token,
	})

	res, err := h.db.TagGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get tags info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns tag info.
func (h *tagHandler) Get(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Get",
		"tag_id": id,
	})

	res, err := h.db.TagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get tag info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateBasicInfo updates tag's basic info.
func (h *tagHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "UpdateBasicInfo",
		"tag_id": id,
		"name":   name,
		"detail": detail,
	})

	if err := h.db.TagSetBasicInfo(ctx, id, name, detail); err != nil {
		log.Errorf("Could not update the tag basic info. err: %v", err)
		return nil, err
	}

	res, err := h.db.TagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get tag info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, tag.EventTypeTagUpdated, res)

	return res, nil
}

// Create creates a new tag.
func (h *tagHandler) Create(ctx context.Context, customerID uuid.UUID, name string, detail string) (*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"name":        name,
		"detail":      detail,
	})
	log.Debug("Creating a new tag.")

	id := h.utilHandler.UUIDCreate()
	a := &tag.Tag{
		ID:         id,
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,
	}
	log = log.WithField("tag_id", id)

	if err := h.db.TagCreate(ctx, a); err != nil {
		log.Errorf("Could not create a new tag. err: %v", err)
		return nil, err
	}

	res, err := h.db.TagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created tag info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, tag.EventTypeTagCreated, res)

	log.WithField("tag", res).Debug("Created a new tag.")

	return res, nil
}

// Delete deletes the tag info.
func (h *tagHandler) Delete(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Delete",
		"tag_id": id,
	})
	log.Debug("Deleting the tag info.")

	// get tag
	t, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get tag info. err: %v", err)
		return nil, err
	}

	// get all agents
	ags, err := h.agentHandler.GetsByTagIDs(ctx, t.CustomerID, []uuid.UUID{t.ID})
	if err != nil {
		log.Errorf("Could not get agents. err: %v", err)
		return nil, err
	}

	// delete agent's tag.
	for _, ag := range ags {
		newTagIDs := []uuid.UUID{}
		for i, tID := range ag.TagIDs {
			if tID == id {
				newTagIDs = append(ag.TagIDs[:i], ag.TagIDs[i+1:]...)
				break
			}
		}

		_, err := h.agentHandler.UpdateTagIDs(ctx, ag.ID, newTagIDs)
		if err != nil {
			log.WithField("agent", ag).Errorf("Could not delete the tag from the agent. err: %v", err)
		}
	}

	if err := h.db.TagDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the tag. err: %v", err)
		return nil, err
	}

	res, err := h.db.TagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get tag info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, tag.EventTypeTagDeleted, res)

	return res, nil
}
