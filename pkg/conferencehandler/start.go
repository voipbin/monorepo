package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

// createConference is handy function for creating a conference.
// it increases corresponded counter
func (h *conferenceHandler) createConference(ctx context.Context, cf *conference.Conference) error {

	// set timestamp
	cf.TMCreate = getCurTime()
	cf.TMUpdate = defaultTimeStamp
	cf.TMDelete = defaultTimeStamp

	// create a conference record
	if err := h.db.ConferenceCreate(ctx, cf); err != nil {
		return fmt.Errorf("could not create a conference. err: %v", err)
	}
	promConferenceCreateTotal.WithLabelValues(string(cf.Type)).Inc()

	return nil
}

func (h *conferenceHandler) Start(reqConf *conference.Conference) (*conference.Conference, error) {
	log := log.WithFields(
		log.Fields{
			"conference_type": reqConf.Type,
		})
	log.Info("Start conference.")

	// check valid conference type
	if ret := conference.IsValidConferenceType(reqConf.Type); ret != true {
		return nil, fmt.Errorf("wrong conference type. type: %s", reqConf.Type)
	}

	return h.startConference(reqConf)
}

// startConference inits the conference.
func (h *conferenceHandler) startConference(req *conference.Conference) (*conference.Conference, error) {
	ctx := context.Background()
	conferenceID := uuid.Must(uuid.NewV4())

	log := log.WithFields(
		log.Fields{
			"conference": conferenceID.String(),
			"user_id":    req.UserID,
			"type":       req.Type,
		})
	log.Debug("Starting conference.")

	// create a conference with given requested conference info
	cf := &conference.Conference{
		ID:       conferenceID,
		Type:     req.Type,
		BridgeID: "",

		Status: conference.StatusProgressing,

		UserID:  req.UserID,
		Name:    req.Name,
		Detail:  req.Detail,
		Data:    req.Data,
		Timeout: req.Timeout,

		WebhookURI: req.WebhookURI,

		CallIDs:      []uuid.UUID{},
		RecordingIDs: []uuid.UUID{},
	}
	log.Debugf("Creating a conference. conference: %v", cf)

	// create a conference to database
	if err := h.createConference(ctx, cf); err != nil {
		log.Errorf("Could not create a conference. err: %v", err)
		return nil, err
	}

	res, err := h.db.ConferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get a created conference. err: %v", err)
		return nil, err
	}

	// set the timeout if it was set
	if cf.Timeout > 0 {
		if err := h.reqHandler.CallConferenceTerminate(conferenceID, cf.Timeout*1000); err != nil {
			log.Errorf("Could not start conference timeout. err: %v", err)
		}
	}

	// send webhook event
	h.notifyHandler.NotifyEvent(notifyhandler.EventTypeConferenceCreated, cf.WebhookURI, cf)

	return res, nil
}
