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

func (h *callHandler) ExecuteAction(c *call.Call, a *action.Action) error {
	ctx := context.Background()

	// set action to call
	if err := h.db.CallSetAction(ctx, c.ID, a); err != nil {
		return err
	}

	switch a.Type {
	case action.TypeEcho:
		return h.executeActionEcho(c, a)

	default:
		return fmt.Errorf("no action handle found. type: %s", a.Type)
	}
}

// ActionNext Execute next action
func (h *callHandler) ActionNext(c *call.Call) error {
	//
	log := log.WithFields(
		logrus.Fields{
			"call": c.ID,
		})

	// validate next action
	if c.Action.Next == uuid.Nil {
		log.Debug("End of call flow. No more next action left.")
		h.reqHandler.AstChannelHangup(c.AsteriskID, c.ChannelID, ari.ChannelCauseNoRouteDestination)
		return nil
	} else if c.Action.ID == c.Action.Next {
		log.WithFields(
			logrus.Fields{
				"action_current": c.Action.ID.String(),
				"action_next":    c.Action.Next.String(),
			}).Warning("Loop detected. Current and the next action id is the same.")
		return nil
	}

	return nil
}

func (h *callHandler) executeActionEcho(c *call.Call, a *action.Action) error {
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
