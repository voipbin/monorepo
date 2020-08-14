package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
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
		conference.TypeEcho:       h.startTypeEcho,
		// conference.TypeTransfer:   h.startConferTypeTransfer,
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

// startTypeEcho
// echo conference makes a bridge and create a snoop channel and put the bridge together.
func (h *conferenceHandler) startTypeEcho(req *conference.Conference, c *call.Call) (*conference.Conference, error) {
	ctx := context.Background()

	// create a conference
	id := uuid.Must(uuid.NewV4())

	log := log.WithFields(
		log.Fields{
			"call":       c.ID.String(),
			"conference": id.String(),
			"type":       conference.TypeEcho,
		})
	log.Debug("Starting conference.")

	// create a bridge and add to conference
	bridgeID := uuid.Must(uuid.NewV4()).String()
	bridgeName := generateBridgeName(conference.TypeEcho, id, false)
	if err := h.reqHandler.AstBridgeCreate(c.AsteriskID, bridgeID, bridgeName, []bridge.Type{bridge.TypeMixing}); err != nil {
		return nil, fmt.Errorf("could not create a bridge for echo conference. err: %v", err)
	}

	cf := conference.NewConference(id, conference.TypeEcho, bridgeID, req)
	log = log.WithFields(
		logrus.Fields{
			"bridge": bridgeID,
		})
	log.Debug("Created bridge.")

	// create a conference
	if err := h.createConference(ctx, cf); err != nil {
		return nil, fmt.Errorf("could not create a conference. err: %v", err)
	}

	// create a snoop channel
	args := fmt.Sprintf("CONTEXT=%s,CONFERENCE_ID=%s,BRIDGE_ID=%s,CALL_ID=%s",
		contextConferenceEcho,
		cf.ID.String(),
		bridgeID,
		c.ID.String(),
	)
	snoopID := uuid.Must(uuid.NewV4())
	if err := h.reqHandler.AstChannelCreateSnoop(
		c.AsteriskID,
		c.ChannelID,
		snoopID.String(),
		args,
		channel.SnoopDirectionIn,   // spy:in
		channel.SnoopDirectionNone, // whisper:nil
	); err != nil {
		return nil, fmt.Errorf("could not create a snopp channel for echo conference. err: %v", err)
	}

	// put the channel into the bridge
	if err := h.reqHandler.AstBridgeAddChannel(c.AsteriskID, bridgeID, c.ChannelID, "", false, false); err != nil {
		h.reqHandler.AstBridgeDelete(c.AsteriskID, bridgeID)
		return nil, fmt.Errorf("could not add the channel into the the bridge. bridge: %s", bridgeID)
	}

	// answer
	if err := h.reqHandler.AstChannelAnswer(c.AsteriskID, c.ChannelID); err != nil {
		h.reqHandler.AstBridgeDelete(c.AsteriskID, bridgeID)
		return nil, fmt.Errorf("could not answer the channel. err: %v", err)
	}

	return cf, nil
}
