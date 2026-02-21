package variable

import "github.com/gofrs/uuid"

// Variable struct
type Variable struct {
	ID        uuid.UUID         `json:"id"` // same with the activeflow id.
	Variables map[string]string `json:"variables"`
}
