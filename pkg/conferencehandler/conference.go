package conferencehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
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
	log := logrus.WithFields(logrus.Fields{
		"func":            "Create",
		"customer_id":     customerID,
		"conference_type": conferenceType,
	})

	id := uuid.Must(uuid.NewV4())
	log = log.WithField("conference_id", id.String())

	// send confbridge create request
	confbridgeType := cmconfbridge.TypeConnect
	if conferenceType == conference.TypeConference {
		confbridgeType = cmconfbridge.TypeConference
	}

	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, customerID, confbridgeType)
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
	tmp := &conference.Conference{
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

		ConferencecallIDs: []uuid.UUID{},
		RecordingIDs:      []uuid.UUID{},
	}

	// create a conference record
	if err := h.db.ConferenceCreate(ctx, tmp); err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}
	promConferenceCreateTotal.WithLabelValues(string(tmp.Type)).Inc()

	// get created conference and notify
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created conference. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conference.EventTypeConferenceCreated, res)
	log.WithField("conference", res).Debugf("Created a new conference. conference_id: %s", res.ID)

	// set the timeout if it was set
	if res.Timeout > 0 {
		if err := h.reqHandler.ConferenceV1ConferenceDeleteDelay(ctx, id, res.Timeout*1000); err != nil {
			log.Errorf("Could not start conference timeout. err: %v", err)
		}
	}

	return res, nil
}

// createConferenceFlowActions creates the actions for conference join.
func (h *conferenceHandler) createConferenceFlowActions(confbridgeID uuid.UUID, preActions []fmaction.Action, postActions []fmaction.Action) ([]fmaction.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "createConferenceFlowActions",
		"confbridge_id": confbridgeID,
		"pre_actions":   preActions,
		"post_actions":  postActions,
	})
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
	log := logrus.WithFields(logrus.Fields{
		"func":          "createConferenceFlow",
		"customer_id":   customerID,
		"conference_id": conferenceID,
		"confbridge_id": confbridgeID,
		"pre_actions":   preActions,
		"post_actions":  postActions,
	})

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
	resFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, customerID, fmflow.TypeConference, flowName, "generated for conference by conference-manager.", actions, true)
	if err != nil {
		log.Errorf("Could not create a conference flow. err: %v", err)
		return nil, err
	}
	log.Debugf("Created a conference flow. res: %v", resFlow)

	return resFlow, nil
}

// Gets returns list of conferences.
func (h *conferenceHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*conference.Conference, error) {
	res, err := h.db.ConferenceGets(ctx, size, token, filters)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Get returns conference.
func (h *conferenceHandler) Get(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	res, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get conference.")
	}

	return res, nil
}

// Delete deletes the conference.
func (h *conferenceHandler) Delete(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Delete",
		"conference_id": id,
	})

	_, errTerm := h.Terminating(ctx, id)
	if errTerm != nil {
		log.Errorf("Could not terminate the conference. err: %v", errTerm)
		return nil, errTerm
	}

	if errDel := h.db.ConferenceDelete(ctx, id); errDel != nil {
		log.Errorf("Could not delete conference. err: %v", errDel)
		return nil, errDel
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted conference. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conference.EventTypeConferenceDeleted, res)

	return res, nil
}

// GetByConfbridgeID returns conference of the given confbridge id.
func (h *conferenceHandler) GetByConfbridgeID(ctx context.Context, confbridgeID uuid.UUID) (*conference.Conference, error) {
	res, err := h.db.ConferenceGetByConfbridgeID(ctx, confbridgeID)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get conference.")
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
	log := logrus.WithFields(logrus.Fields{
		"func":          "Update",
		"conference_id": id,
	})
	log.Debugf("Updating the conference. name: %s, detail: %s, timeout: %d, pre_actions: %v, post_actions: %v",
		name, detail, timeout, preActions, postActions)

	// get conference
	cf, err := h.Get(ctx, id)
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
	f, err := h.reqHandler.FlowV1FlowGet(ctx, cf.FlowID)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return nil, err
	}
	f.Actions = actions

	// update flow
	newFlow, err := h.reqHandler.FlowV1FlowUpdate(ctx, f)
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
	res, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conference. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conference.EventTypeConferenceUpdated, res)

	// set the timeout if it was set
	if cf.Timeout > 0 {
		if err := h.reqHandler.ConferenceV1ConferenceDeleteDelay(ctx, id, cf.Timeout*1000); err != nil {
			log.Errorf("Could not start conference timeout. err: %v", err)
		}
	}

	return res, nil
}

// UpdateRecordingID updates the conference's recording id.
// if the recording id is not uuid.Nil, it also adds to the recording_ids
func (h *conferenceHandler) UpdateRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateRecordingID",
		"conference_id": id,
		"recording_id":  recordingID,
	})

	if errSet := h.db.ConferenceSetRecordingID(ctx, id, recordingID); errSet != nil {
		log.Errorf("Could not set the recording id. err: %v", errSet)
		return nil, errSet
	}

	if recordingID != uuid.Nil {
		// add the recording id
		log.Debugf("Adding the recording id. conference_id: %s, recording_id: %s", id, recordingID)
		if errAdd := h.db.ConferenceAddRecordingIDs(ctx, id, recordingID); errAdd != nil {
			log.Errorf("Could not add the recording id. err: %v", errAdd)
			return nil, errAdd
		}
	}

	// get updated conference
	res, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conference. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateTranscribeID updates the conference's transcribe id.
// if the transcribe id is not uuid.Nil, it also adds to the transcribe_ids
func (h *conferenceHandler) UpdateTranscribeID(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateTranscribeID",
		"conference_id": id,
		"transcribe_id": transcribeID,
	})

	if errSet := h.db.ConferenceSetTranscribeID(ctx, id, transcribeID); errSet != nil {
		log.Errorf("Could not set the transcribe id. err: %v", errSet)
		return nil, errSet
	}

	if transcribeID != uuid.Nil {
		// add the transcribe id
		log.Debugf("Adding the transcribe id. conference_id: %s, transcribe_id: %s", id, transcribeID)
		if errAdd := h.db.ConferenceAddTranscribeIDs(ctx, id, transcribeID); errAdd != nil {
			log.Errorf("Could not add the transcribe id. err: %v", errAdd)
			return nil, errAdd
		}
	}

	// get updated conference
	res, err := h.db.ConferenceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conference. err: %v", err)
		return nil, err
	}

	return res, nil
}

// AddConferencecallID adds the conference's conferencecall id.
func (h *conferenceHandler) AddConferencecallID(ctx context.Context, id uuid.UUID, conferencecallID uuid.UUID) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "AddConferencecallID",
		"conference_id":     id,
		"conferencecall_id": conferencecallID,
	})

	// add the call to the conference.
	if errAdd := h.db.ConferenceAddConferencecallID(ctx, id, conferencecallID); errAdd != nil {
		log.Errorf("Could not add the conferencecall to the conference. err: %v", errAdd)
		return nil, errAdd
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conference info. err: %v", err)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conference.EventTypeConferenceUpdated, res)

	return res, nil
}

// UpdateStatus updates the status and return the updated conference info
func (h *conferenceHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status conference.Status) (*conference.Conference, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateStatus",
		"conference_id": id,
		"status":        status,
	})

	// add the call to the conference.
	if errUpdate := h.db.ConferenceSetStatus(ctx, id, status); errUpdate != nil {
		log.Errorf("Could not update the conference status. err: %v", errUpdate)
		return nil, errors.Wrap(errUpdate, "Could not update the conference status.")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conference info. err: %v", err)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conference.EventTypeConferenceUpdated, res)

	return res, nil
}
