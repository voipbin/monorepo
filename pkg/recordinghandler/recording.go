package recordinghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// Start start the recording of the given reference info
// duration: milliseconds
func (h *recordingHandler) Start(
	ctx context.Context,
	referenceType recording.ReferenceType,
	referenceID uuid.UUID,
	format string,
	endOfSilence int,
	endOfKey string,
	duration int,
) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	switch referenceType {
	case recording.ReferenceTypeCall:
		return h.createReferenceTypeCall(ctx, referenceID, format, endOfSilence, endOfKey, duration)

	case recording.ReferenceTypeConference:
		return h.createReferenceTypeConference(ctx, referenceID, format, endOfSilence, endOfKey, duration)

	default:
		log.Errorf("Unimplemented reference type. reference_type: %s, reference_id: %s", referenceType, referenceID)
		return nil, fmt.Errorf("unsupported reference type")
	}
}

// createReferenceTypeCall creates a new reocording for call type
func (h *recordingHandler) createReferenceTypeCall(
	ctx context.Context,
	referenceID uuid.UUID,
	format string,
	endOfSilence int,
	endOfKey string,
	duration int,
) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "createReferenceTypeCall",
		"reference_type": recording.ReferenceTypeCall,
		"reference_id":   referenceID,
	})

	c, err := h.reqHandler.CallV1CallGet(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get reference info. err: %v", err)
		return nil, err
	}

	if c.Status != call.StatusProgressing {
		log.Errorf("Invalid status. call_id: %s, status: %s", c.ID, c.Status)
		return nil, fmt.Errorf("invalid status")
	}

	id := h.utilHandler.CreateUUID()
	channelIDs := []string{}
	filenames := []string{}

	recordingName := h.createRecordingName(recording.ReferenceTypeCall, c.ID.String())
	for _, direction := range []channel.SnoopDirection{channel.SnoopDirectionIn, channel.SnoopDirectionOut} {
		// filenames
		filename := fmt.Sprintf("%s_%s.%s", recordingName, direction, format)
		filenames = append(filenames, filename)

		// channel ids
		channelID := h.utilHandler.CreateUUID().String()
		channelIDs = append(channelIDs, channelID)

		// set app args
		appArgs := fmt.Sprintf("context=%s,reference_type=%s,reference_id=%s,recording_id=%s,recording_name=%s,direction=%s,format=%s,end_of_silence=%d,end_of_key=%s,duration=%d",
			ContextRecording,
			recording.ReferenceTypeCall,
			c.ID,
			id,
			recordingName,
			direction,
			format,
			endOfSilence,
			endOfKey,
			duration,
		)

		// create a snoop channel
		tmpChannel, err := h.reqHandler.AstChannelCreateSnoop(ctx, c.AsteriskID, c.ChannelID, channelID, appArgs, direction, channel.SnoopDirectionNone)
		if err != nil {
			log.Errorf("Could not create a snoop channel for recroding. err: %v", err)
			return nil, fmt.Errorf("could not create snoop chanel for recrod. err: %v", err)
		}

		log.WithField("channel", tmpChannel).Debugf("Created a snoop channel for recording. channel_id: %s", tmpChannel.ID)
	}

	tmp := &recording.Recording{
		ID:         id,
		CustomerID: c.CustomerID,

		ReferenceType: recording.ReferenceTypeCall,
		ReferenceID:   c.ID,
		Status:        recording.StatusInitiating,
		Format:        format,
		RecordingName: recordingName,
		Filenames:     filenames,

		AsteriskID: c.AsteriskID,
		ChannelIDs: channelIDs,

		TMStart: dbhandler.DefaultTimeStamp,
		TMEnd:   dbhandler.DefaultTimeStamp,
	}

	if errCreate := h.db.RecordingCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create the record. err: %v", errCreate)
		return nil, fmt.Errorf("could not create the record. err: %v", errCreate)
	}

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created reocording. err: %v", err)
		return nil, err
	}

	cc, err := h.reqHandler.CallV1CallSetRecordingID(ctx, res.ReferenceID, res.ID)
	if err != nil {
		log.Errorf("Could not update the call's recording id. err: %v", err)
		return nil, err
	}
	log.WithField("call", cc).Debugf("Updated call's recording id. call_id: %s", cc.ID)

	return res, nil
}

// createReferenceTypeConference creates a new reocording for conference type
func (h *recordingHandler) createReferenceTypeConference(
	ctx context.Context,
	conferenceID uuid.UUID,
	format string,
	endOfSilence int,
	endOfKey string,
	duration int,
) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "createReferenceTypeConference",
		"reference_type": recording.ReferenceTypeCall,
		"reference_id":   conferenceID,
	})
	log.Debugf("Start recording the conference. conference_id: %s", conferenceID)

	cf, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}
	if cf.Status != cmconference.StatusProgressing {
		log.Errorf("Invalid status. conference_id: %s, status: %s", cf.ID, cf.Status)
		return nil, fmt.Errorf("invalid status")
	}

	// get confbridge info
	cb, err := h.confbridgeHandler.Get(ctx, cf.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return nil, err
	}

	// get bridge info
	br, err := h.bridgeHandler.Get(ctx, cb.BridgeID)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return nil, err
	}

	// recreate recording name and filename
	recordingName := h.createRecordingName(recording.ReferenceTypeConference, cf.ID.String())
	filename := fmt.Sprintf("%s_in", recordingName)

	if errRecord := h.reqHandler.AstBridgeRecord(
		ctx,
		br.AsteriskID,
		br.ID,
		filename,
		format,
		duration,
		endOfSilence,
		false,
		endOfKey,
		"fail",
	); errRecord != nil {
		log.Errorf("Could not record the bridge. err: %v", errRecord)
		return nil, errRecord
	}

	id := h.utilHandler.CreateUUID()
	recordingFilename := fmt.Sprintf("%s.%s", filename, format)
	filenames := []string{
		recordingFilename,
	}
	tmp := &recording.Recording{
		ID:         id,
		CustomerID: cf.CustomerID,

		ReferenceType: recording.ReferenceTypeConference,
		ReferenceID:   cf.ID,
		Status:        recording.StatusInitiating,
		Format:        format,
		RecordingName: recordingName,
		Filenames:     filenames,

		AsteriskID: br.AsteriskID,

		TMStart: dbhandler.DefaultTimeStamp,
		TMEnd:   dbhandler.DefaultTimeStamp,
	}

	if errCreate := h.db.RecordingCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create recording. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.RecordingGet(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Could not get created recording. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Started updates recording's status to the recording and notify the event
func (h *recordingHandler) Started(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Started",
		"recording_id": id,
	})
	if errStatus := h.db.RecordingSetStatus(ctx, id, recording.StatusRecording); errStatus != nil {
		log.Errorf("Could not update the recording status. err: %v", errStatus)
		return nil, errStatus
	}

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the updated recording info. err: %v", err)
		return nil, err
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, recording.EventTypeRecordingStarted, res)
	return res, nil
}

// GetsByCustomerID returns list of recordings of the given customerID
func (h *recordingHandler) GetsByCustomerID(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*recording.Recording, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByCustomerID",
			"customer_id": customerID,
		},
	)

	res, err := h.db.RecordingGets(ctx, customerID, size, token)
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

	case recording.ReferenceTypeConference:
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

	if errStop := h.reqHandler.AstRecordingStop(ctx, r.AsteriskID, r.RecordingName); errStop != nil {
		log.Errorf("Could not stop the recording. err: %v", errStop)
		return errStop
	}

	return nil
}

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

	switch res.ReferenceType {
	case recording.ReferenceTypeCall:
		tmp, err := h.reqHandler.CallV1CallSetRecordingID(ctx, res.ReferenceID, uuid.Nil)
		if err != nil {
			log.Errorf("Could not update the call's recording id. call_id: %s, err: %v", res.ReferenceID, err)
			return nil, err
		}
		log.WithField("call", tmp).Debugf("Updated call's recording id. call_id: %s", tmp.ID)

	case recording.ReferenceTypeConference:
		tmp, err := h.reqHandler.ConferenceV1ConferenceUpdateRecordingID(ctx, res.ReferenceID, uuid.Nil)
		if err != nil {
			log.Errorf("Could not update the conference's recording id. conference_id: %s, err: %v", res.ReferenceID, err)
			return nil, err
		}
		log.WithField("conference", tmp).Debugf("Updated conference's recording id. reference_id: %s", tmp.ID)

	default:
		// nothing todo
		log.Infof("Unsupported reference type. reference_type: %s, reference_id: %s", res.ReferenceType, res.ReferenceID)
		return nil, fmt.Errorf("unsupported reference type")
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, recording.EventTypeRecordingFinished, res)

	return res, nil
}

// Delete deletes recording
func (h *recordingHandler) Delete(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Delete",
			"recording_id": id,
		},
	)

	if errDelete := h.db.RecordingDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not get reocording. err: %v", errDelete)
		return nil, errDelete
	}

	go func() {
		// send request to delete recording files
		log.Debugf("Deleting recording files. recording_id: %s", id)
		if errDelete := h.reqHandler.StorageV1RecordingDelete(ctx, id); errDelete != nil {
			log.Errorf("Could not delete the recording files. err: %v", errDelete)
			return
		}
	}()

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted recording. err: %v", err)
		return nil, err
	}

	return res, nil
}
