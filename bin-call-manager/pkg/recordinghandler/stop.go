package recordinghandler

import (
	"context"
	"fmt"
	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-flow-manager/models/activeflow"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
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
		return nil, errors.Wrapf(err, "Could not get recording info")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, recording.EventTypeRecordingFinished, res)

	// store the recording
	if errStopped := h.stopped(ctx, res); errStopped != nil {
		return nil, errors.Wrapf(errStopped, "Could not handle the stopped recording")
	}

	return res, nil
}

// stopped handels stopped recording
// store the recording files and execute the new activeflow if the on_end_flow_id is not empty
func (h *recordingHandler) stopped(ctx context.Context, r *recording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "stopped",
		"recording_id": r.ID,
	})

	// store the recording files
	if errFile := h.storeRecordingFiles(ctx, r); errFile != nil {
		return errors.Wrapf(errFile, "Could not store the recording files")
	}

	if r.OnEndFlowID == uuid.Nil {
		// has no on_end_flow_id. nothing to do
		return nil
	}

	log.Debugf("The on_end_flow_id is not empty. Executing the new activeflow. on_end_flow_id: %s", r.OnEndFlowID)
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, uuid.Nil, r.CustomerID, r.OnEndFlowID, activeflow.ReferenceTypeNone, uuid.Nil, r.ActiveflowID)
	if err != nil {
		return errors.Wrapf(err, "Could not create the activeflow")
	}
	log = log.WithField("activeflow", af)
	log.Debugf("Created a new activeflow. activeflow_id: %s", af.ID)

	if errSet := h.variablesSet(ctx, af.ID, r); errSet != nil {
		// if the variable set is failed, but just log the error and continue the flow
		log.Errorf("Could not set the variables. err: %v", errSet)
	}

	if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); errExecute != nil {
		return errors.Wrapf(errExecute, "Could not execute the activeflow")
	}

	return nil
}

// storeRecordingFiles send a request to the storage-manager to store the recording files
// into the customer's storage account.
func (h *recordingHandler) storeRecordingFiles(ctx context.Context, r *recording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "storeRecordingFiles",
		"recording": r,
	})

	if errMove := h.reqHandler.AstProxyRecordingFileMove(ctx, r.AsteriskID, r.Filenames); errMove != nil {
		return errors.Wrapf(errMove, "Could not move the recording files. err: %v", errMove)
	}

	for _, filename := range r.Filenames {

		filepath := h.getFilepath(filename)
		tmp, err := h.reqHandler.StorageV1FileCreate(ctx, r.CustomerID, uuid.Nil, smfile.ReferenceTypeRecording, r.ID, "recording file", "", filename, defaultBucketName, filepath, 30000)
		if err != nil {
			log.Errorf("Could not send the request for the storing the recording correctly. err: %v", err)
			return err
		}
		log.WithField("storage_file", tmp).Debugf("Recording file stored. filename: %s, filepath: %s", filename, filepath)
	}

	return nil
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
