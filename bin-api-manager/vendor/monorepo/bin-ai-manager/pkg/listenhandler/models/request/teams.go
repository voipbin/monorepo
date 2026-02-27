package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/team"
)

// V1DataTeamsPost is
// v1 data type request struct for
// /v1/teams POST
type V1DataTeamsPost struct {
	CustomerID    uuid.UUID     `json:"customer_id,omitempty"`
	Name          string        `json:"name,omitempty"`
	Detail        string        `json:"detail,omitempty"`
	StartMemberID uuid.UUID     `json:"start_member_id,omitempty"`
	Members       []team.Member `json:"members,omitempty"`
}

// V1DataTeamsIDPut is
// v1 data type request struct for
// /v1/teams/<team-id> PUT
type V1DataTeamsIDPut struct {
	Name          string        `json:"name,omitempty"`
	Detail        string        `json:"detail,omitempty"`
	StartMemberID uuid.UUID     `json:"start_member_id,omitempty"`
	Members       []team.Member `json:"members,omitempty"`
}
