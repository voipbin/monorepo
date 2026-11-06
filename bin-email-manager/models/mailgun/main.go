package mailgun

import "encoding/json"

// Webhook event names of the Mailgun webhook payload.
const (
	EventAccepted     = "accepted"
	EventDelivered    = "delivered"
	EventFailed       = "failed"
	EventOpened       = "opened"
	EventClicked      = "clicked"
	EventUnsubscribed = "unsubscribed"
	EventComplained   = "complained"
)

// Severity values of the failed event.
const (
	SeverityTemporary = "temporary"
	SeverityPermanent = "permanent"
)

// UserVariableMessageID is the name of the user variable (sent as "v:<name>")
// that carries the voipbin email id. Mailgun returns it in "user-variables".
const UserVariableMessageID = "voipbin_message_id"

// Webhook is the body of a Mailgun webhook request. The event is either under
// "event-data" (next to "signature") or the body itself.
type Webhook struct {
	Signature *Signature `json:"signature,omitempty"`
	EventData *Event     `json:"event-data,omitempty"`
	Event
}

// Signature is the "signature" object of a Mailgun webhook request.
// Timestamp is kept raw because it is the exact text that was signed.
type Signature struct {
	Timestamp json.RawMessage `json:"timestamp"`
	Token     string          `json:"token"`
	Signature string          `json:"signature"`
}

// Event is the part of a Mailgun event used by the email manager.
type Event struct {
	Event    string `json:"event"`
	Severity string `json:"severity,omitempty"`
	Reason   string `json:"reason,omitempty"`

	// UserVariables is "[]" when empty and an object otherwise.
	UserVariables json.RawMessage `json:"user-variables,omitempty"`
}
