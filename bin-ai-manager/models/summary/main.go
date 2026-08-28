package summary

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type Summary struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
	OnEndFlowID  uuid.UUID `json:"on_end_flow_id,omitempty" db:"on_end_flow_id,uuid"`

	ReferenceType ReferenceType `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`

	Status   Status `json:"status,omitempty" db:"status"`
	Language string `json:"language,omitempty" db:"language"`
	Content  string `json:"content,omitempty" db:"content"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

type ReferenceType string

const (
	ReferenceTypeNone       ReferenceType = ""
	ReferenceTypeCall       ReferenceType = "call"
	ReferenceTypeConference ReferenceType = "conference"
	ReferenceTypeTranscribe ReferenceType = "transcribe"
	ReferenceTypeRecording  ReferenceType = "recording"
)

type Status string

const (
	StatusNone        Status = ""
	StatusProgressing Status = "progressing"
	StatusDone        Status = "done"
)
