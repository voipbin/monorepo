package recordinghandler

import (
	"context"
	"fmt"
	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-flow-manager/models/activeflow"
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
	go h.stopped(res)

	return res, nil
}

// stopped handels stopped recording
// store the recording files and execute the new activeflow if the on_end_flow_id is not empty
func (h *recordingHandler) stopped(r *recording.Recording) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "stopped",
		"recording_id": r.ID,
	})
	ctx := context.Background()

	// store the recording files
	h.storeRecordingFiles(r)

	if r.OnEndFlowID == uuid.Nil {
		return
	}

	time.Sleep(time.Minute * 3) // wait for until the storing the recording files

	log.Debugf("The on_end_flow_id is not empty. Executing the new activeflow. on_end_flow_id: %s", r.OnEndFlowID)
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, uuid.Nil, r.CustomerID, r.OnEndFlowID, activeflow.ReferenceTypeNone, uuid.Nil)
	if err != nil {
		log.Errorf("Could not create the activeflow. err: %v", err)
		return
	}
	log = log.WithField("activeflow", af)
	log.Debugf("Created a new activeflow. activeflow_id: %s", af.ID)

	if errUpdate := h.variableUpdateFromReference(ctx, r, af.ID); errUpdate != nil {
		// if the variable update is failed, but just log the error and continue the flow
		log.Errorf("Could not update the variable from the reference. err: %v", errUpdate)
	}

	if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); errExecute != nil {
		log.Errorf("Could not execute the activeflow. err: %v", errExecute)
		return
	}
}

// storeRecordingFiles send a request to the storage-manager to store the recording files
// into the customer's storage account.
func (h *recordingHandler) storeRecordingFiles(r *recording.Recording) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "storeRecordingFiles",
		"recording": r,
	})

	// store the each recording files
	log.Debugf("Storing the recording files.")
	for _, filename := range r.Filenames {
		log := log.WithField("filename", filename)

		// send delay request wait for file writing
		// note: The asterisk runs the recording file move script in every minutes.
		// so we have to wait for 2 mins to ensure that the file was moved to the bucket correctly.
		// if anyone wants to change this wait time, please don't forget the change the crontab from the asterisks.
		// asterisk-k8s-call, asterisk-k8s-conference
		//
		// # Set cron - recording move script
		// /bin/mkdir -p /var/spool/asterisk/recording
		// crontab -l | { cat; echo "* * * * * /cron_recording_move.sh"; } | crontab -
		delay := requesthandler.DelayMinute * 2

		filepath := h.getFilepath(filename)
		if err := h.reqHandler.StorageV1FileCreateWithDelay(context.Background(), r.CustomerID, uuid.Nil, smfile.ReferenceTypeRecording, r.ID, "", "", filename, defaultBucketName, filepath, delay); err != nil {
			log.Errorf("Could not send the request for the storing the recording correctly. err: %v", err)
			return
		}
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
