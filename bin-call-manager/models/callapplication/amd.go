package callapplication

import "github.com/gofrs/uuid"

// list of AMD machine handles
const (
	AMDMachineHandleHangup   = "hangup"
	AMDMachineHandleContinue = "continue"
)

// AMD defines
type AMD struct {
	CallID        uuid.UUID `json:"call_id"`
	MachineHandle string    `json:"machine_handle"`
	Async         bool      `json:"async"`
}
