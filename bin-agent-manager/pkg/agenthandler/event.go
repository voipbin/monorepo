package agenthandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/models/agent"
)

// EventGroupcallCreated handles the call-manager's groupcall_created event
func (h *agentHandler) EventGroupcallCreated(ctx context.Context, c *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventGroupcallCreated",
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

// EventGroupcallProgressing handles the call-manager's groupcall_progressing event
func (h *agentHandler) EventGroupcallProgressing(ctx context.Context, c *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventGroupcallProgressing",
		"groupcall": c,
	})

	for _, destination := range c.Destinations {
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
	filters := map[agent.Field]any{
		agent.FieldCustomerID: cu.ID,
		agent.FieldDeleted:    false,
	}
	ags, err := h.List(ctx, 1000, h.utilHandler.TimeGetCurTime(), filters)
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

// EventCustomerCreated handles the customer-manager's customer_created event
func (h *agentHandler) EventCustomerCreated(ctx context.Context, cu *cmcustomer.Customer, headless bool) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerCreated",
		"customer": cu,
	})
	log.Debugf("Creating basic customer admin agent for new customer. customer_id: %s", cu.ID)

	// generate random unusable password
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Errorf("Could not generate random password. err: %v", err)
		return errors.Wrap(err, "could not generate random password")
	}
	randomPassword := hex.EncodeToString(randomBytes)

	a, err := h.Create(
		ctx,
		cu.ID,
		cu.Email,
		randomPassword,
		"default admin",
		"default agent account for admin permission",
		agent.RingMethodRingAll,
		agent.PermissionCustomerAdmin,
		[]uuid.UUID{},
		[]commonaddress.Address{},
	)
	if err != nil {
		log.Errorf("Could not create basic customer admin agent. err: %v", err)
		return errors.Wrap(err, "could not create basic customer admin agent")
	}
	log.WithField("agent", a).Debugf("Created basic admin agent for new customer. agent_id: %s", a.ID)

	// send welcome email with password reset link — only for non-headless signups
	if !headless {
		if err := h.PasswordForgot(ctx, cu.Email, PasswordResetEmailTypeWelcome); err != nil {
			log.Errorf("Could not send welcome email. err: %v", err)
			// don't fail the event - the agent was created successfully
		}
	} else {
		log.Debugf("Skipping welcome email for headless signup. customer_id: %s", cu.ID)
	}

	return nil
}
