package callhandler

import (
	"context"

	"monorepo/bin-call-manager/models/call"
	smcontainer "monorepo/bin-sentinel-manager/models/container"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *callHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCUCustomerDeleted",
		"customer_id": cu.ID,
	})
	log.Debugf("Deleting all calls of the customer. customer_id: %s", cu.ID)

	// get all calls of the customer
	filters := map[call.Field]any{
		call.FieldCustomerID: cu.ID,
		call.FieldDeleted:    false,
	}
	calls, err := h.List(ctx, 1000, "", filters)
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

	// soft-delete the customer's OutboundConfig
	cfg, err := h.outboundConfigHandler.GetByCustomerID(ctx, cu.ID)
	if err != nil {
		log.Warnf("Could not get outbound config for deleted customer. customer_id: %s, err: %v", cu.ID, err)
	} else if cfg != nil {
		if _, err := h.outboundConfigHandler.Delete(ctx, cfg.ID); err != nil {
			log.Warnf("Could not delete outbound config for deleted customer. customer_id: %s, err: %v", cu.ID, err)
		} else {
			log.Debugf("Deleted outbound config for customer. customer_id: %s, config_id: %s", cu.ID, cfg.ID)
		}
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

// EventSMContainerDied handles the sentinel-manager's container_died event.
//
// It replaces EventSMPodDeleted (VOIP-1418). The Kubernetes namespace/label filter and the
// `asterisk-id` annotation lookup are gone: sentinel's Docker backend resolves the asterisk-id
// itself and reports the logical service directly, so the filter is a plain field comparison with
// no indirection left to replicate.
func (h *callHandler) EventSMContainerDied(ctx context.Context, c *smcontainer.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventSMContainerDied",
		"container": c,
	})

	if c.Service != smcontainer.ServiceAsteriskCall {
		// conference/registrar containers carry no call legs to recover.
		return nil
	}

	// NEW GUARD (design §3.3/§3.6): sentinel publishes an unresolved asterisk-id when a container
	// died before its id could ever be resolved. Passing "" through to RecoveryStart would run
	// GetChannelsForRecovery against an empty asterisk-id. The previous pod-based handler had no
	// such guard because a pod annotation was assumed always present.
	if c.AsteriskID == "" {
		log.Warnf("Received a container died event without a resolved asterisk id. Skipping the recovery. container_name: %s", c.ContainerName)
		return nil
	}

	log.Debugf("Received container died event for an asterisk-call container. Starting call recovery. container_name: %s, asterisk_id: %s", c.ContainerName, c.AsteriskID)
	if errRecovery := h.RecoveryStart(ctx, c.AsteriskID); errRecovery != nil {
		return errors.Wrapf(errRecovery, "failed to start recovery for container %s", c.ContainerName)
	}

	return nil
}

// EventCUCustomerFrozen handles the customer-manager's customer_frozen event
func (h *callHandler) EventCUCustomerFrozen(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCUCustomerFrozen",
		"customer_id": cu.ID,
	})
	log.Debugf("Hanging up all calls for frozen customer. customer_id: %s", cu.ID)

	// get all active calls of the customer
	filters := map[call.Field]any{
		call.FieldCustomerID: cu.ID,
		call.FieldDeleted:    false,
	}
	calls, err := h.List(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not get calls list. err: %v", err)
		return errors.Wrap(err, "could not get calls list")
	}

	// hangup all calls
	for _, e := range calls {
		log.Debugf("Hanging up call for frozen customer. call_id: %s", e.ID)
		if _, errHangup := h.HangingUp(ctx, e.ID, call.HangupReasonNormal); errHangup != nil {
			log.Errorf("Could not hangup call. call_id: %s, err: %v", e.ID, errHangup)
		}
	}

	return nil
}
