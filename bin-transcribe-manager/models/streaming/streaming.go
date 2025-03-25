package streaming

import (
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// Streaming defines current streaming detail
type Streaming struct {
	commonidentity.Identity

	TranscribeID uuid.UUID            `json:"transcribe_id"`
	Language     string               `json:"language"`
	Direction    transcript.Direction `json:"direction"`
}
