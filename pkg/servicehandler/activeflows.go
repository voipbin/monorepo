package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

// activeflowGet validates the activeflow's ownership and returns the activeflow info.
func (h *serviceHandler) activeflowGet(ctx context.Context, u *cscustomer.Customer, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "activeflowGet",
		"customer_id":   u.ID,
		"activeflow_id": activeflowID,
	})

	// send request
	res, err := h.reqHandler.FlowV1ActiveflowGet(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not get the activeflow info. err: %v", err)
		return nil, err
	}
	log.WithField("activeflow", res).Debug("Received result.")

	if res.TMDelete < defaultTimestamp {
		log.Debugf("Deleted activeflow.. activeflow_id: %s", res.ID)
		return nil, fmt.Errorf("not found")
	}

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	return res, nil
}

// ActiveflowGet sends a request to flow-manager
// to getting a activeflow.
// it returns activeflow if it succeed.
func (h *serviceHandler) ActiveflowGet(ctx context.Context, u *cscustomer.Customer, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ActiveflowGet",
		"customer_id":   u.ID,
		"username":      u.Username,
		"activeflow_id": activeflowID,
	})

	// get activeflow
	tmp, err := h.activeflowGet(ctx, u, activeflowID)
	if err != nil {
		// no call info found
		log.Infof("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// ActiveflowGets sends a request to flow-manager
// to getting a list of activeflows.
// it returns list of activeflows if it succeed.
func (h *serviceHandler) ActiveflowGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*fmactiveflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ActiveflowGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get calls
	tmps, err := h.reqHandler.FlowV1ActiveflowGets(ctx, u.ID, token, size)
	if err != nil {
		log.Infof("Could not get activeflows info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*fmactiveflow.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// ActiveflowStop sends a request to flow-manager
// to stopping the activeflow.
// it returns activeflow if it succeed.
func (h *serviceHandler) ActiveflowStop(ctx context.Context, u *cscustomer.Customer, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ActiveflowStop",
		"customer":      u,
		"activeflow_id": activeflowID,
	})

	// get activeflow
	_, err := h.activeflowGet(ctx, u, activeflowID)
	if err != nil {
		// no call info found
		log.Infof("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.FlowV1ActiveflowStop(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not stop the activeflow. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()

	return res, nil
}

// ActiveflowDelete sends a request to flow-manager
// to delete the activeflow.
// it returns activeflow if it succeed.
func (h *serviceHandler) ActiveflowDelete(ctx context.Context, u *cscustomer.Customer, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ActiveflowDelete",
		"customer":      u,
		"activeflow_id": activeflowID,
	})

	_, err := h.activeflowGet(ctx, u, activeflowID)
	if err != nil {
		// no activeflow info found
		log.Infof("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	// send request
	tmp, err := h.reqHandler.FlowV1ActiveflowDelete(ctx, activeflowID)
	if err != nil {
		// no call info found
		log.Infof("Could not get activeflow info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()
	return res, nil
}
