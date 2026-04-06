package teamhandler

import (
	"context"
	"fmt"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/team"
)

// DirectHashRegenerate regenerates (or creates) the direct hash for the given team.
func (h *teamHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*team.Team, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "DirectHashRegenerate",
		"team_id": id,
	})

	// get current team
	t, err := h.db.TeamGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get team. err: %v", err)
		return nil, fmt.Errorf("could not get team: %w", err)
	}
	log.WithField("team", t).Debugf("Retrieved team info. team_id: %s", t.ID)

	// regenerate or create direct
	var d *dmdirect.Direct
	if t.DirectID != uuid.Nil {
		d, err = h.reqHandler.DirectV1DirectRegenerate(ctx, t.DirectID)
		if err != nil {
			log.Errorf("Could not regenerate direct hash. err: %v", err)
			return nil, fmt.Errorf("could not regenerate direct hash: %w", err)
		}
	} else {
		d, err = h.reqHandler.DirectV1DirectCreate(ctx, t.CustomerID, dmdirect.ResourceTypeAITeam, id)
		if err != nil {
			log.Errorf("Could not create direct hash. err: %v", err)
			return nil, fmt.Errorf("could not create direct hash: %w", err)
		}
	}
	log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s, hash: %s", d.ID, d.Hash)

	// update team with new direct info
	fields := map[team.Field]any{
		team.FieldDirectID:   d.ID,
		team.FieldDirectHash: d.Hash,
	}
	if err := h.db.TeamUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update team direct hash. err: %v", err)
		return nil, fmt.Errorf("could not update team: %w", err)
	}

	// return updated team
	res, err := h.db.TeamGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated team. err: %v", err)
		return nil, err
	}

	return res, nil
}
