package recordinghandler

import (
	"context"
	"fmt"
	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/recording"
	smfile "monorepo/bin-storage-manager/models/file"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Stopped handels stopped recording
func (h *recordingHandler) Stopped(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Stopped",
		"recording_id": id,
	})

	// update recording status
	if errStatus := h.db.RecordingSetStatus(ctx, id, recording.StatusEnded); errStatus != nil {
		log.Errorf("Could not update recording status. err: %v", errStatus)
		return nil, errStatus
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, recording.EventTypeRecordingFinished, res)

	// store the recording
	go h.storeRecordingFiles(res)

	return res, nil
}

// storeRecordingFiles send a request to the storage-manager to store the recording files
// into the customer's storage account.
func (h *recordingHandler) storeRecordingFiles(r *recording.Recording) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "storeRecordingFiles",
		"recording": r,
	})

	// wait for file writing
	// note: The asterisk runs the recording file move script in every minutes.
	// so we have to wait for 2 mins to ensure that the file was moved to the bucket correctly.
	// if anyone wants to change this wait time, please don't forget the change the crontab from the asterisks.
	// asterisk-k8s-call, asterisk-k8s-conference
	//
	// # Set cron - recording move script
	// /bin/mkdir -p /var/spool/asterisk/recording
	// crontab -l | { cat; echo "* * * * * /cron_recording_move.sh"; } | crontab -
	time.Sleep(time.Second * 120)

	log.Debugf("Storing the recording files.")
	for _, recordingFilename := range r.Filenames {
		// store the each recording files
		go func(filename string) {
			log := log.WithField("filename", filename)

			filepath := h.getFilepath(filename)
			f, err := h.reqHandler.StorageV1FileCreate(context.Background(), r.CustomerID, uuid.Nil, smfile.ReferenceTypeRecording, r.ID, "", "", filename, defaultBucketName, filepath, 60000)
			if err != nil {
				log.Errorf("Could not store the recording correctly. err: %v", err)
				return
			}
			log.WithField("file", f).Debugf("Stored recording correctly. storage_file_id: %s", f.ID)

		}(recordingFilename)
	}
}

// Stop stops the recording
func (h *recordingHandler) Stop(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Stop",
		"recording_id": id,
	})

	r, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info. err: %v", err)
		return nil, err
	}

	switch r.ReferenceType {
	case recording.ReferenceTypeCall:
		err = h.stopReferenceTypeCall(ctx, r)

	case recording.ReferenceTypeConfbridge:
		err = h.stopReferenceTypeConference(ctx, r)

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", r.ReferenceType)
		return nil, fmt.Errorf("unsupported reference type")
	}
	if err != nil {
		log.Errorf("Could not stop the recording. reference_type: %s, reference_id: %s", r.ReferenceType, r.ReferenceID)
		return nil, err
	}

	if errStatus := h.db.RecordingSetStatus(ctx, r.ID, recording.StatusStopping); errStatus != nil {
		log.Errorf("Could not update the status. err: %v", errStatus)
		return nil, errStatus
	}

	res, err := h.db.RecordingGet(ctx, r.ID)
	if err != nil {
		log.Errorf("Could not get updated recording info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// stopReferenceTypeCall stops the reference type call recording.
func (h *recordingHandler) stopReferenceTypeCall(ctx context.Context, r *recording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "stopReferenceTypeCall",
		"recording_id": r.ID,
	})

	for _, channelID := range r.ChannelIDs {
		// hangup the channel
		log.WithField("channel_id", channelID).Debugf("Hanging up the recording channel. channel_id: %s", channelID)
		if errHangup := h.reqHandler.AstChannelHangup(ctx, r.AsteriskID, channelID, ari.ChannelCauseNormalClearing, 0); errHangup != nil {
			log.Errorf("Could not hangup the recording channel. err: %v", errHangup)
		}
	}

	return nil
}

// stopReferenceTypeConference stops the reference type conference recording.
func (h *recordingHandler) stopReferenceTypeConference(ctx context.Context, r *recording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "stopReferenceTypeConference",
		"recording_id": r.ID,
	})

	filename := fmt.Sprintf("%s_in", r.RecordingName)
	log.Debugf("Stopping conference recording. recording_name: %s", filename)
	if errStop := h.reqHandler.AstRecordingStop(ctx, r.AsteriskID, filename); errStop != nil {
		log.Errorf("Could not stop the recording. err: %v", errStop)
		return errStop
	}

	return nil
}
