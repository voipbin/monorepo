package flowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/flow"
)

// DirectHashRegenerate regenerates (or creates) the direct hash for the given flow.
func (h *flowHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "DirectHashRegenerate",
		"flow_id": id,
	})

	// get current flow
	f, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return nil, fmt.Errorf("could not get flow: %w", err)
	}
	log.WithField("flow", f).Debugf("Retrieved flow info. flow_id: %s", f.ID)

	// regenerate or create direct
	var directID uuid.UUID
	var directHash string
	if f.DirectID != uuid.Nil {
		d, err := h.reqHandler.DirectV1DirectRegenerate(ctx, f.DirectID)
		if err != nil {
			log.Errorf("Could not regenerate direct hash. err: %v", err)
			return nil, fmt.Errorf("could not regenerate direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	} else {
		d, err := h.reqHandler.DirectV1DirectCreate(ctx, f.CustomerID, "flow", id)
		if err != nil {
			log.Errorf("Could not create direct hash. err: %v", err)
			return nil, fmt.Errorf("could not create direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash created. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	}

	// update flow with new direct info
	fields := map[flow.Field]any{
		flow.FieldDirectID:   directID,
		flow.FieldDirectHash: directHash,
	}
	if err := h.db.FlowUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update flow direct hash. err: %v", err)
		return nil, fmt.Errorf("could not update flow: %w", err)
	}

	// return updated flow
	res, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated flow. err: %v", err)
		return nil, err
	}

	return res, nil
}
