package agentdial

import "github.com/gofrs/uuid"

// AgentDial type
type AgentDial struct {
	AgentID uuid.UUID   `json:"agent_id"` // agent's id
	CallIDs []uuid.UUID `json:"call_ids"`
}
