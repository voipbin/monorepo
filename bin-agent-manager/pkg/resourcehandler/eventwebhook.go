package resourcehandler

import (
	"context"
	"monorepo/bin-agent-manager/models/resource"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// eventWebhookCallCreated handles the call-manager's call_created event.
func (h *resourceHandler) eventWebhookCallCreated(ctx context.Context, c *cmcall.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "eventWebhookCallCreated",
		"call": c,
	})
	log.Debugf("Creating resource for the call. call_id: %s", c.ID)

	// // Determine the address based on the call's direction
	// addr := c.Source
	// if c.Direction == cmcall.DirectionOutgoing {
	// 	addr = c.Destination
	// }
	// log.WithField("address", addr).Debugf("Found call address. address_type: %s, address_target: %s", addr.Type, addr.Target)

	// // Get agents associated with the call's address
	// ags, err := h.agentHandler.GetByCustomerIDAndAddress(ctx, c.CustomerID, addr)
	// if err != nil {
	// 	log.Errorf("Could not get agents info. err:  %v", err)
	// 	return errors.Wrapf(err, "could not get agents info. err: %v", err)
	// }
	// log.WithField("agents", ags).Debugf("Found agents informations. len: %d", len(ags))

	// // Create a resource for each agent
	// for _, a := range ags {
	// 	log.Debugf("Creating resource for the agent. agent_id: %s", a.ID)
	// 	r, err := h.Create(ctx, c.CustomerID, a.ID, resource.ReferenceTypeCall, c.ID, c)
	// 	if err != nil {
	// 		log.Errorf("Could not create the resource. err: %v", err)
	// 		continue
	// 	}
	// 	log.WithField("resource", r).Debugf("Created resource. resource_id: %s", r.ID)
	// }

	return nil
}

// eventWebhookCallUpdated handles the call-manager's call_(updated) event.
func (h *resourceHandler) eventWebhookCallUpdated(ctx context.Context, c *cmcall.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "eventWebhookCallUpdated",
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

	return h.updataDataWithFilter(ctx, c, filters)
}

// eventWebhookCallDeleted handles the call-manager's call_deleted event.
func (h *resourceHandler) eventWebhookCallDeleted(ctx context.Context, c *cmcall.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "eventWebhookCallDeleted",
		"call": c,
	})
	log.Debugf("Deleting resource for the call. call_id: %s", c.ID)

	// gets the resources
	filters := map[string]string{
		"customer_id":    c.CustomerID.String(),
		"reference_type": string(resource.ReferenceTypeCall),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}

	return h.deleteWithFilter(ctx, filters)
}

// eventWebhookGroupcallCreated handles the groupcall_created event for resources.
func (h *resourceHandler) eventWebhookGroupcallCreated(ctx context.Context, c *cmgroupcall.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventWebhookGroupcallCreated",
		"groupcall": c,
	})
	log.Debugf("Creating resource for the groupcall. groupcall_id: %s", c.ID)

	// // Determine the address based on the call's direction
	// for _, addr := range c.Destinations {
	// 	if addr.Type != commonaddress.TypeExtension && addr.Type != commonaddress.TypeTel {
	// 		continue
	// 	}

	// 	// Get agents associated with the call's address
	// 	ags, err := h.agentHandler.GetByCustomerIDAndAddress(ctx, c.CustomerID, addr)
	// 	if err != nil {
	// 		log.Errorf("Could not get agents info. err:  %v", err)
	// 		return errors.Wrapf(err, "could not get agents info. err: %v", err)
	// 	}
	// 	log.WithField("agents", ags).Debugf("Found agent list. len: %d", len(ags))

	// 	// Create a resource for each agent
	// 	for _, a := range ags {
	// 		log.Debugf("Creating resource for the agent. agent_id: %s", a.ID)
	// 		r, err := h.Create(ctx, c.CustomerID, a.ID, resource.ReferenceTypeGroupcall, c.ID, c)
	// 		if err != nil {
	// 			log.Errorf("Could not create the resource. err: %v", err)
	// 			continue
	// 		}
	// 		log.WithField("resource", r).Debugf("Created resource. resource_id: %s", r.ID)
	// 	}
	// }

	return nil
}

// eventWebhookGroupcallUpdated handles the groupcall_(update) event for resources.
func (h *resourceHandler) eventWebhookGroupcallUpdated(ctx context.Context, c *cmgroupcall.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventWebhookGroupcallUpdated",
		"groupcall": c,
	})
	log.Debugf("Updating resource for the groupcall. groupcall_id: %s", c.ID)

	// get resources
	filters := map[string]string{
		"customer_id":    c.CustomerID.String(),
		"reference_type": string(resource.ReferenceTypeGroupcall),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}

	return h.updataDataWithFilter(ctx, c, filters)
}

// eventWebhookGroupcallDeleted handles the call-manager's groupcall_deleted event.
func (h *resourceHandler) eventWebhookGroupcallDeleted(ctx context.Context, c *cmgroupcall.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "eventWebhookGroupcallDeleted",
		"call": c,
	})
	log.Debugf("Deleting resource for the groupcall. call_id: %s", c.ID)

	// gets the resources
	filters := map[string]string{
		"customer_id":    c.CustomerID.String(),
		"reference_type": string(resource.ReferenceTypeGroupcall),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}

	return h.deleteWithFilter(ctx, filters)
}

// eventWebhookChatroomCreated handles the chatroom_created event for resources.
func (h *resourceHandler) eventWebhookChatroomCreated(ctx context.Context, c *chatchatroom.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventWebhookChatroomCreated",
		"groupcall": c,
	})
	log.Debugf("Creating resource for the chatroom. chatroom_id: %s", c.ID)

	_, err := h.Create(ctx, c.CustomerID, c.AgentID, resource.ReferenceTypeChatroom, c.ID, c)
	if err != nil {
		log.Errorf("Could not create the resource. err: %v", err)
		return errors.Wrapf(err, "could not create the resource. err: %v", err)
	}

	return nil
}

// eventWebhookChatroomUpdated handles the chatroom_updated event for resources.
func (h *resourceHandler) eventWebhookChatroomUpdated(ctx context.Context, c *chatchatroom.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventWebhookChatroomUpdated",
		"groupcall": c,
	})
	log.Debugf("Updating resource for the chatroom. chatroom_id: %s", c.ID)

	// get resources
	filters := map[string]string{
		"customer_id":    c.CustomerID.String(),
		"reference_type": string(resource.ReferenceTypeChatroom),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}

	return h.updataDataWithFilter(ctx, c, filters)
}

// eventWebhookChatroomDeleted handles the chatroom_deleted event for resources.
func (h *resourceHandler) eventWebhookChatroomDeleted(ctx context.Context, c *chatchatroom.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventWebhookChatroomDeleted",
		"groupcall": c,
	})
	log.Debugf("Deleting resource for the chatroom. chatroom_id: %s", c.ID)

	// get resources
	filters := map[string]string{
		"customer_id":    c.CustomerID.String(),
		"reference_type": string(resource.ReferenceTypeChatroom),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}

	return h.deleteWithFilter(ctx, filters)
}

// eventWebhookMessagechatroomCreated handles the messagechatroom_created event for resources.
func (h *resourceHandler) eventWebhookMessagechatroomCreated(ctx context.Context, c *chatmessagechatroom.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventWebhookMessagechatroomCreated",
		"groupcall": c,
	})
	log.Debugf("Creating resource for the messagechatroom. messagechatroom_id: %s", c.ID)

	_, err := h.Create(ctx, c.CustomerID, c.AgentID, resource.ReferenceTypeMessagechatroom, c.ID, c)
	if err != nil {
		log.Errorf("Could not create the resource. err: %v", err)
		return errors.Wrapf(err, "could not create the resource. err: %v", err)
	}

	return nil
}

// eventWebhookMessagechatroomUpdated handles the messagechatroom_updated event for resources.
func (h *resourceHandler) eventWebhookMessagechatroomUpdated(ctx context.Context, c *chatmessagechatroom.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventWebhookMessagechatroomUpdated",
		"groupcall": c,
	})
	log.Debugf("Updating resource for the messagechatroom. messagechatroom_id: %s", c.ID)

	// get resources
	filters := map[string]string{
		"customer_id":    c.CustomerID.String(),
		"reference_type": string(resource.ReferenceTypeMessagechatroom),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}

	return h.updataDataWithFilter(ctx, c, filters)
}

// eventWebhookMessagechatroomDeleted handles the messagechatroom_deleted event for resources.
func (h *resourceHandler) eventWebhookMessagechatroomDeleted(ctx context.Context, c *chatmessagechatroom.WebhookMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "eventWebhookMessagechatroomDeleted",
		"groupcall": c,
	})
	log.Debugf("Deleting resource for the messagechatroom. messagechatroom_id: %s", c.ID)

	// get resources
	filters := map[string]string{
		"customer_id":    c.CustomerID.String(),
		"reference_type": string(resource.ReferenceTypeMessagechatroom),
		"reference_id":   c.ID.String(),
		"deleted":        "false",
	}

	return h.deleteWithFilter(ctx, filters)
}
