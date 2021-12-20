package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// Gets returns queuecalls
func (h *queuecallHandler) Gets(ctx context.Context, userID, size uint64, token string) ([]*queuecall.Queuecall, error) {
	log := logrus.WithField("func", "Gets")

	res, err := h.db.QueuecallGets(ctx, userID, size, token)
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
