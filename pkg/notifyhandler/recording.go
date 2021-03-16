package notifyhandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

func (n *notifyHandler) recordingCommon(t eventType, r *recording.Recording) {
	log := logrus.WithFields(
		logrus.Fields{
			"recording":  r,
			"evnet_type": t,
		},
	)
	log.Debugf("Sending recording event. event_type: %s, recording: %s, recording_type: %s, recording_reference_id: %s", t, r.ID, r.Type, r.ReferenceID)

	m, err := json.Marshal(r)
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	if err := n.publishNotify(t, dataTypeJSON, m, requestTimeoutDefault); err != nil {
		log.Errorf("Could not publish the notify. err: %v", err)
		return
	}

	return
}

// RecordingStarted sends the notify for recording started.
func (n *notifyHandler) RecordingStarted(r *recording.Recording) {

	n.recordingCommon(eventTypeRecordingStarted, r)
	return
}

// RecordingFinished sends the notify for recording finished.
func (n *notifyHandler) RecordingFinished(r *recording.Recording) {

	n.recordingCommon(eventTypeRecordingFinished, r)
	return
}
