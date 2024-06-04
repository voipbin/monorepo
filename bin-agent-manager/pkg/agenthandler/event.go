package agenthandler

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/models/resource"
)

// EventGroupcallCreated handles the call-manager's groupcall_created event
func (h *agentHandler) EventGroupcallCreated(ctx context.Context, c *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventGroupcallCreated",
		"groupcall": c,
	})

	if errAgent := h.eventGroupcallCreatedHandleAgent(ctx, c); errAgent != nil {
		log.Errorf("Could not handle the event from the agent handler.")
	}

	if errResource := h.eventGroupcallCreatedHandleResource(ctx, c); errResource != nil {
		log.Errorf("Could not handle the event from the resource handler.")
	}

	return nil
}

func (h *agentHandler) eventGroupcallCreatedHandleAgent(ctx context.Context, c *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventGroupcallCreatedHandleAgent",
		"groupcall": c,
	})
	log.Debugf("Creating resource for the groupcall. groupcall_id: %s", c.ID)

	for _, destination := range c.Destinations {
		if destination.Type != commonaddress.TypeAgent {
			// nothing to do
			continue
		}

		id := uuid.FromStringOrNil(destination.Target)
		if id == uuid.Nil {
			log.Errorf("Could not parse the agent id. target: %s", destination.Target)
			continue
		}

		// get agent info
		tmp, err := h.Get(ctx, id)
		if err != nil {
			log.Errorf("Could not get agent. err: %v", err)
			continue
		}

		if tmp.Status != agent.StatusAvailable {
			// nothing to do.
			continue
		}

		// update agent status
		tmp, err = h.UpdateStatus(ctx, tmp.ID, agent.StatusRinging)
		if err != nil {
			log.Errorf("Could not update agent status. err: %v", err)
			continue
		}
		log.WithField("agent", tmp).Debugf("Updated agent status to the ringing. agent_id: %s", tmp.ID)
	}

	return nil
}

func (h *agentHandler) eventGroupcallCreatedHandleResource(ctx context.Context, c *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventGroupcallCreatedHandleResource",
		"groupcall": c,
	})
	log.Debugf("Creating resource for the groupcall. groupcall_id: %s", c.ID)

	// Determine the address based on the call's direction
	for _, addr := range c.Destinations {
		if addr.Type != commonaddress.TypeExtension {
			continue
		}

		// Get agents associated with the call's address
		ags, err := h.dbGetsByCustomerIDAndAddress(ctx, c.CustomerID, addr)
		if err != nil {
			log.Errorf("Could not get agents info. err:  %v", err)
			return errors.Wrapf(err, "could not get agents info. err: %v", err)
		}

		// Create a resource for each agent
		for _, a := range ags {
			log.Debugf("Creating resource for the agent. agent_id: %s", a.ID)
			r, err := h.resourceHandler.Create(ctx, c.CustomerID, a.ID, resource.TypeCall, c)
			if err != nil {
				log.Errorf("Could not create the resource. err: %v", err)
				continue
			}
			log.WithField("resource", r).Debugf("Created resource. resource_id: %s", r.ID)
		}
	}

	return nil
}

// EventGroupcallProgressing handles the call-manager's groupcall_progressing event
func (h *agentHandler) EventGroupcallProgressing(ctx context.Context, groupcall *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventGroupcallProgressing",
		"groupcall": groupcall,
	})

	for _, destination := range groupcall.Destinations {
		if destination.Type != commonaddress.TypeAgent {
			// nothing to do
			continue
		}

		// parse agent id
		id := uuid.FromStringOrNil(destination.Target)
		if id == uuid.Nil {
			log.Errorf("Could not parse the agent id. target: %s", destination.Target)
			continue
		}

		// get agent info
		tmp, err := h.Get(ctx, id)
		if err != nil {
			log.Errorf("Could not get agent. err: %v", err)
			continue
		}

		// update agent status to busy
		tmp, err = h.UpdateStatus(ctx, tmp.ID, agent.StatusBusy)
		if err != nil {
			log.Errorf("Could not update agent status. err: %v", err)
			continue
		}
		log.WithField("agent", tmp).Debugf("Updated agent status to the busy. agent_id: %s", tmp.ID)
	}

	return nil
}

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *agentHandler) EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all agents in customer. customer_id: %s", cu.ID)

	// get all agents in customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	ags, err := h.Gets(ctx, 1000, h.utilHandler.TimeGetCurTime(), filters)
	if err != nil {
		log.Errorf("Could not gets agents list. err: %v", err)
		return errors.Wrap(err, "could not get agents list")
	}

	// delete all agents
	for _, a := range ags {
		log.Debugf("Deleting agent info. agent_id: %s", a.ID)
		tmp, err := h.deleteForce(ctx, a.ID)
		if err != nil {
			log.Errorf("Could not delete agent info. err: %v", err)
			continue
		}
		log.WithField("agent", tmp).Debugf("Deleted agent info. agent_id: %s", tmp.ID)
	}

	return nil
}

// EventCallCreated handles the call-manager's call_created event.
// It creates a resource for each agent associated with the call's address.
//
// Parameters:
// ctx (context.Context): The context for the request.
// c (*cmcall.Call): The call object.
//
// Returns:
// error: An error if any occurred during the operation, otherwise nil.
func (h *agentHandler) EventCallCreated(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "EventCallCreated",
		"call": c,
	})
	log.Debugf("Creating resource for the call. call_id: %s", c.ID)

	// Determine the address based on the call's direction
	addr := c.Source
	if c.Direction == cmcall.DirectionOutgoing {
		addr = c.Destination
	}

	// Get agents associated with the call's address
	ags, err := h.dbGetsByCustomerIDAndAddress(ctx, c.CustomerID, addr)
	if err != nil {
		log.Errorf("Could not get agents info. err:  %v", err)
		return errors.Wrapf(err, "could not get agents info. err: %v", err)
	}

	// Create a resource for each agent
	for _, a := range ags {
		log.Debugf("Creating resource for the agent. agent_id: %s", a.ID)
		r, err := h.resourceHandler.Create(ctx, c.CustomerID, a.ID, resource.TypeCall, c)
		if err != nil {
			log.Errorf("Could not create the resource. err: %v", err)
			continue
		}
		log.WithField("resource", r).Debugf("Created resource. resource_id: %s", r.ID)
	}

	return nil
}
