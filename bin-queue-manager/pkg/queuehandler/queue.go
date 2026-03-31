package queuehandler

import (
	"context"

	"monorepo/bin-queue-manager/models/queue"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Delete updates the queue's basic info.
func (h *queueHandler) Delete(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Delete",
		"queue_id": id,
	})
	log.Debug("Deleting the queue info.")

	// fetch queue to get direct_id for cleanup
	q, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue info. err: %v", err)
		return nil, err
	}

	// delete direct hash via direct-manager (best-effort, don't block queue deletion)
	if q.DirectID != uuid.Nil {
		if _, errDirect := h.reqHandler.DirectV1DirectDelete(ctx, q.DirectID); errDirect != nil {
			log.Errorf("Could not delete direct hash. direct_id: %s, err: %v", q.DirectID, errDirect)
		}
	}

	// Update execute to stop using the generic Update method
	fields := map[queue.Field]any{
		queue.FieldExecute: queue.ExecuteStop,
	}
	if err := h.db.QueueUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the queue execute to stop. err: %v", err)
		return nil, err
	}

	// get all queuecalls
	// todo: kick out all queueucalls.

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the queue info. err: %v", err)
		return nil, err
	}

	return res, nil
}
