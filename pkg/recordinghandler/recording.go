package recordinghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

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
		tmp, err := h.reqHandler.CallV1CallGet(ctx, referenceID)
		if err != nil {
			log.Errorf("Could not get reference info. err: %v", err)
			return nil, err
		}

		return h.createReferenceTypeCall(ctx, tmp, format, endOfSilence, endOfKey, duration)

	default:
		log.Errorf("Unimplemented reference type. reference_type: %s, reference_id: %s", referenceType, referenceID)
		return nil, fmt.Errorf("unsupported reference type")
	}
}

// createReferenceTypeCall creates a new reocording for call type
func (h *recordingHandler) createReferenceTypeCall(
	ctx context.Context,
	c *call.Call,
	format string,
	endOfSilence int,
	endOfKey string,
	duration int,
) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"reference_type": recording.ReferenceTypeCall,
		"reference_id":   c.ID,
	})

	id := h.utilHandler.CreateUUID()
	channelIDs := []string{}
	filenames := []string{}
	ts := h.utilHandler.GetCurTimeRFC3339()
	recordingName := fmt.Sprintf("%s_%s_%s", recording.ReferenceTypeCall, c.ID, ts)
	for _, direction := range []channel.SnoopDirection{channel.SnoopDirectionIn, channel.SnoopDirectionOut} {
		// filenames
		filename := fmt.Sprintf("%s_%s.%s", recordingName, direction, format)
		filenames = append(filenames, filename)

		// channel ids
		channelID := h.utilHandler.CreateUUID().String()
		channelIDs = append(channelIDs, channelID)

		// set app args
		appArgs := fmt.Sprintf("context=%s,call_id=%s,recording_id=%s,recording_name=%s,direction=%s,format=%s,end_of_silence=%d,end_of_key=%s,duration=%d",
			ContextRecording,
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

	if err := h.db.RecordingCreate(ctx, tmp); err != nil {
		log.Errorf("Could not create the record. err: %v", err)
		return nil, fmt.Errorf("could not create the record. err: %v", err)
	}

	res, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created reocordings. err: %v", err)
		return nil, err
	}

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

	log.WithField("recording", r).Debugf("Found recording info. recording_id: %s", r.ID)
	for _, channelID := range r.ChannelIDs {
		// hangup the channel
		log.WithField("channel_id", channelID).Debugf("Hanging up the recording channel. channel_id: %s", channelID)
		if errHangup := h.reqHandler.AstChannelHangup(ctx, r.AsteriskID, channelID, ari.ChannelCauseNormalClearing, 0); errHangup != nil {
			log.Errorf("Could not hangup the recording channel. err: %v", errHangup)
		}
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated recording info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Stop stops the recording
func (h *recordingHandler) Stopped(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Stop",
		"recording_id": id,
	})

	// update recording status
	if errStatus := h.db.RecordingSetStatus(ctx, id, recording.StatusEnd, h.utilHandler.GetCurTime()); errStatus != nil {
		log.Errorf("Could not update recording status. err: %v", errStatus)
		return nil, errStatus
	}

	// send call request to set recording id to empty
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated recording info. err: %v", err)
		return nil, err
	}

	return res, nil

	// r, err := h.Get(ctx, id)
	// if err != nil {
	// 	log.Errorf("Could not get recording info. err: %v", err)
	// 	return nil, err
	// }

	// // update recording_id

	// log.WithField("recording", r).Debugf("Found recording info. recording_id: %s", r.ID)
	// for _, channelID := range r.ChannelIDs {
	// 	// hangup the channel
	// 	log.WithField("channel_id", channelID).Debugf("Hanging up the recording channel. channel_id: %s", channelID)
	// 	if errHangup := h.reqHandler.AstChannelHangup(ctx, r.AsteriskID, channelID, ari.ChannelCauseNormalClearing, 0); errHangup != nil {
	// 		log.Errorf("Could not hangup the recording channel. err: %v", errHangup)
	// 	}
	// }

	// // update recording status
	// if errStatus := h.db.RecordingSetStatus(ctx, id, recording.StatusEnd, h.utilHandler.GetCurTime()); errStatus != nil {
	// 	log.Errorf("Could not update recording status. err: %v", errStatus)
	// 	return nil, errStatus
	// }

	// res, err := h.Get(ctx, id)
	// if err != nil {
	// 	log.Errorf("Could not get updated recording info. err: %v", err)
	// 	return nil, err
	// }

	// return res, nil
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
