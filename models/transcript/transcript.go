package transcript

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
)

// Transcript struct
type Transcript struct {
	ID           uuid.UUID        `json:"id"`
	CustomerID   uuid.UUID        `json:"customer_id"`
	TranscribeID uuid.UUID        `json:"transcribe_id"`
	Direction    common.Direction `json:"direction"` // direction. in/out
	Message      string           `json:"message"`   // message

	TMCreate string `json:"tm_create"` // timestamp
}
