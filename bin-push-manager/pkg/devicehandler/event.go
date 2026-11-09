package devicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
)

// EventCustomerDeleted handles the customer-manager's customer_deleted event: the devices of the customer
// are deleted.
func (h *deviceHandler) EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error {
	if err := h.db.DeviceDeleteByCustomerID(ctx, c.ID); err != nil {
		return fmt.Errorf("could not delete the devices of the customer. err: %w", err)
	}

	return nil
}

// EventAgentDeleted handles the agent-manager's agent_deleted event: the devices of the agent are deleted.
func (h *deviceHandler) EventAgentDeleted(ctx context.Context, a *amagent.Agent) error {
	if err := h.db.DeviceDeleteByAgentID(ctx, a.ID); err != nil {
		return fmt.Errorf("could not delete the devices of the agent. err: %w", err)
	}

	return nil
}
