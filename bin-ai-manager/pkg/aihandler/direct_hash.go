package aihandler

import (
	"context"
	"fmt"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
)

// DirectHashRegenerate regenerates (or creates) the direct hash for the given AI.
func (h *aiHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "DirectHashRegenerate",
		"ai_id": id,
	})

	// get current AI
	a, err := h.db.AIGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get ai. err: %v", err)
		return nil, fmt.Errorf("could not get ai: %w", err)
	}
	log.WithField("ai", a).Debugf("Retrieved ai info. ai_id: %s", a.ID)

	// regenerate or create direct
	var d *dmdirect.Direct
	if a.DirectID != uuid.Nil {
		d, err = h.reqHandler.DirectV1DirectRegenerate(ctx, a.DirectID)
		if err != nil {
			log.Errorf("Could not regenerate direct hash. err: %v", err)
			return nil, fmt.Errorf("could not regenerate direct hash: %w", err)
		}
	} else {
		d, err = h.reqHandler.DirectV1DirectCreate(ctx, a.CustomerID, dmdirect.ResourceTypeAI, id)
		if err != nil {
			log.Errorf("Could not create direct hash. err: %v", err)
			return nil, fmt.Errorf("could not create direct hash: %w", err)
		}
	}
	log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s, hash: %s", d.ID, d.Hash)

	// update AI with new direct info
	fields := map[ai.Field]any{
		ai.FieldDirectID:   d.ID,
		ai.FieldDirectHash: d.Hash,
	}
	if err := h.db.AIUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update ai direct hash. err: %v", err)
		return nil, fmt.Errorf("could not update ai: %w", err)
	}

	// return updated AI
	res, err := h.db.AIGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated ai. err: %v", err)
		return nil, err
	}

	return res, nil
}
