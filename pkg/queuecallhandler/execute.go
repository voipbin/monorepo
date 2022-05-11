package queuecallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// Execute starts the queuecall agent search.
func (h *queuecallHandler) Execute(ctx context.Context, queuecallID uuid.UUID, delay int) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Execute",
			"queuecall_id": queuecallID,
		},
	)

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, queuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return nil, err
	}
	log.WithField("queuecall", qc).Debug("Found queuecall info.")

	// check the status
	if qc.Status != queuecall.StatusWaiting {
		// no need to execute anymore.
		log.Errorf("The queuecall status is not wait. Will handle in other place. status: %v", qc.Status)
		return nil, fmt.Errorf("invalid queuecall status. status: %s", qc.Status)
	}

	// send agent search request
	if err := h.reqHandler.QMV1QueuecallSearchAgent(ctx, queuecallID, delay); err != nil {
		log.Errorf("Could not send queuecall agent search request. err: %v", err)
		return nil, err
	}

	return qc, nil
}
