package notifyhandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

// CallCreate sends the notify for call creation.
func (r *notifyHandler) CallCreate(c *call.Call) {

	log := logrus.WithFields(
		logrus.Fields{
			"call": c,
		},
	)

	m, err := json.Marshal(c)
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	if err := r.publishNotify(eventTypeCallCreate, ContentTypeJSON, m, 0); err != nil {
		log.Errorf("Could not publish the notify. err: %v", err)
		return
	}

	return
}
