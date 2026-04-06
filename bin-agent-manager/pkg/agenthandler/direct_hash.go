package agenthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/models/agent"
	dmdirect "monorepo/bin-direct-manager/models/direct"
)

// DirectHashRegenerate regenerates (or creates) the direct hash for the given agent.
func (h *agentHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "DirectHashRegenerate",
		"agent_id": id,
	})

	// get current agent
	a, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		return nil, fmt.Errorf("could not get agent: %w", err)
	}
	log.WithField("agent", a).Debugf("Retrieved agent info. agent_id: %s", a.ID)

	// regenerate or create direct
	var directID uuid.UUID
	var directHash string
	if a.DirectID != uuid.Nil {
		d, err := h.reqHandler.DirectV1DirectRegenerate(ctx, a.DirectID)
		if err != nil {
			log.Errorf("Could not regenerate direct hash. err: %v", err)
			return nil, fmt.Errorf("could not regenerate direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	} else {
		d, err := h.reqHandler.DirectV1DirectCreate(ctx, a.CustomerID, dmdirect.ResourceTypeAgent, id)
		if err != nil {
			log.Errorf("Could not create direct hash. err: %v", err)
			return nil, fmt.Errorf("could not create direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash created. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	}

	// update agent with new direct info
	fields := map[agent.Field]any{
		agent.FieldDirectID:   directID,
		agent.FieldDirectHash: directHash,
	}
	if err := h.db.AgentUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update agent direct hash. err: %v", err)
		return nil, fmt.Errorf("could not update agent: %w", err)
	}

	// return updated agent
	res, err := h.db.AgentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated agent. err: %v", err)
		return nil, err
	}

	return res, nil
}
