package conference

import uuid "github.com/satori/go.uuid"

// Conference type
type Conference struct {
	ID   uuid.UUID
	Type Type

	Name   string
	Detail string
	Data   []map[string]interface{}

	Bridges []uuid.UUID
	Calls   []uuid.UUID

	TMCreate string
	TMUpdate string
	TMDelete string
}

// Type conference types
type Type string

// List of conference types
const (
	TypeEcho       Type = "echo"       // conference for echoing.
	TypeTransfer   Type = "transfer"   // conference for simple transfer
	TypeConference Type = "conference" // conference for more than 3 calls join
)

// NewConference creates a new conference
func NewConference() *Conference {
	cf := &Conference{}

	return cf
}
