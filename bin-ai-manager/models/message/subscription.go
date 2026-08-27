package message

// This file holds the eventtopic.SubscriptionIdentifier override for
// *IntermediateWebhookMessage, whose type is declared in webhook.go.
//
// Why it is NOT in webhook.go: by the root CLAUDE.md convention, a `models/<entity>/webhook.go`
// is the single source of truth for the externally documented wire shape -- the RST struct docs
// (`*_struct_*.rst`) are written against WebhookMessage and must be re-verified and rebuilt
// whenever that file changes. EventSubscriptionID adds no field and changes no serialized output,
// so routing it through the webhook.go review/rebuild surface would signal a wire-shape change
// that did not happen. Keeping the publish-side override in its own file leaves webhook.go as a
// pure wire-shape declaration. The method is still on the same type in the same package, so the
// method set and behavior are identical either way.

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404 / VOIP-1405 §2.2). It is the parent AIcallID, not the
// fragment's own ID: an intermediate message is a non-persisted streaming fragment whose ID is
// the per-delta id echoed from bin-pipecat-manager, minted anew for every event of the same
// utterance (the Sequence field orders them). Such an id is not an address anybody could bind to,
// and a well-formed but meaningless address is worse than none. Subscribers follow the AIcall,
// which is the same address `aimessage_created` resolves to.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *IntermediateWebhookMessage) EventSubscriptionID() string {
	return h.AIcallID.String()
}
