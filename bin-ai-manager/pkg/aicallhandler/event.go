package aicallhandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmdtmf "monorepo/bin-call-manager/models/dtmf"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

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

	if cc.ConfbridgeID != evt.ID {
		return
	}

	_, err = h.ProcessTerminate(ctx, cc)
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

func (h *aicallHandler) EventCMDTMFReceived(ctx context.Context, evt *cmdtmf.DTMF) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "EventCMDTMFReceived",
		"dtmf_id": evt.ID,
	})

	// get aicall
	cc, err := h.GetByReferenceID(ctx, evt.CallID)
	if err != nil {
		log.Errorf("Could not get aicall by reference id. err: %v", err)
		return
	}

	messageText := fmt.Sprintf("type: %s\ndigit: %s\nduration: %d\n", defaultDTMFEvent, evt.Digit, evt.Duration)
	log.Debugf("Prepared the dtmf message to send to the aicall. message: %s", messageText)

	tmp, err := h.SendReferenceTypeCall(ctx, cc, message.RoleUser, messageText, true, true)
	if err != nil {
		log.Errorf("Could not send dtmf message to aicall. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Sent the dtmf message to the aicall.")
}

func (h *aicallHandler) EventPMPipecatcallInitialized(ctx context.Context, evt *pmpipecatcall.Pipecatcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "EventPMPipecatcallInitialized",
		"pipecatcall_id": evt.ID,
	})

	if evt.ReferenceType != pmpipecatcall.ReferenceTypeAICall {
		// nothing to do
		return
	}

	// get aicall
	cc, err := h.Get(ctx, evt.ReferenceID)
	if err != nil {
		log.Errorf("Could not get aicall info. err: %v", err)
		return
	}

	log.Debugf("The aicall's pipecatcall has initiated. aicall_id: %s, pipecatcall_id: %s", cc.ID, evt.ID)

	if cc.ReferenceType != aicall.ReferenceTypeCall {
		return
	}

	log.Debugf("Stopping currently playing media. aicall_id: %s, call_id: %s", cc.ID, cc.ReferenceID)
	if errStop := h.reqHandler.CallV1CallMediaStop(ctx, cc.ReferenceID); errStop != nil {
		log.Errorf("Could not stop the media on the call during pipecatcall initialization. err: %v", errStop)
		return
	}
}
