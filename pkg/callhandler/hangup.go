package callhandler

import (
	"context"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
)

// Start starts the call service
func (h *callHandler) Hangup(cn *channel.Channel) error {
	ctx := context.Background()
	c, err := h.db.CallGetByChannelID(ctx, cn.ID)
	if err != nil {
		// nothing we can do
		return nil
	}

	// calculate hangup_reason, hangup_by
	reason := call.CalculateHangupReason(c.Status, cn.HangupCause)
	hangupBy := call.CalculateHangupBy(c.Status)

	// update call
	if err := h.db.CallSetHangup(ctx, c.ID, reason, hangupBy, cn.TMEnd); err != nil {
		// we don't channel hangup here, because the channel has already gone.
		return err
	}
	promCallHangupTotal.WithLabelValues(string(c.Direction), string(c.Type), string(reason)).Inc()

	return nil
}
