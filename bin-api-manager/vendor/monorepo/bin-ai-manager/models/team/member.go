package team

import "github.com/gofrs/uuid"

// Member represents a node in the team graph, backed by an existing AI config.
type Member struct {
	ID          uuid.UUID    `json:"id,omitempty"`
	Name        string       `json:"name,omitempty"`
	AIID        uuid.UUID    `json:"ai_id,omitempty"`
	Transitions []Transition `json:"transitions,omitempty"`
}

// Transition defines an LLM function that triggers a switch to another member.
type Transition struct {
	FunctionName string    `json:"function_name,omitempty"`
	Description  string    `json:"description,omitempty"`
	NextMemberID uuid.UUID `json:"next_member_id,omitempty"`
}
