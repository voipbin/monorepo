package agenthandler

import (
	"context"
	"monorepo/bin-agent-manager/models/resource"
	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// webhookCallCreated handles the call-manager's call_created event.
// It creates a resource for each agent associated with the call's address.
//
// Parameters:
// ctx (context.Context): The context for the request.
// c (*cmcall.Call): The call object.
//
// Returns:
// error: An error if any occurred during the operation, otherwise nil.
func (h *agentHandler) webhookCallCreated(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "webhookCallCreated",
		"call": c,
	})
	log.Debugf("Creating resource for the call. call_id: %s", c.ID)

	// Determine the address based on the call's direction
	addr := c.Source
	if c.Direction == cmcall.DirectionOutgoing {
		addr = c.Destination
	}
	log.WithField("address", addr).Debugf("Found call address. address_type: %s, address_target: %s", addr.Type, addr.Target)

	// Get agents associated with the call's address
	ags, err := h.dbGetsByCustomerIDAndAddress(ctx, c.CustomerID, addr)
	if err != nil {
		log.Errorf("Could not get agents info. err:  %v", err)
		return errors.Wrapf(err, "could not get agents info. err: %v", err)
	}
	log.WithField("agents", ags).Debugf("Found agents informations. len: %d", len(ags))

	// Create a resource for each agent
	for _, a := range ags {
		log.Debugf("Creating resource for the agent. agent_id: %s", a.ID)
		r, err := h.resourceHandler.Create(ctx, c.CustomerID, a.ID, resource.ReferenceTypeCall, c.ID, c)
		if err != nil {
			log.Errorf("Could not create the resource. err: %v", err)
			continue
		}
		log.WithField("resource", r).Debugf("Created resource. resource_id: %s", r.ID)
	}

	return nil
}

// webhookCallUpdated handles the call-manager's call_(updated) event.
// It creates a resource for each agent associated with the call's address.
//
// Parameters:
// ctx (context.Context): The context for the request.
// c (*cmcall.Call): The call object.
//
// Returns:
// error: An error if any occurred during the operation, otherwise nil.
func (h *agentHandler) webhookCallUpdated(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "webhookCallUpdated",
		"call": c,
	})
	log.Debugf("Updating resource for the call. call_id: %s", c.ID)

	// get resources
	filters := map[string]string{
		"customer_id":    c.CustomerID.String(),
		"reference_type": string(resource.ReferenceTypeCall),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}

	// get related resources
	rs, err := h.resourceHandler.Gets(ctx, 100, "", filters)
	if err != nil {
		log.Errorf("Could not get resources. err: %v", err)
		return nil
	}

	// update resources
	for _, r := range rs {
		log.WithField("resource", r).Debugf("Updating resource info. resource_id: %s", r.ID)
		tmp, err := h.resourceHandler.UpdateData(ctx, r.ID, c)
		if err != nil {
			log.Errorf("Could not update the resource info. err: %v", err)
			continue
		}
		log.WithField("resource", tmp).Debugf("Updated resource info. resource_id: %s", tmp.ID)
	}

	return nil
}
