package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
)

// processEventAMAgentDeleted handles the agent-manager's agent_deleted event.
func (h *subscribeHandler) processEventAMAgentDeleted(ctx context.Context, m *sock.Event) error {
	a := &amagent.Agent{}
	if err := json.Unmarshal([]byte(m.Data), a); err != nil {
		return fmt.Errorf("could not unmarshal the data. err: %w", err)
	}

	return h.deviceHandler.EventAgentDeleted(ctx, a)
}

// processEventCMCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCMCustomerDeleted(ctx context.Context, m *sock.Event) error {
	c := &cmcustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), c); err != nil {
		return fmt.Errorf("could not unmarshal the data. err: %w", err)
	}

	return h.deviceHandler.EventCustomerDeleted(ctx, c)
}
