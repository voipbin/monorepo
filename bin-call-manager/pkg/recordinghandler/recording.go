package recordinghandler

import (
	"context"
	"fmt"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// recordingReferenceTypeCall creates a new reocording for call type
func (h *recordingHandler) recordingReferenceTypeCall(
	ctx context.Context,
	referenceID uuid.UUID,
	format recording.Format,
	endOfSilence int,
	endOfKey string,
	duration int,
	onEndFlowID uuid.UUID,
) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "recordingReferenceTypeCall",
		"reference_id":   referenceID,
		"format":         format,
		"endOfSilence":   endOfSilence,
		"endOfKey":       endOfKey,
		"duration":       duration,
		"on_end_flow_iD": onEndFlowID,
	})

	c, err := h.reqHandler.CallV1CallGet(ctx, referenceID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get call info")
	}

	if c.Status != call.StatusProgressing {
		return nil, fmt.Errorf("invalid status. call_id: %s, status: %s", c.ID, c.Status)
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
			return nil, errors.Wrapf(err, "could not create a snoop channel for recording")
		}
		log.WithField("channel", tmpChannel).Debugf("Created a snoop channel for recording. channel_id: %s", tmpChannel.ID)
		asteriskID = tmpChannel.AsteriskID
	}

	res, err := h.Create(
		ctx,
		id,
		c.CustomerID,
		recording.ReferenceTypeCall,
		c.ID,
		format,
		onEndFlowID,
		recordingName,
		filenames,
		asteriskID,
		channelIDs,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the record")
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
	onEndFlowID uuid.UUID,
) (*recording.Recording, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "recordingReferenceTypeConfbridge",
		"reference_id":   confbridgeID,
		"format":         format,
		"end_of_silence": endOfSilence,
		"end_of_key":     endOfKey,
		"duration":       duration,
		"on_end_flow_iD": onEndFlowID,
	})
	log.Debugf("Start recording the confbridge. confbridge_id: %s", confbridgeID)

	// get confbridge info
	cb, err := h.reqHandler.CallV1ConfbridgeGet(ctx, confbridgeID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get confbridge info")
	}

	if cb.TMDelete < dbhandler.DefaultTimeStamp {
		return nil, fmt.Errorf("invalid confbridge. confbridge_id: %s", confbridgeID)
	}

	// get bridge info
	br, err := h.bridgeHandler.Get(ctx, cb.BridgeID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get bridge info")
	}

	// recreate recording name and filename
	recordingName := h.createRecordingName(recording.ReferenceTypeConfbridge, cb.ID.String())
	filename := fmt.Sprintf("%s_in", recordingName)

	id := h.utilHandler.UUIDCreate()
	recordingFilename := fmt.Sprintf("%s.%s", filename, format)
	filenames := []string{
		recordingFilename,
	}

	res, err := h.Create(
		ctx,
		id,
		cb.CustomerID,
		recording.ReferenceTypeConfbridge,
		cb.ID,
		format,
		onEndFlowID,
		recordingName,
		filenames,
		br.AsteriskID,
		[]string{},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the record")
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
		return nil, errors.Wrapf(errRecord, "could not send the recording request")
	}

	return res, nil
}
