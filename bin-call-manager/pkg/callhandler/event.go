package callhandler

import (
	"context"

	"monorepo/bin-call-manager/models/call"
	smpod "monorepo/bin-sentinel-manager/models/pod"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *callHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all calls of the customer. customer_id: %s", cu.ID)

	// get all calls of the customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	calls, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets calls list. err: %v", err)
		return errors.Wrap(err, "could not get calls list")
	}

	// delete all calls
	for _, e := range calls {
		log.Debugf("Deleting call info. call_id: %s", e.ID)
		tmp, err := h.Delete(ctx, e.ID)
		if err != nil {
			log.Errorf("Could not delete call info. err: %v", err)
			continue
		}
		log.WithField("call", tmp).Debugf("Deleted call info. call_id: %s", tmp.ID)
	}

	return nil
}

// EventFMActiveflowUpdated handles the flow-manager's activeflow_updated event
func (h *callHandler) EventFMActiveflowUpdated(ctx context.Context, a *fmactiveflow.Activeflow) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "EventFMActiveflowUpdated",
		"activeflow": a,
	})

	if a.Status != fmactiveflow.StatusEnded || a.ReferenceType != fmactiveflow.ReferenceTypeCall {
		// nothing to do
		return nil
	}
	log.Debugf("Received activeflow status ended. activeflow_id: %s", a.ID)

	// safe to hanging up the hangup call
	c, err := h.HangingUp(ctx, a.ReferenceID, call.HangupReasonNormal)
	if err != nil {
		log.Errorf("Could not hangup the call. err: %v", err)
		return err
	}
	log.WithField("call", c).Debugf("Hangup call detail. call_id: %s", c.ID)

	return nil
}

// EventSMPodDeleted handles the sentinel-manager's pod_deleted event
func (h *callHandler) EventSMPodDeleted(ctx context.Context, p *smpod.Pod) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "EventSMPodDeleted",
		"pod":  p,
	})

	if p.Namespace == asteriskPodNamespace && p.Labels["app"] == asteriskPodLabelApp {
		log.Debugf("Received pod deleted event for asterisk-call pod. Starting call recovery. pod_name: %s, pod_namespace: %s", p.Name, p.Namespace)
		if errRecovery := h.RecoveryStart(ctx, p.Annotations["asterisk-id"]); errRecovery != nil {
			return errors.Wrapf(errRecovery, "failed to start recovery for pod %s/%s", p.Namespace, p.Name)
		}
	}

	return nil
}
