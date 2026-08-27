// Package eventtopic holds the routing-key schema of the global topic exchange
// `bin-manager.event` (VOIP-1404).
//
// The schema is `<publisher>.<resource>.<subscription-id>.<action>`, one event = one key = one
// publish. Publishers generate keys through RoutingKey; subscribers build their bindings through
// the Pattern* builders and the existing sockhandler.QueueBind/QueueUnbind. Everything in this
// package is a pure function, so a key and the pattern meant to match it are always normalized
// the same way.
//
// # Admission rule
//
// bin-common-handler admits a package only when 3+ services use it, and at pilot time only
// transcribe-manager does. This package is admitted anyway because it is internal plumbing of
// notifyhandler itself -- a shared-library component that every service already links -- and the
// 3+-services rule targets service-level utilities. The routing key must be generated inside
// notifyhandler.publishEvent, so the schema cannot live in a service package without inverting
// the dependency. Follow-up A opts every remaining publisher into the same package. See design
// doc §4.3 (2026-08-27-voip-1404-global-topic-exchange-design.md).
//
// # Shared-binding caveat
//
// A broker binding is shared by every logical subscriber that uses the same queue + pattern. One
// QueueUnbind severs all of them, and QueueBind is idempotent in the reconnect-tracking list.
// Callers that multiplex several logical subscribers over one queue must keep refcount discipline
// and only bind/unbind on the 0<->1 transition, the way
// bin-api-manager/pkg/websockhandler/scoperefcount.go does. A shared refcount helper is a
// follow-up, not part of this package.
package eventtopic

// SubscriptionIdentifier is the opt-in override for the third routing-key segment.
//
// The third segment answers "by which ID will subscribers address this stream?" -- a subscription
// address, not necessarily the resource's own id. The default (no implementation) is the
// top-level `id` of the marshaled event payload, which is correct for a resource that subscribers
// address by its own id. Stream-child resources must override: a per-utterance or per-result id
// that is newly generated for every event is not an address, because nobody can bind to it in
// advance. A non-Nil but meaningless id is worse than no id -- it produces well-formed keys that
// match nothing and evades the placeholder metric. Example: streaming.Speech overrides with its
// parent TranscribeID.
//
// Implement it with a POINTER receiver. Event data reaches notifyhandler.PublishEvent as a
// pointer (e.g. *transcript.Transcript), and the type assertion matches the dynamic type, so a
// value receiver would silently never be picked up.
//
// An empty return value, or the uuid.Nil string, falls back to the `-` placeholder exactly like
// an absent id.
type SubscriptionIdentifier interface {
	EventSubscriptionID() string
}
