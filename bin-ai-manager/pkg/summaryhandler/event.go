package summaryhandler

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"
	cfconference "monorepo/bin-conference-manager/models/conference"
)

// EventCMCallHangup handles the call-manager's call_hangup event
func (h *summaryHandler) EventCMCallHangup(ctx context.Context, c *cmcall.Call) {
	sm, err := h.GetByReferenceID(ctx, c.ID)
	if err != nil {
		// summary not found. nothing todo
		return
	}

	h.ContentProcess(ctx, sm)
}

// EventCMConferenceUpdated handles the conference-manager's conference event
func (h *summaryHandler) EventCMConferenceUpdated(ctx context.Context, c *cfconference.Conference) {
	if c.Status != cfconference.StatusTerminated {
		// nothing to do
		return
	}

	sm, err := h.GetByReferenceID(ctx, c.ID)
	if err != nil {
		// summary not found. nothing todo
		return
	}

	h.ContentProcess(ctx, sm)
}
