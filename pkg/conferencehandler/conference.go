package conferencehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferenceconfbridge"
)

const defaultConferenceTimeout = 86400

// Create is handy function for creating a conference.
// it increases corresponded counter
func (h *conferenceHandler) Create(
	ctx context.Context,
	conferenceType conference.Type,
	customerID uuid.UUID,
	name string,
	detail string,
	timeout int,
	preActions []fmaction.Action,
	postActions []fmaction.Action,
) (*conference.Conference, error) {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":            "Create",
			"customer_id":     customerID,
			"conference_type": conferenceType,
		},
	)

	id := uuid.Must(uuid.NewV4())
	log = log.WithField("confbridge_id", id.String())

	// send confbridge create request
	cb, err := h.reqHandler.CMV1ConfbridgeCreate(ctx)
	if err != nil {
		log.Errorf("Could not crate confbridge. err: %v", err)
		return nil, err
	}
	log.Debugf("Created confbridge. confbridge_id: %s", cb.ID)

	// create flow
	f, err := h.createConferenceFlow(ctx, customerID, id, cb.ID, preActions, postActions)
	if err != nil {
		log.Errorf("Could not create conference flow. err: %v", err)
		return nil, err
	}
	log.Debugf("Created flow. flow_id: %s", f.ID)

	if timeout > 0 && timeout < 60 {
		timeout = defaultConferenceTimeout
	}

	// create a conference struct
	newCf := &conference.Conference{
		ID:           id,
		CustomerID:   customerID,
		ConfbridgeID: cb.ID,
		FlowID:       f.ID,
		Type:         conferenceType,
		Status:       conference.StatusProgressing,
		Name:         name,
		Detail:       detail,
		Data:         map[string]interface{}{},
		Timeout:      timeout,

		PreActions:  preActions,
		PostActions: postActions,

		CallIDs:      []uuid.UUID{},
		RecordingIDs: []uuid.UUID{},

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

	// create conference-confbridge
	confCb := &conferenceconfbridge.ConferenceConfbridge{
		ConferenceID: newCf.ID,
		ConfbridgeID: cb.ID,
	}
	if err := h.db.ConferenceConfbridgeSet(ctx, confCb); err != nil {
		log.Errorf("Could not set conference-confbridge. err: %v", err)
		return nil, err
	}

	// get created conference and notify
	cf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created conference. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, cf.CustomerID, conference.EventTypeConferenceCreated, cf)

	// set the timeout if it was set
	if cf.Timeout > 0 {
		if err := h.reqHandler.CFV1ConferenceDeleteDelay(ctx, id, cf.Timeout*1000); err != nil {
			log.Errorf("Could not start conference timeout. err: %v", err)
		}
	}

	return cf, nil
}

// createConferenceFlowActions creates the actions for conference join.
func (h *conferenceHandler) createConferenceFlowActions(confbridgeID uuid.UUID, preActions []fmaction.Action, postActions []fmaction.Action) ([]fmaction.Action, error) {
	log := logrus.New().WithField("func", "createConferenceFlow")
	actions := []fmaction.Action{}

	// append the pre actions
	actions = append(actions, preActions...)

	// append the confbridge join
	option := fmaction.OptionConfbridgeJoin{
		ConfbridgeID: confbridgeID,
	}
	opt, err := json.Marshal(option)
	if err != nil {
		log.Errorf("Could not marshal the option. err: %v", err)
		return nil, err
	}

	confbridgeJoin := fmaction.Action{
		Type:   fmaction.TypeConfbridgeJoin,
		Option: opt,
	}
	actions = append(actions, confbridgeJoin)

	// append the post actions
	actions = append(actions, postActions...)

	return actions, nil
}

// createConferenceFlow creates a conference flow and returns created flow.
func (h *conferenceHandler) createConferenceFlow(ctx context.Context, customerID uuid.UUID, conferenceID uuid.UUID, confbridgeID uuid.UUID, preActions []fmaction.Action, postActions []fmaction.Action) (*fmflow.Flow, error) {
	log := logrus.WithField("func", "createConferenceFlow")

	// create flow actions
	actions, err := h.createConferenceFlowActions(confbridgeID, preActions, postActions)
	if err != nil {
		log.Errorf("Could not create actions. err: %v", err)
		return nil, err
	}
	log.Debugf("Created flow actions. actions: %v", actions)

	// create flow name
	flowName := fmt.Sprintf("conference-%s", conferenceID.String())

	// create flow
	resFlow, err := h.reqHandler.FMV1FlowCreate(ctx, customerID, fmflow.TypeConference, flowName, "generated for conference by conference-manager.", actions, true)
	if err != nil {
		log.Errorf("Could not create a conference flow. err: %v", err)
		return nil, err
	}
	log.Debugf("Created a conference flow. res: %v", resFlow)

	return resFlow, nil
}

// Gets returns list of conferences.
func (h *conferenceHandler) Gets(ctx context.Context, customerID uuid.UUID, confType conference.Type, size uint64, token string) ([]*conference.Conference, error) {
	log := logrus.WithField("func", "Gets")

	var res []*conference.Conference
	var err error
	if confType == "" {
		res, err = h.db.ConferenceGets(ctx, customerID, size, token)
	} else {
		res, err = h.db.ConferenceGetsWithType(ctx, customerID, confType, size, token)
	}

	if err != nil {
		log.Errorf("Could not get conferences. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns conference.
func (h *conferenceHandler) Get(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithField("func", "Get")

	res, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conferences. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create is handy function for creating a conference.
// it increases corresponded counter
func (h *conferenceHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	timeout int,
	preActions []fmaction.Action,
	postActions []fmaction.Action,
) (*conference.Conference, error) {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":          "Update",
			"conference_id": id,
		},
	)
	log.Debugf("Updating the conference. name: %s, detail: %s, timeout: %d, pre_actions: %v, post_actions: %v",
		name, detail, timeout, preActions, postActions)

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
	f, err := h.reqHandler.FMV1FlowGet(ctx, cf.FlowID)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return nil, err
	}
	f.Actions = actions

	// update flow
	newFlow, err := h.reqHandler.FMV1FlowUpdate(ctx, f)
	if err != nil {
		log.Errorf("Could not update the flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", newFlow).Debugf("Updated the flow.")

	if timeout > 0 && timeout < 60 {
		timeout = defaultConferenceTimeout
	}

	// update conference
	if errSet := h.db.ConferenceSet(ctx, id, name, detail, timeout, preActions, postActions); errSet != nil {
		log.Errorf("Could not update the conference. err: %v", errSet)
		return nil, err
	}

	// get updated conference and notify
	newConf, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conference. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, newConf.CustomerID, conference.EventTypeConferenceUpdated, newConf)

	// set the timeout if it was set
	if cf.Timeout > 0 {
		if err := h.reqHandler.CFV1ConferenceDeleteDelay(ctx, id, cf.Timeout*1000); err != nil {
			log.Errorf("Could not start conference timeout. err: %v", err)
		}
	}

	return newConf, nil
}
