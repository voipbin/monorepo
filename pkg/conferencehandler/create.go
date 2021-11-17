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
	cb, err := h.reqHandler.CMV1ConfbridgeCreate(ctx, id)
	if err != nil {
		log.Errorf("Could not crate confbridge. err: %v", err)
		return nil, err
	}
	log.Debugf("Created confbridge. confbridge_id: %s", cb.ID)

	// create flow
	f, err := h.createConferenceFlow(userID, id, cb.ID, preActions, postActions)
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
		UserID:       userID,
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
		if err := h.reqHandler.CFV1ConferenceDeleteDelay(ctx, id, cf.Timeout*1000); err != nil {
			log.Errorf("Could not start conference timeout. err: %v", err)
		}
	}

	return cf, nil
}

// createConferenceFlowActions creates the actions for conference join.
func (h *conferenceHandler) createConferenceFlowActions(confbridgeID uuid.UUID, preActions []action.Action, postActions []action.Action) ([]action.Action, error) {
	log := logrus.New().WithField("func", "createConferenceFlow")
	actions := []action.Action{}

	// append the pre actions
	actions = append(actions, preActions...)

	// append the confbridge join
	option := action.OptionConfbridgeJoin{
		ConfbridgeID: confbridgeID.String(),
	}
	opt, err := json.Marshal(option)
	if err != nil {
		log.Errorf("Could not marshal the option. err: %v", err)
		return nil, err
	}

	confbridgeJoin := action.Action{
		Type:   action.TypeConfbridgeJoin,
		Option: opt,
	}
	actions = append(actions, confbridgeJoin)

	// append the post actions
	actions = append(actions, postActions...)

	return actions, nil
}

// createConferenceFlow creates a conference flow and returns created flow.
func (h *conferenceHandler) createConferenceFlow(userID uint64, conferenceID uuid.UUID, confbridgeID uuid.UUID, preActions []action.Action, postActions []action.Action) (*flow.Flow, error) {
	ctx := context.Background()
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
	resFlow, err := h.reqHandler.FMV1FlowCreate(ctx, userID, flowName, "generated for conference by conference-manager.", "", actions, true)
	if err != nil {
		log.Errorf("Could not create a conference flow. err: %v", err)
		return nil, err
	}
	log.Debugf("Created a conference flow. res: %v", resFlow)

	return resFlow, nil
}

// Gets returns list of conferences.
func (h *conferenceHandler) Gets(ctx context.Context, userID uint64, confType conference.Type, size uint64, token string) ([]*conference.Conference, error) {
	log := logrus.WithField("func", "Gets")

	var res []*conference.Conference
	var err error
	if confType == "" {
		res, err = h.db.ConferenceGets(ctx, userID, size, token)
	} else {
		res, err = h.db.ConferenceGetsWithType(ctx, userID, confType, size, token)
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
