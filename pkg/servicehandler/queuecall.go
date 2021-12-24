package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
)

// queuecallGet validates the queuecall's ownership and returns the queuecall info.
func (h *serviceHandler) queuecallGet(ctx context.Context, u *user.User, id uuid.UUID) (*qmqueuecall.Event, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "queuecallGet",
			"user_id":  u.ID,
			"agent_id": id,
		},
	)

	// send request
	tmp, err := h.reqHandler.QMV1QueuecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the queuecall info. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debug("Received result.")

	if u.Permission != user.PermissionAdmin && u.ID != tmp.UserID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertEvent()
	return res, nil
}

// QueuecallGet sends a request to queue-manager
// to getting the queuecall.
func (h *serviceHandler) QueuecallGet(u *user.User, queueID uuid.UUID) (*qmqueuecall.Event, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "QueueGet",
		"user_id":  u.ID,
		"username": u.Username,
		"agent_id": queueID,
	})

	res, err := h.queuecallGet(ctx, u, queueID)
	if err != nil {
		log.Errorf("Could not validate the queue info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// QueuecallGets sends a request to queue-manager
// to getting a list of queuecalls.
// it returns queuecall info if it succeed.
func (h *serviceHandler) QueuecallGets(u *user.User, size uint64, token string) ([]*qmqueuecall.Event, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "QueuecallGets",
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    token,
	})

	if token == "" {
		token = getCurTime()
	}

	tmps, err := h.reqHandler.QMV1QueuecallGets(ctx, u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get queues from the queue-manager. err: %v", err)
		return nil, err
	}

	res := []*qmqueuecall.Event{}
	for _, tmp := range tmps {
		e := tmp.ConvertEvent()
		res = append(res, e)
	}

	return res, nil
}

// QueuecallDelete sends a request to the queue-manager
// to kick out the given queuecall.
func (h *serviceHandler) QueuecallDelete(u *user.User, queuecallID uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "QueuecallDelete",
		"user":     u.ID,
		"username": u.Username,
	})

	_, err := h.queuecallGet(ctx, u, queuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. err: %v", err)
		return err
	}

	if err := h.reqHandler.QMV1QueuecallDelete(ctx, queuecallID); err != nil {
		log.Errorf("Could not delete the queuecall. err: %v", err)
		return err
	}

	return nil
}

// QueuecallDeleteByReferenceID sends a request to the queue-manager
// to kick out the given reference id's queuecall.
func (h *serviceHandler) QueuecallDeleteByReferenceID(u *user.User, referenceID uuid.UUID) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":     "QueuecallDeleteByReferenceID",
		"user":     u.ID,
		"username": u.Username,
	})

	tmp, err := h.reqHandler.QMV1QueuecallReferenceGet(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get queuecall reference. err: %v", err)
		return err
	}

	if u.Permission != user.PermissionAdmin && u.ID != tmp.UserID {
		log.Info("The user has no permission for this agent.")
		return fmt.Errorf("user has no permission")
	}

	if tmp.CurrentQueuecallID == uuid.Nil {
		log.Errorf("The current queuecall id has not set. current_queuecall_id: %s", tmp.CurrentQueuecallID)
		return fmt.Errorf("current queuecall id has not set")
	}

	if err := h.reqHandler.QMV1QueuecallDeleteByReferenceID(ctx, tmp.CurrentQueuecallID); err != nil {
		log.Errorf("Could not delete the queuecall. err: %v", err)
		return err
	}

	return nil

}
