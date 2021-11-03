package conferencehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/notifyhandler"
)

const defaultConferenceTimeout = 86400

// Create is handy function for creating a conference.
// it increases corresponded counter
func (h *conferenceHandler) Create(
	ctx context.Context,
	conferenceType conference.Type,
	userID uint64,
	name string,
	detail string,
	timeout int,
	webhookURI string,
	preActions []action.Action,
	postActions []action.Action,
) (*conference.Conference, error) {
	log := logrus.New().WithField("func", "Create")

	id := uuid.Must(uuid.NewV4())
	log = log.WithField("confbridge_id", id.String())

	// send confbridge create request
	cb, err := h.reqHandler.CMConfbridgesPost(id)
	if err != nil {
		log.Errorf("Could not crate confbridge. err: %v", err)
		return nil, err
	}
	log.Debugf("Created confbridge. confbridge_id: %s", cb.ID)

	// create flow
	f, err := h.createConferenceFlow(userID, id, preActions, postActions)
	if err != nil {
		log.Errorf("Could not create conference flow. err: %v", err)
		return nil, err
	}
	log.Debugf("Created flow. flow_id: %s", f.ID)

	if timeout == 0 {
		timeout = defaultConferenceTimeout
	}

	// create a conference struct
	newCf := &conference.Conference{
		ID:           id,
		UserID:       userID,
		ConfbridgeID: cb.ID,
		FlowID:       f.ID,
		Type:         conferenceType,
		Status:       conference.StatusProgressing,
		Name:         name,
		Detail:       detail,
		Data:         map[string]interface{}{},
		Timeout:      timeout,

		CallIDs:      []uuid.UUID{},
		RecordingIDs: []uuid.UUID{},
		WebhookURI:   webhookURI,

		TMCreate: getCurTime(),
		TMUpdate: defaultTimeStamp,
		TMDelete: defaultTimeStamp,
	}

	// set timestamp
	newCf.TMCreate = getCurTime()
	newCf.TMUpdate = defaultTimeStamp
	newCf.TMDelete = defaultTimeStamp

	// create a conference record
	if err := h.db.ConferenceCreate(ctx, newCf); err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}
	promConferenceCreateTotal.WithLabelValues(string(newCf.Type)).Inc()

	// get created conference and notify
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created conference. err: %v", err)
		return nil, err
	}
	h.notifyHandler.NotifyEvent(notifyhandler.EventTypeConferenceCreated, cf.WebhookURI, cf)

	// set the timeout if it was set
	if cf.Timeout > 0 {
		if err := h.reqHandler.CFConferencesIDDelete(id, cf.Timeout*1000); err != nil {
			log.Errorf("Could not start conference timeout. err: %v", err)
		}
	}

	return newCf, nil
}

// createConferenceFlowActions creates the actions for conference.
func (h *conferenceHandler) createConferenceFlowActions(conferenceID uuid.UUID, preActions []action.Action, postActions []action.Action) ([]action.Action, error) {
	log := logrus.New().WithField("func", "createConferenceFlow")
	actions := []action.Action{}

	// append the pre actions
	actions = append(actions, preActions...)

	// append the conference enter
	option := action.OptionConferenceEnter{
		ConferenceID: conferenceID.String(),
	}
	opt, err := json.Marshal(option)
	if err != nil {
		log.Errorf("Could not marshal the option. err: %v", err)
		return nil, err
	}

	conferenceEnter := action.Action{
		Type:   action.TypeConferenceEnter,
		Option: opt,
	}
	actions = append(actions, conferenceEnter)

	// append the post actions
	actions = append(actions, postActions...)

	return actions, nil
}

// createConferenceFlow creates a conference flow and returns created flow.
func (h *conferenceHandler) createConferenceFlow(userID uint64, conferenceID uuid.UUID, preActions []action.Action, postActions []action.Action) (*flow.Flow, error) {
	log := logrus.WithField("func", "createConferenceFlow")

	// create flow actions
	actions, err := h.createConferenceFlowActions(conferenceID, preActions, postActions)
	if err != nil {
		log.Errorf("Could not create actions. err: %v", err)
		return nil, err
	}
	log.Debugf("Created flow actions. actions: %v", actions)

	// create flow name
	flowName := fmt.Sprintf("conference-%s", conferenceID.String())

	// create conference flow
	f := &flow.Flow{
		UserID:   userID,
		Name:     flowName,
		Detail:   "generated for conference by conference-manager.",
		Persist:  true,
		Actions:  actions,
		TMCreate: getCurTime(),
		TMUpdate: defaultTimeStamp,
		TMDelete: defaultTimeStamp,
	}

	// create flow
	resFlow, err := h.reqHandler.FMFlowCreate(f)
	if err != nil {
		log.Errorf("Could not create a conference flow. err: %v", err)
		return nil, err
	}
	log.Debugf("Created a conference flow. res: %v", resFlow)

	return resFlow, nil
}
