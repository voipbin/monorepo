package utilhandler

import "github.com/gofrs/uuid"

// CreateUUID returns a new uuid v4
func (h *utilHandler) CreateUUID() uuid.UUID {
	return CreateUUID()
}

// CreateUUID returns a new uuid v4
func CreateUUID() uuid.UUID {
	return uuid.Must(uuid.NewV4())
}
