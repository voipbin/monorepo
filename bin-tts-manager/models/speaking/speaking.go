package speaking

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
)

// Speaking represents an active streaming TTS session attached to a call or conference.
type Speaking struct {
	commonidentity.Identity

	ReferenceType streaming.ReferenceType `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID               `json:"reference_id"   db:"reference_id,uuid"`
	Language      string                  `json:"language"        db:"language"`
	Provider      string                  `json:"provider"        db:"provider"`
	VoiceID       string                  `json:"voice_id"        db:"voice_id"`
	Direction     streaming.Direction     `json:"direction"       db:"direction"`
	Status        Status                  `json:"status"          db:"status"`
	PodID         string                  `json:"pod_id"          db:"pod_id"`

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}
