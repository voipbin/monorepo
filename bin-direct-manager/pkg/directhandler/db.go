package directhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-direct-manager/models/direct"
)

// dbList returns directs
func (h *directHandler) dbList(ctx context.Context, size uint64, token string, filters map[direct.Field]any) ([]*direct.Direct, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "dbList",
		"filters": filters,
		"size":    size,
		"token":   token,
	})

	res, err := h.db.DirectGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get directs info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbGet returns direct info.
func (h *directHandler) dbGet(ctx context.Context, id uuid.UUID) (*direct.Direct, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "dbGet",
		"direct_id": id,
	})

	res, err := h.db.DirectGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get direct info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbGetByHash returns direct info by hash.
func (h *directHandler) dbGetByHash(ctx context.Context, hash string) (*direct.Direct, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "dbGetByHash",
		"hash": hash,
	})

	res, err := h.db.DirectGetByHash(ctx, hash)
	if err != nil {
		log.Errorf("Could not get direct by hash. err: %v", err)
		return nil, err
	}

	return res, nil
}

// dbCreate creates a new direct.
func (h *directHandler) dbCreate(ctx context.Context, d *direct.Direct) error {
	log := logrus.WithFields(logrus.Fields{
		"func":   "dbCreate",
		"direct": d,
	})

	if err := h.db.DirectCreate(ctx, d); err != nil {
		log.Errorf("Could not create a new direct. err: %v", err)
		return err
	}

	return nil
}

// dbDelete deletes the direct info.
func (h *directHandler) dbDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "dbDelete",
		"direct_id": id,
	})

	if err := h.db.DirectDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the direct. err: %v", err)
		return err
	}

	return nil
}

// dbUpdate updates the direct info.
func (h *directHandler) dbUpdate(ctx context.Context, id uuid.UUID, fields map[direct.Field]any) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "dbUpdate",
		"direct_id": id,
		"fields":    fields,
	})

	if err := h.db.DirectUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the direct. err: %v", err)
		return err
	}

	return nil
}
