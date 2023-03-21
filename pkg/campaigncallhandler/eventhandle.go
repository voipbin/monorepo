package campaigncallhandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
)

// EventHandleActiveflowDeleted handles activeflow's deleted event.
func (h *campaigncallHandler) EventHandleActiveflowDeleted(ctx context.Context, cc *campaigncall.Campaigncall) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "EventHandleActiveflowDeleted",
		"campaigncall_id": cc.ID,
	})

	// update campaigncall to done.
	res, err := h.Done(ctx, cc.ID, campaigncall.ResultSuccess)
	if err != nil {
		log.Errorf("Could not done the campaigncall. err: %v", err)
		return nil, err
	}

	return res, nil
}

// EventhandleReferenceCallHungup handles reference call's hangup.
func (h *campaigncallHandler) EventHandleReferenceCallHungup(ctx context.Context, c *cmcall.Call, cc *campaigncall.Campaigncall) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "EventhandleReferenceCallHungup",
		"campaigncall_id": cc.ID,
	})

	// get result
	result, err := calcCampaigncallResultByCallHangupReason(c.HangupReason)
	if err != nil {
		log.Errorf("Could not calculate call result. err: %v", err)
		return nil, err
	}

	// update campaigncall to done.
	res, err := h.Done(ctx, cc.ID, result)
	if err != nil {
		log.Errorf("Could not done the campaigncall. err: %v", err)
		return nil, err
	}

	return res, nil
}

func calcCampaigncallResultByCallHangupReason(reason cmcall.HangupReason) (campaigncall.Result, error) {

	mapResult := map[cmcall.HangupReason]campaigncall.Result{
		cmcall.HangupReasonNormal:   campaigncall.ResultSuccess,
		cmcall.HangupReasonFailed:   campaigncall.ResultFail,
		cmcall.HangupReasonBusy:     campaigncall.ResultFail,
		cmcall.HangupReasonCanceled: campaigncall.ResultFail,
		cmcall.HangupReasonTimeout:  campaigncall.ResultFail,
		cmcall.HangupReasonNoanswer: campaigncall.ResultFail,
		cmcall.HangupReasonDialout:  campaigncall.ResultFail,
		cmcall.HangupReasonAMD:      campaigncall.ResultFail,
	}

	res, ok := mapResult[reason]
	if !ok {
		return campaigncall.ResultNone, fmt.Errorf("result code not found. reason: %s", reason)
	}

	return res, nil
}
