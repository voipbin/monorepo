package utilhandler

import "github.com/gofrs/uuid"

// UUIDCreate returns a new uuid v4
func (h *utilHandler) UUIDCreate() uuid.UUID {
	return UUIDCreate()
}

// UUIDCreate returns a new uuid v4
func UUIDCreate() uuid.UUID {
	return uuid.Must(uuid.NewV4())
}
