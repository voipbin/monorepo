package conferencehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/notifyhandler"
)

// Create is handy function for creating a conference.
// it increases corresponded counter
func (h *conferenceHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	timeout int,
	webhookURI string,
	preActions []action.Action,
	postActions []action.Action,
) (*conference.Conference, error) {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":          "Update",
			"conference_id": id,
		},
	)
	log.Debugf("Updating the conference. name: %s, detail: %s, timeout: %d, webhookURI: %s, pre_actions: %v, post_actions: %v",
		name, detail, timeout, webhookURI, preActions, postActions)

	// get conference
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		return nil, err
	}

	// create flow actions
	actions, err := h.createConferenceFlowActions(cf.ConfbridgeID, preActions, postActions)
	if err != nil {
		log.Errorf("Could not create actions. err: %v", err)
		return nil, err
	}
	log.Debugf("Created flow actions. actions: %v", actions)

	// get flow
	f, err := h.reqHandler.FMFlowGet(cf.FlowID)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return nil, err
	}
	f.Actions = actions

	// update flow
	newFlow, err := h.reqHandler.FMFlowUpdate(f)
	if err != nil {
		log.Errorf("Could not update the flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", newFlow).Debugf("Updated the flow.")

	if timeout > 0 && timeout < 60 {
		timeout = defaultConferenceTimeout
	}

	// update conference
	if errSet := h.db.ConferenceSet(ctx, id, name, detail, timeout, webhookURI, preActions, postActions); errSet != nil {
		log.Errorf("Could not update the conference. err: %v", errSet)
		return nil, err
	}

	// get updated conference and notify
	newConf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conference. err: %v", err)
		return nil, err
	}
	h.notifyHandler.NotifyEvent(notifyhandler.EventTypeConferenceUpdated, newConf.WebhookURI, newConf)

	// set the timeout if it was set
	if cf.Timeout > 0 {
		if err := h.reqHandler.CFConferencesIDDelete(id, cf.Timeout*1000); err != nil {
			log.Errorf("Could not start conference timeout. err: %v", err)
		}
	}

	return newConf, nil
}
