package queuecallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// Gets returns queuecalls
func (h *queuecallHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*queuecall.Queuecall, error) {
	log := logrus.WithField("func", "Gets")

	res, err := h.db.QueuecallGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get queuecalls info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns queuecall info.
func (h *queuecallHandler) Get(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithField("func", "Get")

	res, err := h.db.QueuecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get queuecall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceID returns queuecall info of the given referenceID.
func (h *queuecallHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "GetByReferenceID",
			"reference_id": referenceID,
		})

	qcf, err := h.queuecallReferenceHandler.Get(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get queuecall reference. err: %v", err)
		return nil, err
	}

	if qcf.CurrentQueuecallID == uuid.Nil {
		log.Errorf("No current queuecall info exist.")
		return nil, fmt.Errorf("no current queuecall id info")
	}

	// get current queuecall info
	res, err := h.db.QueuecallGet(ctx, qcf.CurrentQueuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall reference info. err: %v", err)
		return nil, err
	}

	return res, nil
}
