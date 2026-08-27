package dtmf

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type DTMF struct {
	commonidentity.Identity

	CallID   uuid.UUID `json:"call_id,omitempty"`
	Digit    string    `json:"digit,omitempty"`
	Duration int       `json:"duration,omitempty"` // in milliseconds

	TMCreate *time.Time `json:"tm_create,omitempty"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404 §4.2, VOIP-1405 §2.2). It is the parent CallID, not the DTMF's
// own ID: pkg/callhandler.digitNotifyDTMFEvent mints a fresh uuid for every single digit event, so
// the own ID is not an address anybody could bind to in advance. Subscribers follow one call, and
// every digit of that call carries the same call-id.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type -- a value receiver would
// silently never be picked up.
func (h *DTMF) EventSubscriptionID() string {
	return h.CallID.String()
}
