package request

import (
	"monorepo/bin-ai-manager/models/summary"

	"github.com/gofrs/uuid"
)

// V1DataSummariesPost is
// v1 data type request struct for
// /v1/summaries POST
type V1DataSummariesPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	OnEndFlowID  uuid.UUID `json:"on_end_flow_id,omitempty"`

	ReferenceType summary.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID             `json:"reference_id,omitempty"`

	Language string `json:"language,omitempty"`
}
