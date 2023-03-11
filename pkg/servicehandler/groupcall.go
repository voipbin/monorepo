package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// groupcallGet validates the call's ownership and returns the call info.
func (h *serviceHandler) groupcallGet(ctx context.Context, u *cscustomer.Customer, groupcallID uuid.UUID) (*cmgroupcall.Groupcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "groupcallGet",
			"customer_id":  u.ID,
			"groupcall_id": groupcallID,
		},
	)

	// send request
	res, err := h.reqHandler.CallV1GroupcallGet(ctx, groupcallID)
	if err != nil {
		log.Errorf("Could not get the groupcall info. err: %v", err)
		return nil, err
	}
	log.WithField("groupcall", res).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	return res, nil
}

// CallGets sends a request to call-manager
// to getting a list of calls.
// it returns list of calls if it succeed.
func (h *serviceHandler) GroupcallGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GroupcallGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.GetCurTime()
	}

	// get calls
	tmps, err := h.reqHandler.CallV1GroupcallGets(ctx, u.ID, token, size)
	if err != nil {
		log.Infof("Could not get calls info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*cmgroupcall.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// GroupcallGet sends a request to call-manager
// to getting a groupcall.
// it returns groupcall if it succeed.
func (h *serviceHandler) GroupcallGet(ctx context.Context, u *cscustomer.Customer, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "GroupcallGet",
		"customer_id":  u.ID,
		"username":     u.Username,
		"groupcall_id": groupcallID,
	})

	// get call
	c, err := h.groupcallGet(ctx, u, groupcallID)
	if err != nil {
		// no call info found
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	// convert
	res := c.ConvertWebhookMessage()

	return res, nil
}

// GroupcallCreate sends a request to call-manager
// to creating a groupcall.
// it returns created groupcall info if it succeed.
func (h *serviceHandler) GroupcallCreate(ctx context.Context, u *cscustomer.Customer, source commonaddress.Address, destinations []commonaddress.Address, flowID uuid.UUID, actions []fmaction.Action, ringMethod cmgroupcall.RingMethod, answerMethod cmgroupcall.AnswerMethod) (*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "GroupcallCreate",
		"customer_id":   u.ID,
		"username":      u.Username,
		"flow_id":       flowID,
		"actions":       actions,
		"source":        source,
		"destinations":  destinations,
		"ring_method":   ringMethod,
		"answer_method": answerMethod,
	})

	// send request
	log.Debug("Creating a new groupcall.")

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

	tmp, err := h.reqHandler.CallV1GroupcallCreate(ctx, u.ID, source, destinations, targetFlowID, uuid.Nil, ringMethod, answerMethod, false)
	if err != nil {
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, err
}

// GroupcallHangup sends a request to groupcall-manager
// to hangup the groupcall.
// it returns groupcall if it succeed.
func (h *serviceHandler) GroupcallHangup(ctx context.Context, u *cscustomer.Customer, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "GroupcallHangup",
		"customer_id":  u.ID,
		"username":     u.Username,
		"groupcall_id": groupcallID,
	})

	_, err := h.groupcallGet(ctx, u, groupcallID)
	if err != nil {
		// no call info found
		log.Infof("Could not get groupcall info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.CallV1GroupcallHangup(ctx, groupcallID)
	if err != nil {
		// no call info found
		log.Infof("Could not get groupcall info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// GroupcallDelete sends a request to groupcall-manager
// to delete the groupcall.
// it returns groupcall if it succeed.
func (h *serviceHandler) GroupcallDelete(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID) (*cmgroupcall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GroupcallDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"call_id":     callID,
	})

	_, err := h.groupcallGet(ctx, u, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get groupcall info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.CallV1GroupcallDelete(ctx, callID)
	if err != nil {
		// no call info found
		log.Infof("Could not get groupcall info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}
