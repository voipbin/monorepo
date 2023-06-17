package recordinghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// recordingReferenceTypeCall creates a new reocording for call type
func (h *recordingHandler) recordingReferenceTypeCall(
	ctx context.Context,
	referenceID uuid.UUID,
	format recording.Format,
	endOfSilence int,
	endOfKey string,
	duration int,
) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "recordingReferenceTypeCall",
		"reference_id": referenceID,
		"format":       format,
		"endOfSilence": endOfSilence,
		"endOfKey":     endOfKey,
		"duration":     duration,
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

	id := h.utilHandler.UUIDCreate()
	channelIDs := []string{}
	filenames := []string{}

	recordingName := h.createRecordingName(recording.ReferenceTypeCall, c.ID.String())
	asteriskID := ""
	for _, direction := range []channel.SnoopDirection{channel.SnoopDirectionIn, channel.SnoopDirectionOut} {
		// filenames
		filename := fmt.Sprintf("%s_%s.%s", recordingName, direction, format)
		filenames = append(filenames, filename)

		// channel ids
		channelID := h.utilHandler.UUIDCreate().String()
		channelIDs = append(channelIDs, channelID)

		// set app args
		appArgs := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s,%s=%s,%s=%s,%s=%s,%s=%s,%s=%d,%s=%s,%s=%d",
			channel.StasisDataTypeContextType, channel.TypeCall,
			channel.StasisDataTypeContext, channel.ContextRecording,
			channel.StasisDataTypeReferenceType, recording.ReferenceTypeCall,
			channel.StasisDataTypeReferenceID, c.ID,
			channel.StasisDataTypeRecordingID, id,
			channel.StasisDataTypeRecordingName, recordingName,
			channel.StasisDataTypeRecordingDirection, direction,
			channel.StasisDataTypeRecordingFormat, format,
			channel.StasisDataTypeRecordingEndOfSilence, endOfSilence,
			channel.StasisDataTypeRecordingEndOfKey, endOfKey,
			channel.StasisDataTypeRecordingDuration, duration,
		)

		// create a snoop channel
		tmpChannel, err := h.channelHandler.StartSnoop(ctx, c.ChannelID, channelID, appArgs, direction, channel.SnoopDirectionNone)
		if err != nil {
			log.Errorf("Could not create a snoop channel for recroding. err: %v", err)
			return nil, fmt.Errorf("could not create snoop chanel for recrod. err: %v", err)
		}

		log.WithField("channel", tmpChannel).Debugf("Created a snoop channel for recording. channel_id: %s", tmpChannel.ID)
		asteriskID = tmpChannel.AsteriskID
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

		AsteriskID: asteriskID,
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

	return res, nil
}

// recordingReferenceTypeConfbridge creates a new reocording for conference type
func (h *recordingHandler) recordingReferenceTypeConfbridge(
	ctx context.Context,
	confbridgeID uuid.UUID,
	format recording.Format,
	endOfSilence int,
	endOfKey string,
	duration int,
) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "recordingReferenceTypeConfbridge",
		"reference_id":   confbridgeID,
		"format":         format,
		"end_of_silence": endOfSilence,
		"end_of_key":     endOfKey,
		"duration":       duration,
	})
	log.Debugf("Start recording the confbridge. confbridge_id: %s", confbridgeID)

	// get confbridge info
	cb, err := h.reqHandler.CallV1ConfbridgeGet(ctx, confbridgeID)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return nil, err
	}

	if cb.TMDelete < dbhandler.DefaultTimeStamp {
		log.Errorf("Invalid confbridge. confbridge_id: %s", confbridgeID)
		return nil, fmt.Errorf("invalid confbridge")
	}

	// get bridge info
	br, err := h.bridgeHandler.Get(ctx, cb.BridgeID)
	if err != nil {
		log.Errorf("Could not get bridge info. err: %v", err)
		return nil, err
	}

	// recreate recording name and filename
	recordingName := h.createRecordingName(recording.ReferenceTypeConfbridge, cb.ID.String())
	filename := fmt.Sprintf("%s_in", recordingName)

	id := h.utilHandler.UUIDCreate()
	recordingFilename := fmt.Sprintf("%s.%s", filename, format)
	filenames := []string{
		recordingFilename,
	}
	tmp := &recording.Recording{
		ID:         id,
		CustomerID: cb.CustomerID,

		ReferenceType: recording.ReferenceTypeConfbridge,
		ReferenceID:   cb.ID,
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

	// send recording request
	if errRecord := h.reqHandler.AstBridgeRecord(
		ctx,
		br.AsteriskID,
		br.ID,
		filename,
		string(format),
		duration,
		endOfSilence,
		false,
		endOfKey,
		"fail",
	); errRecord != nil {
		log.Errorf("Could not record the bridge. err: %v", errRecord)
		return nil, errRecord
	}

	return res, nil
}
