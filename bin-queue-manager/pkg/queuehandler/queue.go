package queuehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
)

// Delete updates the queue's basic info.
func (h *queueHandler) Delete(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Delete",
		"queue_id": id,
	})
	log.Debug("Deleting the queue info.")

	if err := h.db.QueueSetExecute(ctx, id, queue.ExecuteStop); err != nil {
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
