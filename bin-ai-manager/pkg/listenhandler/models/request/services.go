package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/summary"
)

// V1DataServicesTypeAIcallPost is
// data type request struct for
// /v1/services/aicall POST
type V1DataServicesTypeAIcallPost struct {
	AIID uuid.UUID `json:"ai_id"`

	ActiveflowID  uuid.UUID            `json:"activeflow_id"`
	ReferenceType aicall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID            `json:"reference_id"`

	Resume bool `json:"resume"`

	Gender   aicall.Gender `json:"gender"`
	Language string        `json:"language"`
}

// V1DataServicesTypeSummaryPost is
// data type request struct for
// /v1/services/summary POST
type V1DataServicesTypeSummaryPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	OnEndFlowID  uuid.UUID `json:"on_end_flow_id,omitempty"`

	ReferenceType summary.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID             `json:"reference_id,omitempty"`

	Language string `json:"language,omitempty"`
}

// V1DataServicesTypeTaskPost is
// data type request struct for
// /v1/services/task POST
type V1DataServicesTypeTaskPost struct {
	AIID         uuid.UUID `json:"ai_id"`
	ActiveflowID uuid.UUID `json:"activeflow_id"`
}
