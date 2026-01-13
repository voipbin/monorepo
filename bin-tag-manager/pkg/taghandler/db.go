package taghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tag-manager/models/tag"
)

// dbGets returns tags
func (h *tagHandler) dbGets(ctx context.Context, size uint64, token string, filters map[tag.Field]any) ([]*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "dbGets",
		"filters": filters,
		"size":    size,
		"token":   token,
	})

	res, err := h.db.TagGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get tags info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbGet returns tag info.
func (h *tagHandler) dbGet(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "dbGet",
		"tag_id": id,
	})

	res, err := h.db.TagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get tag info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbUpdateInfo updates tag's basic info.
func (h *tagHandler) dbUpdateInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "dbUpdateInfo",
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
		log.Errorf("Could not get updated tag info. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishEvent(ctx, tag.EventTypeTagUpdated, res)

	return res, nil
}

// dbCreate creates a new tag.
func (h *tagHandler) dbCreate(ctx context.Context, customerID uuid.UUID, name string, detail string) (*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "dbCreate",
		"customer_id": customerID,
		"name":        name,
		"detail":      detail,
	})
	log.Debug("Creating a new tag.")

	id := h.utilHandler.UUIDCreate()
	a := &tag.Tag{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Name:   name,
		Detail: detail,
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

// dbDelete deletes the tag info.
func (h *tagHandler) dbDelete(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "dbDelete",
		"tag_id": id,
	})
	log.Debug("Deleting the tag info.")

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
