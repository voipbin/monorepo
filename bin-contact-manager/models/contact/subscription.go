package contact

// This file holds the eventtopic.SubscriptionIdentifier implementation for *WebhookMessage,
// whose type is declared in webhook.go.
//
// Why it is NOT in webhook.go: by the root CLAUDE.md convention, a `models/<entity>/webhook.go`
// is the single source of truth for the externally documented wire shape -- the RST struct docs
// (`*_struct_*.rst`) are written against WebhookMessage and must be re-verified and rebuilt
// whenever that file changes. EventSubscriptionID adds no field and changes no serialized output,
// so routing it through the webhook.go review/rebuild surface would signal a wire-shape change
// that did not happen. Keeping the publish-side method in its own file leaves webhook.go as a
// pure wire-shape declaration (precedent: bin-ai-manager models/message/subscription.go,
// VOIP-1405).

// EventSubscriptionID returns the subscription address of this type on the global topic
// exchange `bin-manager.event`: the resource's own id (VOIP-1404 §4.2, VOIP-1419).
func (h *WebhookMessage) EventSubscriptionID() string {
	return h.ID.String()
}
