package conferencehandler

import (
	"context"
	"fmt"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conference-manager/models/conference"
)

// DirectHashRegenerate regenerates (or creates) the direct hash for the given conference.
func (h *conferenceHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "DirectHashRegenerate",
		"conference_id": id,
	})

	// get current conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		return nil, fmt.Errorf("could not get conference: %w", err)
	}
	log.WithField("conference", cf).Debugf("Retrieved conference info. conference_id: %s", cf.ID)

	// regenerate or create direct
	var d *dmdirect.Direct
	if cf.DirectID != uuid.Nil {
		d, err = h.reqHandler.DirectV1DirectRegenerate(ctx, cf.DirectID)
		if err != nil {
			log.Errorf("Could not regenerate direct hash. err: %v", err)
			return nil, fmt.Errorf("could not regenerate direct hash: %w", err)
		}
	} else {
		d, err = h.reqHandler.DirectV1DirectCreate(ctx, cf.CustomerID, "conference", id)
		if err != nil {
			log.Errorf("Could not create direct hash. err: %v", err)
			return nil, fmt.Errorf("could not create direct hash: %w", err)
		}
	}
	log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s, hash: %s", d.ID, d.Hash)

	// update conference with new direct info
	fields := map[conference.Field]any{
		conference.FieldDirectID:   d.ID,
		conference.FieldDirectHash: d.Hash,
	}
	if err := h.db.ConferenceUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update conference direct hash. err: %v", err)
		return nil, fmt.Errorf("could not update conference: %w", err)
	}

	// return updated conference
	res, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conference. err: %v", err)
		return nil, err
	}

	return res, nil
}
