package agentcall

import "github.com/gofrs/uuid"

// AgentCall type
type AgentCall struct {
	ID          uuid.UUID `json:"id"` // call id
	CustomerID  uuid.UUID `json:"customer_id"`
	AgentID     uuid.UUID `json:"agent_id"` // agent's id
	AgentDialID uuid.UUID `json:"agent_dial_id"`
}
