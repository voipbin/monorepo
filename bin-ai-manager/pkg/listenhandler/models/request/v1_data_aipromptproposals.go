package request

import "github.com/gofrs/uuid"

// V1DataAIPromptProposalsPost is the body of POST /v1/aipromptproposals.
type V1DataAIPromptProposalsPost struct {
	CustomerID uuid.UUID   `json:"customer_id"`
	AIID       uuid.UUID   `json:"ai_id"`
	AuditIDs   []uuid.UUID `json:"audit_ids"`
	Language   string      `json:"language,omitempty"`
}

// V1DataAIPromptProposalsAcceptPost is the body of POST /v1/aipromptproposals/<id>/accept (and /reject).
type V1DataAIPromptProposalsAcceptPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
}
