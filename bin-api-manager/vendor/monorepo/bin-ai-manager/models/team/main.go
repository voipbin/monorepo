package team

import (
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
)

// Team represents a group of AI members organized as a directed graph.
// Each member is backed by an existing AI config, and transitions between
// members are driven by LLM function calling at runtime.
type Team struct {
	identity.Identity

	Name          string    `json:"name,omitempty" db:"name"`
	Detail        string    `json:"detail,omitempty" db:"detail"`
	StartMemberID uuid.UUID `json:"start_member_id,omitempty" db:"start_member_id,uuid"`
	Members       []Member       `json:"members,omitempty" db:"members,json"`
	Parameter     map[string]any `json:"parameter,omitempty" db:"parameter,json"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
