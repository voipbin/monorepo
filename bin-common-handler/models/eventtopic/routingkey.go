package eventtopic

import (
	"strings"

	"github.com/gofrs/uuid"
)

const (
	// separator is the AMQP routing-key segment separator.
	separator = "."

	// placeholder replaces a segment that has no meaningful value. Type-level bindings
	// (`<publisher>.<resource>.#`) still match a placeholder key; instance bindings never do,
	// which is correct because no valid subscription address exists.
	placeholder = "-"

	// subscriptionIDMaxLen bounds the subscription-address segment. AMQP caps a whole routing key
	// at 255 bytes and the broker rejects a publish that exceeds it, so an unbounded address turns
	// a routing concern into a publish failure. Every address today is a 36-byte uuid; 64 leaves
	// generous headroom for a future publisher that addresses its stream by some other stable id,
	// while keeping the worst-case key far below the AMQP limit. An address longer than this is
	// treated exactly like an absent one -- see normalizeSubscriptionID.
	subscriptionIDMaxLen = 64
)

// segmentReplacer removes every AMQP-significant character from a computed segment. `.` would
// create an extra segment, `*` and `#` would turn a published key into an accidental wildcard.
var segmentReplacer = strings.NewReplacer(".", "_", "*", "_", "#", "_")

// RoutingKey builds the routing key of the global topic exchange for the given event:
//
//	<publisher>.<resource>.<subscription-id>.<action>
//	transcribe-manager.transcript.9f01c3d2-....created
//
// The event type is normalized first (lowercased, every `.` replaced with `_`) and then split on
// the first `_` into resource and action. A type without any `_` becomes the action, and the
// resource falls back to the placeholder. An empty or uuid.Nil subscriptionID becomes the
// placeholder as well.
//
// The split is purely mechanical, mirroring bin-webhook-manager's routingkey.go: a
// multi-underscore type such as `customer_balance_updated` splits into `customer` /
// `balance_updated`. What matters for binding is that the generated keys are deterministic and
// stable, not that every segment is semantically perfect.
func RoutingKey(publisher string, eventType string, subscriptionID string) string {
	normalized := strings.ReplaceAll(strings.ToLower(eventType), ".", "_")

	resource := ""
	action := normalized
	if tmps := strings.SplitN(normalized, "_", 2); len(tmps) == 2 {
		resource = tmps[0]
		action = tmps[1]
	}

	return strings.Join([]string{
		normalizeSegment(publisher),
		normalizeSegment(resource),
		normalizeSubscriptionID(subscriptionID),
		normalizeSegment(action),
	}, separator)
}

// PatternAll returns the binding pattern for every event of the given publisher.
func PatternAll(publisher string) string {
	return strings.Join([]string{normalizeSegment(publisher), "#"}, separator)
}

// PatternResource returns the binding pattern for every event of the given publisher's resource.
func PatternResource(publisher string, resource string) string {
	return strings.Join([]string{normalizeSegment(publisher), normalizeSegment(resource), "#"}, separator)
}

// PatternInstance returns the binding pattern for every event of one resource instance, addressed
// by its subscription id.
func PatternInstance(publisher string, resource string, subscriptionID string) string {
	return strings.Join([]string{
		normalizeSegment(publisher),
		normalizeSegment(resource),
		normalizeSubscriptionID(subscriptionID),
		"#",
	}, separator)
}

// PatternAction returns the binding pattern for one action of the given publisher's resource,
// across every instance.
func PatternAction(publisher string, resource string, action string) string {
	return strings.Join([]string{
		normalizeSegment(publisher),
		normalizeSegment(resource),
		"*",
		normalizeSegment(action),
	}, separator)
}

// normalizeSegment makes the given value safe and deterministic as a single routing-key segment.
// The pattern builders use the very same normalization as RoutingKey, so a pattern always matches
// the keys it was built for.
func normalizeSegment(segment string) string {
	res := segmentReplacer.Replace(strings.ToLower(segment))
	if res == "" {
		return placeholder
	}

	return res
}

// normalizeSubscriptionID normalizes the subscription-address segment. An absent address and an
// all-zero uuid are the same thing to a subscriber -- there is nothing to bind to -- so both
// become the placeholder. Mapping uuid.Nil to the placeholder is what makes the VOIP-1258 all-Nil
// failure mode observable through the placeholder metric.
//
// An address longer than subscriptionIDMaxLen becomes the placeholder too. No publisher produces
// one today (every address is a uuid), but the guard keeps a future non-uuid address from pushing
// the key past the AMQP 255-byte limit, where the broker would reject the publish outright. A
// placeholder key is a degraded routing outcome; a failed publish is a lost event.
func normalizeSubscriptionID(subscriptionID string) string {
	if subscriptionID == "" || subscriptionID == uuid.Nil.String() || len(subscriptionID) > subscriptionIDMaxLen {
		return placeholder
	}

	return normalizeSegment(subscriptionID)
}

// IsPlaceholderSubscriptionID reports whether the given subscription address collapses to the `-`
// placeholder segment. Publishers meter their placeholder rate through this helper so the rules
// that produce a placeholder (absent, uuid.Nil, oversized) stay owned by this package alone --
// duplicating them at the call site is how an address silently becomes a placeholder without ever
// incrementing the metric that exists to make exactly that visible (design §4.2).
func IsPlaceholderSubscriptionID(subscriptionID string) bool {
	return normalizeSubscriptionID(subscriptionID) == placeholder
}
