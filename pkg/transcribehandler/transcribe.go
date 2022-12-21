package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// Get returns transcribe
func (h *transcribeHandler) Get(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Get",
			"transcribe_id": id,
		},
	)

	res, err := h.db.TranscribeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceIDAndLanguage returns transcribe of the given referenceID and language
func (h *transcribeHandler) GetByReferenceIDAndLanguage(ctx context.Context, referenceID uuid.UUID, language string) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "GetByReferenceIDAndLanguage",
			"reference_id": referenceID,
			"language":     language,
		},
	)

	res, err := h.db.TranscribeGetByReferenceIDAndLanguage(ctx, referenceID, language)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Gets returns list of transcribes.
func (h *transcribeHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Gets",
			"customer_id": customerID,
		},
	)

	res, err := h.db.TranscribeGetsByCustomerID(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get transcribes. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new transcribe
func (h *transcribeHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType transcribe.ReferenceType,
	referenceID uuid.UUID,
	language string,
	direction transcribe.Direction,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Create",
			"type":         referenceType,
			"reference_id": referenceID,
		},
	)

	id := h.utilHandler.CreateUUID()
	tmp := &transcribe.Transcribe{
		ID:            id,
		CustomerID:    customerID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Status:    transcribe.StatusProgressing,
		HostID:    h.hostID,
		Language:  language,
		Direction: direction,
	}

	if err := h.db.TranscribeCreate(ctx, tmp); err != nil {
		log.Errorf("Could not create transcribe. err: %v", err)
		return nil, err
	}
	log.WithField("transcribe", tmp).Debugf("Created a new transcribe. transcribe_id: %s, reference_id: %s", tmp.ID, tmp.ReferenceID)

	res, err := h.db.TranscribeGet(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Could not get created transcribe. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, transcribe.EventTypeTranscribeCreated, res)

	return res, nil
}

// TranscribeGet returns transcribe
func (h *transcribeHandler) Delete(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Delete",
		},
	)

	if errDelete := h.db.TranscribeDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the transcribe info. err: %v", errDelete)
		return nil, errDelete
	}

	// get deleted item
	res, err := h.db.TranscribeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted transcribe. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, transcribe.EventTypeTranscribeDeleted, res)

	return res, nil

}

// UpdateStatus updates the transcribe's status
func (h *transcribeHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status transcribe.Status) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "UpdateStatus",
			"transcribe_id": id,
			"status":        status,
		},
	)

	// // get transcribe and evaluate
	// tmp, err := h.Get(ctx, id)
	// if err != nil {
	// 	log.Errorf("Could not get transcribe. err: %v", err)
	// 	return nil, errors.Wrap(err, "could not get transcribe info")
	// }

	// if !transcribe.IsUpdatableStatus(tmp.Status, status) {
	// 	log.Errorf("Invalid status. old_status: %s, new_status: %s", tmp.Status, status)
	// 	return nil, fmt.Errorf("invalid status")
	// }

	if errSet := h.db.TranscribeSetStatus(ctx, id, status); errSet != nil {
		log.Errorf("Could not delete the transcribe info. err: %v", errSet)
		return nil, errSet
	}

	// get updated item
	res, err := h.db.TranscribeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted transcribe. err: %v", err)
		return nil, err
	}

	switch status {
	case transcribe.StatusProgressing:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, transcribe.EventTypeTranscribeProgressing, res)

	case transcribe.StatusDone:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, transcribe.EventTypeTranscribeDone, res)
	}

	return res, nil

}
