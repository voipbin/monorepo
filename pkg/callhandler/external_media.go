package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// ExternalMediaStart starts the external media processing
func (h *callHandler) ExternalMediaStart(id uuid.UUID, userID uint64, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string, data string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		},
	)
	log.Debug("Creating the external media.")

	ctx := context.Background()
	c, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get a call info. err: %v", err)
		return err
	}

	// create a bridge
	bridgeID := uuid.Must(uuid.NewV4())
	bridgeName := fmt.Sprintf("")
	if errBridge := h.reqHandler.AstBridgeCreate(c.AsteriskID, bridgeID.String(), bridgeName, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}); errBridge != nil {
		log.Errorf("Could not create a bridge for external media. error: %v", errBridge)
		return err
	}

	// create a snoop channel
	// set app args
	appArgs := fmt.Sprintf("context=%s,call_id=%s,bridge_id=%s",
		contextExternalSoop,
		c.ID,
		bridgeID,
	)
	snoopID := uuid.Must(uuid.NewV4())
	if errSnoop := h.reqHandler.AstChannelCreateSnoop(c.AsteriskID, c.ChannelID, snoopID.String(), appArgs, channel.SnoopDirectionBoth, channel.SnoopDirectionBoth); errSnoop != nil {
		log.Errorf("Could not create a snoop channel for the external media. error: %v", errSnoop)
		return errSnoop
	}

	// create a external media channel
	// set variables
	variables := map[string]string{
		"context":   contextExternalMedia,
		"bridge_id": bridgeID.String(),
		"call_id":   c.ID.String(),
	}
	extChannelID := uuid.Must(uuid.NewV4())
	if errExternal := h.reqHandler.AstChannelExternalMedia(c.AsteriskID, extChannelID.String(), externalHost, encapsulation, transport, connectionType, format, direction, data, variables); errExternal != nil {
		log.Errorf("Could not create a external media channel")
		return err
	}

	return nil
}
