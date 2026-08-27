package message

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
