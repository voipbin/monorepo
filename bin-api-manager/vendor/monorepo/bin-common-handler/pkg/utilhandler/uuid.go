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

// NewV5UUID returns a deterministic UUID v5 derived from the given namespace and data.
// This is useful for generating reproducible IDs (e.g., for idempotency checks)
// where the same inputs must always produce the same UUID across all callers.
func (h *utilHandler) NewV5UUID(namespace uuid.UUID, data string) uuid.UUID {
	return uuid.NewV5(namespace, data)
}
