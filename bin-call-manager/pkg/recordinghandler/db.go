package recordinghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	smfile "monorepo/bin-storage-manager/models/file"
)

func (h *recordingHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	referenceType recording.ReferenceType,
	referenceID uuid.UUID,
	format recording.Format,
	OnEndFlowID uuid.UUID,
	recordingName string,
	filenames []string,
	asteriskID string,
	channelIDs []string,
) (*recording.Recording, error) {

	tmp := &recording.Recording{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeNone,
			OwnerID:   uuid.Nil,
		},

		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Status:        recording.StatusInitiating,
		Format:        format,
		RecordingName: recordingName,
		Filenames:     filenames,

		OnEndFlowID: OnEndFlowID,

		AsteriskID: asteriskID,
		ChannelIDs: channelIDs,

		TMStart: dbhandler.DefaultTimeStamp,
		TMEnd:   dbhandler.DefaultTimeStamp,
	}

	if errCreate := h.db.RecordingCreate(ctx, tmp); errCreate != nil {
		return nil, errors.Wrapf(errCreate, "could not create the record")
	}

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get created reocording")
	}

	return res, nil
}

// Gets returns list of recordings of the given filters
func (h *recordingHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"filters": filters,
	})

	res, err := h.db.RecordingGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get reocordings. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns the recording info
func (h *recordingHandler) Get(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Get",
		"recording_id": id,
	})

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByRecordingName returns the recording info of the given recording name
func (h *recordingHandler) GetByRecordingName(ctx context.Context, recordingName string) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "GetByRecordingName",
		"recording_name": recordingName,
	})

	res, err := h.db.RecordingGetByRecordingName(ctx, recordingName)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes recording
func (h *recordingHandler) Delete(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Delete",
		"recording_id": id,
	})

	if errDelete := h.db.RecordingDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not get reocording. err: %v", errDelete)
		return nil, errDelete
	}

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted recording. err: %v", err)
		return nil, err
	}

	// delete storage recording files
	go h.deleteRecordingFiles(res)

	return res, nil
}

// Delete deletes recording
func (h *recordingHandler) deleteRecordingFiles(r *recording.Recording) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "deleteRecordingFiles",
		"recording": r,
	})
	ctx := context.Background()

	// get files
	filters := map[string]string{
		"reference_type": string(smfile.ReferenceTypeRecording),
		"reference_id":   r.ID.String(),
		"deleted":        "false",
	}

	files, err := h.reqHandler.StorageV1FileGets(ctx, "", 1000, filters)
	if err != nil {
		log.Errorf("Could not get recording files. err: %v", err)
		return
	}

	for _, file := range files {
		f, err := h.reqHandler.StorageV1FileDelete(ctx, file.ID, 60000)
		if err != nil {
			log.Errorf("Could not delete the recording file. err: %v", err)
			continue
		}
		log.WithField("file", f).Debugf("Deleted storage file. file_id: %s, filename: %s", f.ID, f.Filename)
	}
}
