package summary

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type Summary struct {
	commonidentity.Identity

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	Status   Status `json:"status,omitempty"`
	Language string `json:"language,omitempty"`
	Content  string `json:"content,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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
