package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
)

// createConference is handy function for creating a conference.
// it increases corresponded counter
func (h *conferenceHandler) createConference(ctx context.Context, cf *conference.Conference) error {
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

	mapHandler := map[conference.Type]func(*conference.Conference) (*conference.Conference, error){
		conference.TypeConference: h.startTypeConference,
	}

	handler := mapHandler[reqConf.Type]
	if handler == nil {
		return nil, fmt.Errorf("could not find conference handler. type: %s", reqConf.Type)
	}

	return handler(reqConf)
}

// startTypeTransfer handles transfer conference
func (h *conferenceHandler) startTypeTransfer(cf *conference.Conference, c *call.Call) error {

	// todo: ????
	return nil
}

// startTypeConference inits the conference for conference type.
func (h *conferenceHandler) startTypeConference(req *conference.Conference) (*conference.Conference, error) {
	ctx := context.Background()
	conferenceID := uuid.Must(uuid.NewV4())

	log := log.WithFields(
		log.Fields{
			"conference": conferenceID.String(),
			"user_id":    req.UserID,
			"type":       conference.TypeConference,
		})
	log.Debug("Starting conference.")

	// create a conference with given requested conference info
	cf := &conference.Conference{
		ID:       conferenceID,
		Type:     conference.TypeConference,
		BridgeID: "",

		UserID:  req.UserID,
		Name:    req.Name,
		Detail:  req.Detail,
		Data:    req.Data,
		Timeout: req.Timeout,

		CallIDs: []uuid.UUID{},
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
		if err := h.reqHandler.CallConferenceTerminate(conferenceID, "timeout", cf.Timeout*1000); err != nil {
			log.Errorf("Could not start conference timeout. err: %v", err)
		}
	}

	return res, nil
}
