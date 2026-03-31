package queuehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queue"
)

// DirectHashRegenerate regenerates (or creates) the direct hash for the given queue.
func (h *queueHandler) DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "DirectHashRegenerate",
		"queue_id": id,
	})

	// get current queue
	q, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, fmt.Errorf("could not get queue: %w", err)
	}
	log.WithField("queue", q).Debugf("Retrieved queue info. queue_id: %s", q.ID)

	// regenerate or create direct
	var directID uuid.UUID
	var directHash string
	if q.DirectID != uuid.Nil {
		d, err := h.reqHandler.DirectV1DirectRegenerate(ctx, q.DirectID)
		if err != nil {
			log.Errorf("Could not regenerate direct hash. err: %v", err)
			return nil, fmt.Errorf("could not regenerate direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash regenerated. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	} else {
		d, err := h.reqHandler.DirectV1DirectCreate(ctx, q.CustomerID, "queue", id)
		if err != nil {
			log.Errorf("Could not create direct hash. err: %v", err)
			return nil, fmt.Errorf("could not create direct hash: %w", err)
		}
		log.WithField("direct", d).Debugf("Direct hash created. direct_id: %s, hash: %s", d.ID, d.Hash)
		directID = d.ID
		directHash = d.Hash
	}

	// update queue with new direct info
	fields := map[queue.Field]any{
		queue.FieldDirectID:   directID,
		queue.FieldDirectHash: directHash,
	}
	if err := h.db.QueueUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update queue direct hash. err: %v", err)
		return nil, fmt.Errorf("could not update queue: %w", err)
	}

	// return updated queue
	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated queue. err: %v", err)
		return nil, err
	}

	return res, nil
}
