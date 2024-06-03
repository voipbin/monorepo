package address

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// Address represents common address to agent ids
type Address struct {
	Address  commonaddress.Address `json:"address"`
	AgentIDs []uuid.UUID           `json:"agent_ids"`
}
