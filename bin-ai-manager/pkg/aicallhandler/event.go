package aicallhandler

import (
	"context"
	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
)

// EventCMConfbridgeJoined handles the call-manager's confbridge_joined event
func (h *aicallHandler) EventCMConfbridgeJoined(ctx context.Context, evt *cmconfbridge.EventConfbridgeJoined) {
	// get aicall
	cc, err := h.GetByReferenceID(ctx, evt.JoinedCallID)
	if err != nil {
		return
	}

	_, err = h.ProcessStart(ctx, cc)
	if err != nil {
		return
	}
}

// EventCMCallHangup handles the call-manager's confbridge_leaved event
func (h *aicallHandler) EventCMConfbridgeLeaved(ctx context.Context, evt *cmconfbridge.EventConfbridgeLeaved) {
	// get aicall
	cc, err := h.GetByReferenceID(ctx, evt.LeavedCallID)
	if err != nil {
		return
	}

	_, err = h.ProcessPause(ctx, cc)
	if err != nil {
		return
	}
}

// EventCMCallHangup handles the call-manager's call_hangup event
func (h *aicallHandler) EventCMCallHangup(ctx context.Context, evt *cmcall.Call) {
	// get aicall
	cc, err := h.GetByReferenceID(ctx, evt.ID)
	if err != nil {
		return
	}

	_, err = h.ProcessEnd(ctx, cc)
	if err != nil {
		return
	}
}
