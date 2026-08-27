package call

import "github.com/gofrs/uuid"

// list of call event types
const (
	EventTypeCallCreated string = "call_created" // the call has created
	EventTypeCallUpdated string = "call_updated" // the call's info has updated
	EventTypeCallDeleted string = "call_deleted" // the call's info has deleted

	EventTypeCallDialing     string = "call_dialing"
	EventTypeCallRinging     string = "call_ringing"
	EventTypeCallProgressing string = "call_progressing"
	EventTypeCallTerminating string = "call_terminating"
	EventTypeCallCanceling   string = "call_canceling"
	EventTypeCallHangup      string = "call_hangup"

	EventTypeCallOutboundWhitelistRejected string = "call.outbound_whitelist_rejected"
)

// OutboundWhitelistRejectedEvent is the payload of EventTypeCallOutboundWhitelistRejected,
// published when an outbound PSTN destination is rejected by the customer's whitelist
// (pkg/callhandler/outgoing_call.go).
//
// It replaces the anonymous map[string]interface{} that used to be built inline at the publish
// site (VOIP-1405 §3.3). The JSON key SET is identical to that map -- only the order differs, as
// Go marshals maps key-sorted and structs in declaration order, which is JSON-semantically
// irrelevant. The reason for the type is the global topic exchange: a map can never satisfy the
// pointer-receiver eventtopic.SubscriptionIdentifier assertion, so the map payload could only ever
// resolve to the `-` placeholder address.
//
// Note there is deliberately NO top-level `id` field: this event is not a persisted resource, it
// is a rejection notice about one call attempt. The subscription address therefore MUST come from
// the override below, which is why the map form was unaddressable.
type OutboundWhitelistRejectedEvent struct {
	CallID             uuid.UUID `json:"call_id"`
	CustomerID         uuid.UUID `json:"customer_id"`
	DestinationCountry string    `json:"destination_country"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404 §4.2, VOIP-1405 §2.2). It is the rejected call's id: subscribers
// follow the call, and this event carries no id of its own to fall back to.
//
// The receiver is a pointer because the event data reaches notifyhandler as a POINTER and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type; a VALUE of this
// pointer-receiver type would fail the assertion (the exact pipecat defect this ticket fixed).
func (h *OutboundWhitelistRejectedEvent) EventSubscriptionID() string {
	return h.CallID.String()
}
