package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// callGet validates the call's ownership and returns the call info.
func (h *serviceHandler) callGet(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID) (*cmcall.Call, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "callGet",
			"customer_id":   u.ID,
			"transcribe_id": callID,
		},
	)

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

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	return res, nil
}

// CallCreate sends a request to call-manager
// to creating a call.
// it returns created call info if it succeed.
func (h *serviceHandler) CallCreate(ctx context.Context, u *cscustomer.Customer, flowID uuid.UUID, actions []fmaction.Action, source *commonaddress.Address, destinations []commonaddress.Address) ([]*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallCreate",
		"customer_id": u.ID,
		"username":    u.Username,
		"flow_id":     flowID,
		"actions":     actions,
		"source":      source,
		"destination": destinations,
	})

	// send request
	log.Debug("Creating a new call.")

	targetFlowID := flowID
	if targetFlowID == uuid.Nil {
		log.Debugf("The flowID is null. Creating a new temp flow for call dialing.")
		f, err := h.FlowCreate(ctx, u, "tmp", "tmp outbound flow", actions, false)
		if err != nil {
			log.Errorf("Could not create a flow for outoing call. err: %v", err)
			return nil, err
		}
		log.WithField("flow", f).Debugf("Create a new tmp flow for call dialing. flow_id: %s", f.ID)

		targetFlowID = f.ID
	}

	tmps, err := h.reqHandler.CallV1CallsCreate(ctx, u.ID, targetFlowID, uuid.Nil, source, destinations, false, false)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}

	res := []*cmcall.WebhookMessage{}
	for _, tmp := range tmps {
		t := tmp.ConvertWebhookMessage()
		res = append(res, t)
	}

	return res, err
}

// CallGet sends a request to call-manager
// to getting a call.
// it returns call if it succeed.
func (h *serviceHandler) CallGet(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"call_id":     callID,
	})

	// get call
	c, err := h.callGet(ctx, u, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	// convert
	res := c.ConvertWebhookMessage()

	return res, nil
}

// CallGets sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) CallGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = getCurTime()
	}

	// get calls
	tmps, err := h.reqHandler.CallV1CallGets(ctx, u.ID, token, size)
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
func (h *serviceHandler) CallDelete(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"call_id":     callID,
	})

	_, err := h.callGet(ctx, u, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.CallV1CallDelete(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// CallHangup sends a request to call-manager
// to hangup the call.
// it returns call if it succeed.
func (h *serviceHandler) CallHangup(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID) (*cmcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallHangup",
		"customer_id": u.ID,
		"username":    u.Username,
		"call_id":     callID,
	})

	_, err := h.callGet(ctx, u, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
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
func (h *serviceHandler) CallTalk(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID, text string, gender string, language string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CallTalk",
		"customer_id": u.ID,
		"username":    u.Username,
		"call_id":     callID,
	})

	_, err := h.callGet(ctx, u, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return err
	}

	// send request
	if errTalk := h.reqHandler.CallV1CallTalk(ctx, callID, text, gender, language, 10000); errTalk != nil {
		// no call info found
		log.Infof("Could not talk to the call. err: %v", errTalk)
		return errTalk
	}

	return nil
}
