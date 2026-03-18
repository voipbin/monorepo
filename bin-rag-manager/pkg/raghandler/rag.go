package raghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-rag-manager/models/rag"
)

func (h *ragHandler) RagCreate(ctx context.Context, customerID uuid.UUID, name, description string) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagCreate",
		"customer_id": customerID,
		"name":        name,
	})

	id, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Could not generate UUID. err: %v", err)
		return nil, fmt.Errorf("could not generate rag id: %w", err)
	}

	r := &rag.Rag{
		ID:          id,
		CustomerID:  customerID,
		Name:        name,
		Description: description,
	}

	if errCreate := h.dbHandler.RagCreate(ctx, r); errCreate != nil {
		log.Errorf("Could not create rag. err: %v", errCreate)
		return nil, fmt.Errorf("could not create rag: %w", errCreate)
	}
	log.WithField("rag", r).Debugf("Created rag. rag_id: %s", r.ID)

	return r, nil
}

func (h *ragHandler) RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "RagGet",
		"id":   id,
	})

	r, err := h.dbHandler.RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag. err: %v", err)
		return nil, fmt.Errorf("could not get rag: %w", err)
	}
	log.WithField("rag", r).Debugf("Retrieved rag. rag_id: %s", r.ID)

	return r, nil
}

func (h *ragHandler) RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "RagList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	rags, err := h.dbHandler.RagList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not list rags. err: %v", err)
		return nil, fmt.Errorf("could not list rags: %w", err)
	}
	log.Debugf("Listed rags. count: %d", len(rags))

	return rags, nil
}

func (h *ragHandler) RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "RagUpdate",
		"id":     id,
		"fields": fields,
	})

	if err := h.dbHandler.RagUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update rag. err: %v", err)
		return nil, fmt.Errorf("could not update rag: %w", err)
	}

	r, err := h.dbHandler.RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated rag. err: %v", err)
		return nil, fmt.Errorf("could not get updated rag: %w", err)
	}
	log.WithField("rag", r).Debugf("Updated rag. rag_id: %s", r.ID)

	return r, nil
}

func (h *ragHandler) RagDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "RagDelete",
		"id":   id,
	})

	// cascade: soft-delete chunks and documents first
	if err := h.dbHandler.ChunkSoftDeleteByRagID(ctx, id); err != nil {
		log.Errorf("Could not soft delete chunks. err: %v", err)
		return fmt.Errorf("could not delete rag chunks: %w", err)
	}

	if err := h.dbHandler.DocumentDeleteByRagID(ctx, id); err != nil {
		log.Errorf("Could not soft delete documents. err: %v", err)
		return fmt.Errorf("could not delete rag documents: %w", err)
	}

	if err := h.dbHandler.RagDelete(ctx, id); err != nil {
		log.Errorf("Could not delete rag. err: %v", err)
		return fmt.Errorf("could not delete rag: %w", err)
	}
	log.Debugf("Deleted rag. rag_id: %s", id)

	return nil
}
