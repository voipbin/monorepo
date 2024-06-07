package resourcehandler

import (
	"context"
	"monorepo/bin-agent-manager/models/resource"
	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCallDeleted handles the call-manager's call_deleted event.
// It creates a resource for each agent associated with the call's address.
//
// Parameters:
// ctx (context.Context): The context for the request.
// c (*cmcall.Call): The call object.
//
// Returns:
// error: An error if any occurred during the operation, otherwise nil.
func (h *resourceHandler) EventCallDeleted(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "EventCallDeleted",
		"call": c,
	})
	log.Debugf("Deleting resource for the call. call_id: %s", c.ID)

	// gets the resources
	filters := map[string]string{
		"reference_type": string(resource.ReferenceTypeCall),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}
	rs, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not get resources. err: %v", err)
		return errors.Wrapf(err, "could not get resources. err: %v", err)
	}

	for _, r := range rs {
		// delete each resource
		tmp, err := h.Delete(ctx, r.ID)
		if err != nil {
			log.Errorf("Could not delete the resource. err: %v", err)
			continue
		}
		log.WithField("resource", tmp).Debugf("Deleted resource. resource_id: %s", tmp.ID)
	}

	return nil
}

// EventCallUpdated handles the call-manager's call_deleted event.
// It creates a resource for each agent associated with the call's address.
//
// Parameters:
// ctx (context.Context): The context for the request.
// c (*cmcall.Call): The call object.
//
// Returns:
// error: An error if any occurred during the operation, otherwise nil.
func (h *resourceHandler) EventCallUpdated(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "EventCallUpdated",
		"call": c,
	})
	log.Debugf("Updating resource for the call. call_id: %s", c.ID)

	// gets the resources
	filters := map[string]string{
		"reference_type": string(resource.ReferenceTypeCall),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}
	rs, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not get resources. err: %v", err)
		return errors.Wrapf(err, "could not get resources. err: %v", err)
	}

	for _, r := range rs {
		// delete each resource
		tmp, err := h.UpdateData(ctx, r.ID, c)
		if err != nil {
			log.Errorf("Could not delete the resource. err: %v", err)
			continue
		}
		log.WithField("resource", tmp).Debugf("Deleted resource. resource_id: %s", tmp.ID)
	}

	return nil
}
