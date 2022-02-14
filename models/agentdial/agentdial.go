package agentdial

import "github.com/gofrs/uuid"

// AgentDial type
type AgentDial struct {
	ID           uuid.UUID   `json:"id"`
	CustomerID   uuid.UUID   `json:"customer_id"`
	AgentID      uuid.UUID   `json:"agent_id"` // agent's id
	AgentCallIDs []uuid.UUID `json:"agent_call_ids"`
}
