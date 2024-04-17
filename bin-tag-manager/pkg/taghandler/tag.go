package taghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tag-manager/models/tag"
)

// Gets returns tags
func (h *tagHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
		"size":        size,
		"token":       token,
	})

	res, err := h.dbGets(ctx, customerID, size, token)
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

	res, err := h.dbGet(ctx, id)
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

	res, err := h.dbUpdateInfo(ctx, id, name, detail)
	if err != nil {
		log.Errorf("Could not update the tag info. err: %v", err)
		return nil, err
	}

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

	res, err := h.dbCreate(ctx, customerID, name, detail)
	if err != nil {
		log.Errorf("Could not create a tag. err: %v", err)
		return nil, errors.Wrap(err, "could not create a tag")
	}

	return res, nil
}

// Delete deletes the tag info.
func (h *tagHandler) Delete(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "Delete",
		"tag_id": id,
	})
	log.Debug("Deleting the tag info.")

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the tag. err: %v", err)
		return nil, errors.Wrap(err, "could not delete the tag")
	}

	return res, nil
}
