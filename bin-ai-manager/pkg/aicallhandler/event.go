package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	tmmessage "monorepo/bin-tts-manager/models/message"

	"github.com/sirupsen/logrus"
)

// EventCMConfbridgeJoined handles the call-manager's confbridge_joined event
func (h *aicallHandler) EventCMConfbridgeJoined(ctx context.Context, evt *cmconfbridge.EventConfbridgeJoined) {
	// get aicall
	cc, err := h.GetByReferenceID(ctx, evt.JoinedCallID)
	if err != nil {
		return
	}

	if cc.Status == aicall.StatusTerminated || cc.Status == aicall.StatusTerminating {
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

	_, err = h.ProcessTerminate(ctx, cc)
	if err != nil {
		return
	}
}

// EventTMPlayFinished handles the tts-manager's play finished event
func (h *aicallHandler) EventTMPlayFinished(ctx context.Context, evt *tmmessage.Message) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "EventTMPlayFinished",
		"streaming_id": evt.StreamingID,
	})

	// get aicall
	cc, err := h.GetByStreamingID(ctx, evt.StreamingID)
	if err != nil {
		return
	}

	if cc.Status != aicall.StatusTerminating {
		return
	}

	tmp, err := h.ProcessTerminate(ctx, cc)
	if err != nil {
		log.Errorf("Could not terminate the aicall. err: %v", err)
		return
	}
	log.WithField("aicall", tmp).Debugf("Terminated the aicall. aicall: %v", tmp)
}
