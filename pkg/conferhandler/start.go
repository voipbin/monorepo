package conferhandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

func (h *conferHandler) Start(cType conference.Type, c *call.Call) (*conference.Conference, error) {
	mapHandler := map[conference.Type]func(*call.Call) (*conference.Conference, error){
		// conference.TypeConference: h.startConferTypeConference,
		conference.TypeEcho: h.startTypeEcho,
		// conference.TypeTransfer:   h.startConferTypeTransfer,
	}

	handler := mapHandler[cType]
	if handler == nil {
		return nil, fmt.Errorf("could not find conference handler. type: %s", cType)
	}

	return handler(c)
}

// startTypeTransfer handles transfer conference
func (h *conferHandler) startTypeTransfer(cf *conference.Conference, c *call.Call) error {

	// todo: ????
	return nil
}

// startTypeConference
func (h *conferHandler) startTypeConference(cf *conference.Conference, c *call.Call) error {

	// todo: ????
	return nil
}

// startTypeEcho
// echo conference makes a bridge and create a snoop channel and put the bridge together.
func (h *conferHandler) startTypeEcho(c *call.Call) (*conference.Conference, error) {
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// defer cancel()

	// create a bridge
	bridgeID := uuid.Must(uuid.NewV4()).String()
	if err := h.reqHandler.AstBridgeCreate(c.AsteriskID, bridgeID, "echo", bridge.TypeMixing); err != nil {
		return nil, fmt.Errorf("could not create a bridge for echo conference. err: %v", err)
	}

	// create a conference
	cf := conference.NewConference(conference.TypeEcho, "echo", "action echo")
	cf.CallIDs = append(cf.CallIDs, c.ID)
	cf.BridgeIDs = append(cf.BridgeIDs, bridgeID)

	// create a snoop channel
	args := fmt.Sprintf("CONTEXT=%s,CONFERENCE_ID=%s,BRIDGE_ID=%s,CALL_ID=%s",
		ContextConferenceEcho,
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
