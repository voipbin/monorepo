package callhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var (
	actionBegin uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
	actionEnd   uuid.UUID = uuid.Nil
)

func (h *callHandler) ActionExecute(c *call.Call, a *action.Action) error {
	ctx := context.Background()

	// set action to call
	if err := h.db.CallSetAction(ctx, c.ID, a); err != nil {
		return err
	}

	switch a.Type {
	case action.TypeEcho:
		return h.actionExecuteEcho(c, a)

	default:
		return fmt.Errorf("no action handle found. type: %s", a.Type)
	}
}

// ActionNext Execute next action
func (h *callHandler) ActionNext(c *call.Call) error {
	log := log.WithFields(
		logrus.Fields{
			"call": c.ID,
		})

	// validate current state
	switch {
	case c.Action.Next == actionEnd:
		// last action
		log.Debug("End of call flow. No more next action left.")
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return nil

	case c.Action.ID == c.Action.Next:
		// loop detected
		log.WithFields(
			logrus.Fields{
				"action_current": c.Action.ID.String(),
				"action_next":    c.Action.Next.String(),
			}).Warning("Loop detected. Current and the next action id is the same.")
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return nil

	case c.FlowID == uuid.Nil:
		// invalid flow id
		log.Info("The call's flow id is not valid. Hanging up the call.")
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNormalClearing)
		return nil
	}

	// get next action from flow-manager
	action, err := h.reqHandler.FlowActionGet(c.FlowID, c.Action.Next)
	if err != nil {
		log.Errorf("Could not get next flow action. err: %v", err)
		return err
	}

	return h.ActionExecute(c, action)
}

func (h *callHandler) actionExecuteEcho(c *call.Call, a *action.Action) error {
	var option action.OptionEcho
	if err := json.Unmarshal(a.Option, &option); err != nil {
		log.WithFields(log.Fields{
			"call":   c.ID,
			"action": a,
		}).Errorf("could not parse the option. err: %v", err)
		return fmt.Errorf("could not parse option. action: %v, err: %v", a, err)
	}

	// answer if the call is ringing
	if c.Direction == call.DirectionIncoming && c.Status == call.StatusRinging {
		// answer
		if err := h.reqHandler.AstChannelAnswer(c.AsteriskID, c.ChannelID); err != nil {
			return err
		}
	}

	// set timeout for 180 sec
	if err := h.reqHandler.AstChannelVariableSet(c.AsteriskID, c.ChannelID, "TIMEOUT(absolute)", "180"); err != nil {
		return err
	}

	// duration
	if option.Duration > 0 {
		// todo: add the delayed messaging.
	}

	// create a talk

	// put the channel into the talk

	// create a snoop channel
	// spy: in

	// put the snoop channel into the talk

	// create a bridge

	// put a channel into bridge

	// create a snoop channel

	// put a snoop channel into bridge

	return nil
}
