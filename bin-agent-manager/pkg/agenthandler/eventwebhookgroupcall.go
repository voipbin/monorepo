package agenthandler

// // webhookGroupcallCreated handles the groupcall_created event for resources.
// // It creates a resource for each agent associated with the groupcall's address.
// //
// // Parameters:
// // ctx (context.Context): The context for the request.
// // c (*cmgroupcall.Groupcall): The groupcall object.
// //
// // Returns:
// // error: An error if any occurred during the operation, otherwise nil.
// func (h *agentHandler) webhookGroupcallCreated(ctx context.Context, c *cmgroupcall.Groupcall) error {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":      "webhookGroupcallCreated",
// 		"groupcall": c,
// 	})
// 	log.Debugf("Creating resource for the groupcall. groupcall_id: %s", c.ID)

// 	// Determine the address based on the call's direction
// 	for _, addr := range c.Destinations {
// 		if addr.Type != commonaddress.TypeExtension && addr.Type != commonaddress.TypeTel {
// 			continue
// 		}

// 		// Get agents associated with the call's address
// 		ags, err := h.dbGetsByCustomerIDAndAddress(ctx, c.CustomerID, addr)
// 		if err != nil {
// 			log.Errorf("Could not get agents info. err:  %v", err)
// 			return errors.Wrapf(err, "could not get agents info. err: %v", err)
// 		}
// 		log.WithField("agents", ags).Debugf("Found agent list. len: %d", len(ags))

// 		// Create a resource for each agent
// 		for _, a := range ags {
// 			log.Debugf("Creating resource for the agent. agent_id: %s", a.ID)
// 			r, err := h.resourceHandler.Create(ctx, c.CustomerID, a.ID, resource.ReferenceTypeGroupcall, c.ID, c)
// 			if err != nil {
// 				log.Errorf("Could not create the resource. err: %v", err)
// 				continue
// 			}
// 			log.WithField("resource", r).Debugf("Created resource. resource_id: %s", r.ID)
// 		}
// 	}

// 	return nil
// }

// // webhookGroupcallUpdated handles the groupcall_(update) event for resources.
// // It creates a resource for each agent associated with the groupcall's address.
// //
// // Parameters:
// // ctx (context.Context): The context for the request.
// // c (*cmgroupcall.Groupcall): The groupcall object.
// //
// // Returns:
// // error: An error if any occurred during the operation, otherwise nil.
// func (h *agentHandler) webhookGroupcallUpdated(ctx context.Context, c *cmgroupcall.Groupcall) error {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":      "webhookGroupcallUpdated",
// 		"groupcall": c,
// 	})
// 	log.Debugf("Updating resource for the groupcall. groupcall_id: %s", c.ID)

// 	// get resources
// 	filters := map[string]string{
// 		"customer_id":    c.CustomerID.String(),
// 		"reference_type": string(resource.ReferenceTypeGroupcall),
// 		"reference_id":   c.ID.String(),
// 		"deleted":        "false",
// 	}

// 	// get related resources
// 	rs, err := h.resourceHandler.Gets(ctx, 100, "", filters)
// 	if err != nil {
// 		log.Errorf("Could not get resources. err: %v", err)
// 		return nil
// 	}

// 	// update resources
// 	for _, r := range rs {
// 		log.WithField("resource", r).Debugf("Updating resource info. resource_id: %s", r.ID)
// 		tmp, err := h.resourceHandler.UpdateData(ctx, r.ID, c)
// 		if err != nil {
// 			log.Errorf("Could not update the resource info. err: %v", err)
// 			continue
// 		}
// 		log.WithField("resource", tmp).Debugf("Updated resource info. resource_id: %s", tmp.ID)
// 	}

// 	return nil
// }
