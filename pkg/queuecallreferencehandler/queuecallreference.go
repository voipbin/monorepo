package queuecallreferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// SetCurrentQueuecallID sets the current_queuecall_id to the queueureference.
func (h *queuecallReferenceHandler) SetCurrentQueuecallID(ctx context.Context, referenceID uuid.UUID, queuecallType queuecall.ReferenceType, queuecallID uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":                     "SetCurrentQueuecallID",
			"queuecall_reference_id":   referenceID,
			"queuecall_reference_type": queuecallType,
		},
	)

	qm, err := h.db.QueuecallReferenceGet(ctx, referenceID)
	if err != nil {

		// create a new QueuecallMaster
		qm = &queuecallreference.QueuecallReference{
			ID:           referenceID,
			Type:         queuecallType,
			QueuecallIDs: []uuid.UUID{},

			TMCreate: dbhandler.GetCurTime(),
			TMUpdate: dbhandler.DefaultTimeStamp,
			TMDelete: dbhandler.DefaultTimeStamp,
		}

		if errCreate := h.db.QueuecallReferenceCreate(ctx, qm); errCreate != nil {
			log.Errorf("Could not create the QueuecallReference. err: %v", errCreate)
			return errCreate
		}

	}

	// set variables
	qm.CurrentQueuecallID = queuecallID
	qm.QueuecallIDs = append(qm.QueuecallIDs, queuecallID)

	// set
	if err := h.db.QueuecallReferenceSetCurrentQueuecallID(ctx, referenceID, queuecallID); err != nil {
		log.Errorf("Could not set queuecallreference. err: %v", err)
		return err
	}

	return nil

}

// Get returns queuecallreference.
func (h *queuecallReferenceHandler) Get(ctx context.Context, id uuid.UUID) (*queuecallreference.QueuecallReference, error) {
	qm, err := h.db.QueuecallReferenceGet(ctx, id)
	if err != nil {
		// we don't write the error log here.
		// there's so many case of returning the error here.
		// log.Errorf("Could not get queuecallreference. err: %v", err)
		return nil, err
	}

	return qm, nil
}
