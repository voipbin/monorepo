package transcribehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
)

// Get returns transcribe
func (h *transcribeHandler) Get(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Get",
		"transcribe_id": id,
	})

	res, err := h.db.TranscribeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceIDAndLanguage returns transcribe of the given referenceID and language
func (h *transcribeHandler) GetByReferenceIDAndLanguage(ctx context.Context, referenceID uuid.UUID, language string) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "GetByReferenceIDAndLanguage",
		"reference_id": referenceID,
		"language":     language,
	})

	res, err := h.db.TranscribeGetByReferenceIDAndLanguage(ctx, referenceID, language)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Gets returns list of transcribes.
func (h *transcribeHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
	})

	res, err := h.db.TranscribeGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get transcribes. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new transcribe
func (h *transcribeHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceType transcribe.ReferenceType,
	referenceID uuid.UUID,
	language string,
	direction transcribe.Direction,
	streamingIDs []uuid.UUID,
) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"id":             id,
		"customer_id":    customerID,
		"activeflow_id":  activeflowID,
		"on_end_flow_id": onEndFlowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
		"direction":      direction,
	})

	tmp := &transcribe.Transcribe{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		ActiveflowID: activeflowID,
		OnEndFlowID:  onEndFlowID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Status:    transcribe.StatusProgressing,
		HostID:    h.hostID,
		Language:  language,
		Direction: direction,

		StreamingIDs: streamingIDs,
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

	if errSet := h.variableSet(ctx, activeflowID, res); errSet != nil {
		// we could not set the variable, but we just ignore the error and continue anyway
		log.Errorf("Could not set the variable. err: %v", errSet)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, transcribe.EventTypeTranscribeCreated, res)

	return res, nil
}

// Delete deletes the transcribe
func (h *transcribeHandler) Delete(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Delete",
		"transcribe_id": id,
	})

	// get transcribe
	tr, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get transcribe info. err: %v", err)
		return nil, err
	}

	if tr.TMDelete != dbhandler.DefaultTimeStamp {
		// already deleted
		return tr, nil
	}

	if tr.Status != transcribe.StatusDone {
		// transcribe is ongoing. need to stop the first.
		tmp, err := h.Stop(ctx, tr.ID)
		if err != nil {
			log.Errorf("Could not stop the transcribing. err: %v", err)
			return nil, err
		}
		log.WithField("transcribe", tmp).Debugf("Stopped transcribe. transcribe_id: %s", tr.ID)
	}

	// delete transcripts
	if errDelete := h.deleteTranscripts(ctx, tr.ID); errDelete != nil {
		log.Errorf("Could not delete transcripts. err: %v", errDelete)
		return nil, errDelete
	}

	// delete
	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the transcribe. err: %v", err)
		return nil, err
	}

	return res, nil
}

// deleteTranscripts deletes all transcripts of the give transcribe
func (h *transcribeHandler) deleteTranscripts(ctx context.Context, transcribeID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "deleteTranscripts",
		"transcribe_id": transcribeID,
	})

	// delete all transcripts
	filters := map[string]string{
		"transcribe_id": transcribeID.String(),
		"deleted":       "false",
	}

	ts, err := h.transcriptHandler.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not get transcripts. err: %v", err)
		return err
	}

	for _, t := range ts {
		tmp, err := h.transcriptHandler.Delete(ctx, t.ID)
		if err != nil {
			log.Errorf("Could not delete transcript. err: %v", err)
			// we couldn't delete the transript for some reason. but we just ignore the error and continue anyway
			continue
		}
		log.WithField("transcript", tmp).Debugf("Deleted transcript info. transcript_id: %s", t.ID)
	}

	return nil
}

// UpdateStatus updates the transcribe's status
func (h *transcribeHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status transcribe.Status) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateStatus",
		"transcribe_id": id,
		"status":        status,
	})

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
