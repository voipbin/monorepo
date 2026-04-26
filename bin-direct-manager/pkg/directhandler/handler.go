package directhandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/dbhandler"
)

// generateHash generates a unique direct hash using crypto/rand.
// The hash includes the "direct." prefix for SIP URI routing.
func generateHash() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate random bytes: %v", err)
	}
	return direct.DirectPrefix + hex.EncodeToString(b), nil
}

// Create creates a new direct hash.
func (h *directHandler) Create(ctx context.Context, customerID uuid.UUID, resourceType string, resourceID uuid.UUID) (*direct.Direct, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Create",
		"customer_id":   customerID,
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
	log.Debug("Creating a new direct.")

	var res *direct.Direct
	var err error

	// retry up to 3 times on hash collision
	for attempt := 0; attempt < 3; attempt++ {
		hash, errHash := generateHash()
		if errHash != nil {
			log.Errorf("Could not generate hash. err: %v", errHash)
			return nil, errors.Wrap(errHash, "could not generate hash")
		}

		id := h.utilHandler.UUIDCreate()
		d := &direct.Direct{
			Identity: commonidentity.Identity{
				ID:         id,
				CustomerID: customerID,
			},
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Hash:         hash,
		}

		if errCreate := h.dbCreate(ctx, d); errCreate != nil {
			log.WithField("attempt", attempt).Debugf("Could not create direct, retrying. err: %v", errCreate)
			continue
		}

		res, err = h.dbGet(ctx, id)
		if err != nil {
			log.Errorf("Could not get created direct info. err: %v", err)
			return nil, err
		}

		h.publishEvent(ctx, direct.EventTypeDirectCreated, res)
		log.WithField("direct", res).Debug("Created a new direct.")
		return res, nil
	}

	return nil, fmt.Errorf("could not create direct after 3 attempts")
}

// Get returns direct info.
func (h *directHandler) Get(ctx context.Context, id uuid.UUID) (*direct.Direct, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Get",
		"direct_id": id,
	})

	res, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get direct info. err: %v", err)
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameDirectManager,
				"DIRECT_NOT_FOUND",
				"The direct was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	return res, nil
}

// GetByHash returns direct info by hash. Checks cache first, falls back to DB.
func (h *directHandler) GetByHash(ctx context.Context, hash string) (*direct.Direct, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "GetByHash",
		"hash": hash,
	})

	// try cache first
	res, err := h.cache.DirectGetByHash(ctx, hash)
	if err == nil {
		log.WithField("direct", res).Debug("Retrieved direct from cache.")
		return res, nil
	}

	// fall back to DB
	res, err = h.dbGetByHash(ctx, hash)
	if err != nil {
		log.Errorf("Could not get direct by hash. err: %v", err)
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameDirectManager,
				"DIRECT_NOT_FOUND",
				"The direct was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	// populate cache
	_ = h.cache.DirectSetByHash(ctx, hash, res)

	log.WithField("direct", res).Debug("Retrieved direct from DB and cached.")
	return res, nil
}

// Gets returns directs based on filters.
func (h *directHandler) Gets(ctx context.Context, size uint64, token string, filters map[direct.Field]any) ([]*direct.Direct, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
		"size":    size,
		"token":   token,
	})

	res, err := h.dbList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get directs info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete hard-deletes the direct info.
func (h *directHandler) Delete(ctx context.Context, id uuid.UUID) (*direct.Direct, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Delete",
		"direct_id": id,
	})
	log.Debug("Deleting the direct info.")

	// get the direct first for return value and cache invalidation
	res, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get direct info before deletion. err: %v", err)
		return nil, errors.Wrap(err, "could not get direct before deletion")
	}

	if err := h.dbDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the direct. err: %v", err)
		return nil, errors.Wrap(err, "could not delete the direct")
	}

	// invalidate cache
	_ = h.cache.DirectDeleteByHash(ctx, res.Hash)

	h.publishEvent(ctx, direct.EventTypeDirectDeleted, res)

	return res, nil
}

// Regenerate generates a new hash for the same resource.
func (h *directHandler) Regenerate(ctx context.Context, id uuid.UUID) (*direct.Direct, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Regenerate",
		"direct_id": id,
	})
	log.Debug("Regenerating the direct hash.")

	// get the current direct
	current, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get current direct info. err: %v", err)
		return nil, errors.Wrap(err, "could not get current direct")
	}
	oldHash := current.Hash

	// generate a new hash with retry
	var newHash string
	for attempt := 0; attempt < 3; attempt++ {
		hash, errHash := generateHash()
		if errHash != nil {
			log.Errorf("Could not generate hash. err: %v", errHash)
			return nil, errors.Wrap(errHash, "could not generate hash")
		}

		fields := map[direct.Field]any{
			direct.FieldHash: hash,
		}

		if errUpdate := h.dbUpdate(ctx, id, fields); errUpdate != nil {
			log.WithField("attempt", attempt).Debugf("Could not update direct hash, retrying. err: %v", errUpdate)
			continue
		}

		newHash = hash
		break
	}

	if newHash == "" {
		return nil, fmt.Errorf("could not regenerate hash after 3 attempts")
	}

	// invalidate old cache entry
	_ = h.cache.DirectDeleteByHash(ctx, oldHash)

	// get updated record
	res, err := h.dbGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get regenerated direct info. err: %v", err)
		return nil, err
	}

	h.publishEvent(ctx, direct.EventTypeDirectRegenerated, res)
	log.WithField("direct", res).Debug("Regenerated the direct hash.")

	return res, nil
}

// EventCustomerDeleted handles the customer deletion event by deleting all directs for the customer.
func (h *directHandler) EventCustomerDeleted(ctx context.Context, customerID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCustomerDeleted",
		"customer_id": customerID,
	})

	// build filters for the customer's directs
	filters := map[direct.Field]any{
		direct.FieldCustomerID: customerID,
	}

	// get all directs of the customer
	directs, err := h.dbList(ctx, 999, h.utilHandler.TimeGetCurTime(), filters)
	if err != nil {
		log.Errorf("Could not get directs for customer. err: %v", err)
		return err
	}

	for _, d := range directs {
		tmp, err := h.Delete(ctx, d.ID)
		if err != nil {
			log.Errorf("Could not delete the direct: %v", err)
			continue
		}

		log.WithField("direct", tmp).Debugf("Deleted direct. direct_id: %s", tmp.ID)
	}

	return nil
}
