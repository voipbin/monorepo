package servicehandler

import (
	"context"
	"fmt"
	"net/http"

	cmcall "monorepo/bin-call-manager/models/call"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// callGet validates the call's ownership and returns the call info.
func (h *serviceHandler) callGet(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "callGet",
		"call_id": callID,
	})

	// send request
	res, err := h.reqHandler.CallV1CallGet(ctx, callID)
	if err != nil {
		log.Errorf("Could not get the call info. err: %v", err)
		return nil, err
	}
	log.WithField("call", res).Debug("Received result.")

	if res.TMDelete < defaultTimestamp {
		log.Debugf("Deleted call. call_id: %s", res.ID)
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// CallCreate sends a request to call-manager
// to creating a call.
// it returns created calls and groupcalls info if it succeed.
func (h *serviceHandler) CallCreate(ctx context.Context, a *amagent.Agent, flowID uuid.UUID, actions []fmaction.Action, source *commonaddress.Address, destinations []commonaddress.Address) ([]*cmcall.WebhookMessage, []*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"flow_id":     flowID,
		"actions":     actions,
		"source":      source,
		"destination": destinations,
	})
	log.Debug("Creating a new call.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, nil, fmt.Errorf("user has no permission")
	}

	targetFlowID := flowID
	if targetFlowID == uuid.Nil {
		log.Debugf("The flowID is null. Creating a new temp flow for call dialing.")
		f, err := h.FlowCreate(ctx, a, "tmp", "tmp outbound flow", actions, false)
		if err != nil {
			log.Errorf("Could not create a flow for outoing call. err: %v", err)
			return nil, nil, err
		}
		log.WithField("flow", f).Debugf("Create a new tmp flow for call dialing. flow_id: %s", f.ID)

		targetFlowID = f.ID
	}

	// verify the flow
	f, err := h.flowGet(ctx, targetFlowID)
	if err != nil {
		log.Errorf("Could not get flow info. err: %v", err)
		return nil, nil, err
	}
	if f.CustomerID != a.CustomerID {
		log.WithField("flow", f).Errorf("The flow has wrong customer id")
		return nil, nil, fmt.Errorf("the flow has wrong customer id")
	}

	tmpCalls, tmpGroupcalls, err := h.reqHandler.CallV1CallsCreate(ctx, a.CustomerID, targetFlowID, uuid.Nil, source, destinations, false, false)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, nil, err
	}

	resCalls := []*cmcall.WebhookMessage{}
	for _, tmp := range tmpCalls {
		t := tmp.ConvertWebhookMessage()
		resCalls = append(resCalls, t)
	}

	resGroupcalls := []*cmgroupcall.WebhookMessage{}
	for _, tmp := range tmpGroupcalls {
		t := tmp.ConvertWebhookMessage()
		resGroupcalls = append(resGroupcalls, t)
	}

	return resCalls, resGroupcalls, nil
}

// CallGet sends a request to call-manager
// to getting a call.
// it returns call if it succeed.
func (h *serviceHandler) CallGet(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"call_id":     callID,
	})

	// get call
	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// convert
	res := c.ConvertWebhookMessage()
	return res, nil
}

// CallGets sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) CallGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	res, err := h.callGetsByFilters(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get calls info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// callGetsByFilters sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) callGetsByFilters(ctx context.Context, size uint64, token string, filters map[string]string) ([]*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "callGetsByFilters",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get calls
	tmps, err := h.reqHandler.CallV1CallGets(ctx, token, size, filters)
	if err != nil {
		log.Infof("Could not get calls info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*cmcall.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// CallDelete sends a request to call-manager
// to delete the call.
// it returns call if it succeed.
func (h *serviceHandler) CallDelete(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res, err := h.callDelete(ctx, callID)
	if err != nil {
		log.Errorf("Could not delete call info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// callDelete sends a request to call-manager
// to delete the call.
// it returns call if it succeed.
func (h *serviceHandler) callDelete(ctx context.Context, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "callDelete",
		"call_id": callID,
	})

	// send request
	tmp, err := h.reqHandler.CallV1CallDelete(ctx, callID)
	if err != nil {
		// no call info found
		log.Errorf("Could not delete call info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// CallHangup sends a request to call-manager
// to hangup the call.
// it returns call if it succeed.
func (h *serviceHandler) CallHangup(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallHangup",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// send request
	tmp, err := h.reqHandler.CallV1CallHangup(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// CallTalk sends a request to call-manager
// to talk to the call.
// it returns call if it succeed.
func (h *serviceHandler) CallTalk(ctx context.Context, a *amagent.Agent, callID uuid.UUID, text string, gender string, language string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallTalk",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	// send request
	if errReq := h.reqHandler.CallV1CallTalk(ctx, callID, text, gender, language, 10000); errReq != nil {
		// no call info found
		log.Infof("Could not talk to the call. err: %v", errReq)
		return errReq
	}

	return nil
}

// CallHoldOn sends a request to call-manager
// to hold the call.
// it returns error if it failed.
func (h *serviceHandler) CallHoldOn(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallHoldOn",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	// send request
	if errReq := h.reqHandler.CallV1CallHoldOn(ctx, callID); errReq != nil {
		// no call info found
		log.Infof("Could not hold the call. err: %v", errReq)
		return errReq
	}

	return nil
}

// CallHoldOff sends a request to call-manager
// to unhold the call.
// it returns error if it failed.
func (h *serviceHandler) CallHoldOff(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallHoldOff",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	// send request
	if errReq := h.reqHandler.CallV1CallHoldOff(ctx, callID); errReq != nil {
		// no call info found
		log.Infof("Could not unhold the call. err: %v", errReq)
		return errReq
	}

	return nil
}

// CallMuteOn sends a request to call-manager
// to mute the call.
// it returns error if it failed.
func (h *serviceHandler) CallMuteOn(ctx context.Context, a *amagent.Agent, callID uuid.UUID, direction cmcall.MuteDirection) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallMuteOn",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	// send request
	if errReq := h.reqHandler.CallV1CallMuteOn(ctx, callID, direction); errReq != nil {
		// no call info found
		log.Infof("Could not mute the call. err: %v", errReq)
		return errReq
	}

	return nil
}

// CallMuteOff sends a request to call-manager
// to unmute the call.
// it returns error if it failed.
func (h *serviceHandler) CallMuteOff(ctx context.Context, a *amagent.Agent, callID uuid.UUID, direction cmcall.MuteDirection) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallMuteOff",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	// send request
	if errReq := h.reqHandler.CallV1CallMuteOff(ctx, callID, direction); errReq != nil {
		// no call info found
		log.Infof("Could not unmute the call. err: %v", errReq)
		return errReq
	}

	return nil
}

// CallMOHOn sends a request to call-manager
// to mute the call.
// it returns error if it failed.
func (h *serviceHandler) CallMOHOn(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallMOHOn",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	// send request
	if errReq := h.reqHandler.CallV1CallMusicOnHoldOn(ctx, callID); errReq != nil {
		// no call info found
		log.Infof("Could not mute the call. err: %v", errReq)
		return errReq
	}

	return nil
}

// CallMOHOff sends a request to call-manager
// to unmute the call.
// it returns error if it failed.
func (h *serviceHandler) CallMOHOff(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallMOHOff",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	// send request
	if errReq := h.reqHandler.CallV1CallMusicOnHoldOff(ctx, callID); errReq != nil {
		// no call info found
		log.Infof("Could not unmute the call. err: %v", errReq)
		return errReq
	}

	return nil
}

// CallSilenceOn sends a request to call-manager
// to mute the call.
// it returns error if it failed.
func (h *serviceHandler) CallSilenceOn(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallSilenceOn",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	// send request
	if errReq := h.reqHandler.CallV1CallSilenceOn(ctx, callID); errReq != nil {
		// no call info found
		log.Infof("Could not mute the call. err: %v", errReq)
		return errReq
	}

	return nil
}

// CallSilenceOff sends a request to call-manager
// to unmute the call.
// it returns error if it failed.
func (h *serviceHandler) CallSilenceOff(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallSilenceOff",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	// send request
	if errReq := h.reqHandler.CallV1CallSilenceOff(ctx, callID); errReq != nil {
		// no call info found
		log.Infof("Could not unmute the call. err: %v", errReq)
		return errReq
	}

	return nil
}

// CallMediaStreamStart starts a media streaming of the call
// it returns error if it failed.
func (h *serviceHandler) CallMediaStreamStart(ctx context.Context, a *amagent.Agent, callID uuid.UUID, encapsulation string, w http.ResponseWriter, r *http.Request) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "CallMediaStreamStart",
		"agent":         a,
		"call_id":       callID,
		"encapsulation": encapsulation,
	})

	c, err := h.callGet(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return fmt.Errorf("agent has no permission")
	}

	if errRun := h.websockHandler.RunMediaStream(ctx, w, r, cmexternalmedia.ReferenceTypeCall, callID, encapsulation); errRun != nil {
		log.Errorf("Could not run the meida stream handler correctly. err: %v", errRun)
		return errRun
	}

	return nil
}

// CallRecordingStart sends a request to call-manager
// to start the recording.
// it returns error if it failed.
func (h *serviceHandler) CallRecordingStart(
	ctx context.Context,
	a *amagent.Agent,
	callID uuid.UUID,
	format cmrecording.Format,
	endOfSilence int,
	endOfKey string,
	duration int,
	onEndFlowID uuid.UUID,
) (*cmcall.WebhookMessage, error) {

	c, err := h.callGet(ctx, callID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get call info")
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("agent has no permission")
	}

	// send request
	tmp, err := h.reqHandler.CallV1CallRecordingStart(
		ctx,
		callID,
		format,
		endOfSilence,
		endOfKey,
		duration,
		onEndFlowID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start the recording")
	}

	// convert
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CallRecordingStop sends a request to call-manager
// to stop the recording.
// it returns error if it failed.
func (h *serviceHandler) CallRecordingStop(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error) {

	c, err := h.callGet(ctx, callID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get call info")
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("agent has no permission")
	}

	// send request
	tmp, err := h.reqHandler.CallV1CallRecordingStop(ctx, callID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start the recording")
	}

	// convert
	res := tmp.ConvertWebhookMessage()
	return res, nil
}
