package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
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

func (h *conferenceHandler) Start(reqConf *conference.Conference, c *call.Call) (*conference.Conference, error) {

	log := log.WithFields(
		log.Fields{
			"conference_type": reqConf.Type,
		})
	log.Info("Start conference.")

	mapHandler := map[conference.Type]func(*conference.Conference, *call.Call) (*conference.Conference, error){
		conference.TypeConference: h.startTypeConference,
	}

	handler := mapHandler[reqConf.Type]
	if handler == nil {
		return nil, fmt.Errorf("could not find conference handler. type: %s", reqConf.Type)
	}

	return handler(reqConf, c)
}

// startTypeTransfer handles transfer conference
func (h *conferenceHandler) startTypeTransfer(cf *conference.Conference, c *call.Call) error {

	// todo: ????
	return nil
}

// startTypeConference inits the conference for conference type.
func (h *conferenceHandler) startTypeConference(req *conference.Conference, c *call.Call) (*conference.Conference, error) {
	ctx := context.Background()
	conferenceID := uuid.Must(uuid.NewV4())

	log := log.WithFields(
		log.Fields{
			"conference": conferenceID.String(),
			"type":       conference.TypeConference,
		})
	log.Debug("Starting conference.")

	// create a bridge for conference
	bridgeID := uuid.Must(uuid.NewV4()).String()
	bridgeName := generateBridgeName(conference.TypeConference, conferenceID, false)
	if err := h.reqHandler.AstBridgeCreate(requesthandler.AsteriskIDConference, bridgeID, bridgeName, []bridge.Type{bridge.TypeMixing, bridge.TypeVideoSFU}); err != nil {
		log.Errorf("Could not create a bridge for a conference. err: %v", err)
		return nil, err
	}

	// create a conference with given requested conference info
	cf := conference.NewConference(conferenceID, conference.TypeConference, "", req)

	log = log.WithFields(
		logrus.Fields{
			"bridge": bridgeID,
		})
	log.Debug("Created bridge.")

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
