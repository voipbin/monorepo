package campaigncallhandler

import (
	"context"
	"fmt"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-campaign-manager/models/campaigncall"

	"github.com/sirupsen/logrus"
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

	// the campaigncall is already done. a second Done would repeat the webhook, the metric and the outdial target update.
	// the only legitimate second Done is a late success after a recorded failure (the create request errored although the
	// call was created and answered): it corrects the result and the outdial target.
	if cc.Status == campaigncall.StatusDone && (cc.Result != campaigncall.ResultFail || result != campaigncall.ResultSuccess) {
		log.Infof("The campaigncall is already done. Skipping. campaigncall_id: %s, stored_result: %s, new_result: %s", cc.ID, cc.Result, result)
		return cc, nil
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
