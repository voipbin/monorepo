package notifyhandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

func (r *notifyHandler) callCommon(t eventType, c *call.Call) {
	log := logrus.WithFields(
		logrus.Fields{
			"call":       c,
			"evnet_type": t,
		},
	)
	log.Debugf("Sending call event. event_type: %s, call: %s", t, c.ID)

	m, err := json.Marshal(c)
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	if err := r.publishNotify(t, dataTypeJSON, m, requestTimeoutDefault); err != nil {
		log.Errorf("Could not publish the notify. err: %v", err)
		return
	}

	return
}

// CallCreated sends the notify for call creation.
func (r *notifyHandler) CallCreated(c *call.Call) {

	r.callCommon(eventTypeCallCreated, c)
	return
}

// CallUpdated sends the notify for call update.
func (r *notifyHandler) CallUpdated(c *call.Call) {
	r.callCommon(eventTypeCallUpdated, c)

	return
}

// CallHungup sends the notify for call hangup.
func (r *notifyHandler) CallHungup(c *call.Call) {
	r.callCommon(eventTypeCallHungup, c)

	return
}
