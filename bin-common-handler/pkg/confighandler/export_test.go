package confighandler

import (
	"time"

	"github.com/gofrs/uuid"
)

// SetClock makes the timestamps and history ids of a handler deterministic in tests.
func SetClock(h Handler, now func() time.Time, newID func() uuid.UUID) {
	hh := h.(*handler)
	hh.now = now
	hh.newID = newID
}
