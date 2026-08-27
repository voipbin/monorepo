package eventtopic

import (
	"strings"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_RoutingKey(t *testing.T) {

	tests := []struct {
		name string

		publisher      string
		eventType      string
		subscriptionID string

		expectRes string
	}{
		{
			name: "normal",

			publisher:      "call-manager",
			eventType:      "call_created",
			subscriptionID: "4a539340-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "call-manager.call.4a539340-a1bc-11f1-92ef-60452e5e40a2.created",
		},
		{
			name: "event type has no underscore",

			publisher:      "call-manager",
			eventType:      "created",
			subscriptionID: "4a539340-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "call-manager.-.4a539340-a1bc-11f1-92ef-60452e5e40a2.created",
		},
		{
			name: "event type has multiple underscores",

			publisher:      "customer-manager",
			eventType:      "customer_balance_updated",
			subscriptionID: "b0e0b3a0-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "customer-manager.customer.b0e0b3a0-a1bc-11f1-92ef-60452e5e40a2.balance_updated",
		},
		{
			name: "event type contains a dot",

			publisher:      "call-manager",
			eventType:      "call.outbound_whitelist_rejected",
			subscriptionID: "c1f1c4b1-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "call-manager.call.c1f1c4b1-a1bc-11f1-92ef-60452e5e40a2.outbound_whitelist_rejected",
		},
		{
			name: "event type has an uppercase letter",

			publisher:      "storage-manager",
			eventType:      "Account_created",
			subscriptionID: "d2020501-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "storage-manager.account.d2020501-a1bc-11f1-92ef-60452e5e40a2.created",
		},
		{
			name: "event type contains amqp wildcards",

			publisher:      "call-manager",
			eventType:      "call_*_#_created",
			subscriptionID: "e3131612-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "call-manager.call.e3131612-a1bc-11f1-92ef-60452e5e40a2.____created",
		},
		{
			name: "event type is empty",

			publisher:      "call-manager",
			eventType:      "",
			subscriptionID: "f4242723-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "call-manager.-.f4242723-a1bc-11f1-92ef-60452e5e40a2.-",
		},
		{
			name: "subscription id is nil uuid",

			publisher:      "call-manager",
			eventType:      "call_created",
			subscriptionID: uuid.Nil.String(),

			expectRes: "call-manager.call.-.created",
		},
		{
			name: "subscription id is empty",

			publisher:      "call-manager",
			eventType:      "call_created",
			subscriptionID: "",

			expectRes: "call-manager.call.-.created",
		},
		{
			name: "publisher is empty",

			publisher:      "",
			eventType:      "call_created",
			subscriptionID: "05353834-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "-.call.05353834-a1bc-11f1-92ef-60452e5e40a2.created",
		},
		{
			name: "everything is empty",

			publisher:      "",
			eventType:      "",
			subscriptionID: "",

			expectRes: "-.-.-.-",
		},
		{
			name: "event type starts with an underscore",

			publisher:      "call-manager",
			eventType:      "_created",
			subscriptionID: "16464945-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "call-manager.-.16464945-a1bc-11f1-92ef-60452e5e40a2.created",
		},
		{
			name: "subscription id has an uppercase letter",

			publisher:      "transcribe-manager",
			eventType:      "transcript_created",
			subscriptionID: "27575A56-A1BC-11F1-92EF-60452E5E40A2",

			expectRes: "transcribe-manager.transcript.27575a56-a1bc-11f1-92ef-60452e5e40a2.created",
		},
		{
			name: "subscription id is exactly at the length limit",

			publisher:      "call-manager",
			eventType:      "call_created",
			subscriptionID: strings.Repeat("a", subscriptionIDMaxLen),

			expectRes: "call-manager.call." + strings.Repeat("a", subscriptionIDMaxLen) + ".created",
		},
		{
			name: "subscription id exceeds the length limit",

			publisher:      "call-manager",
			eventType:      "call_created",
			subscriptionID: strings.Repeat("a", subscriptionIDMaxLen+1),

			expectRes: "call-manager.call.-.created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := RoutingKey(tt.publisher, tt.eventType, tt.subscriptionID)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_PatternAll(t *testing.T) {

	tests := []struct {
		name string

		publisher string

		expectRes string
	}{
		{
			name: "normal",

			publisher: "transcribe-manager",

			expectRes: "transcribe-manager.#",
		},
		{
			name: "publisher is empty",

			publisher: "",

			expectRes: "-.#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := PatternAll(tt.publisher)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_PatternResource(t *testing.T) {

	tests := []struct {
		name string

		publisher string
		resource  string

		expectRes string
	}{
		{
			name: "normal",

			publisher: "transcribe-manager",
			resource:  "transcript",

			expectRes: "transcribe-manager.transcript.#",
		},
		{
			name: "resource has an uppercase letter",

			publisher: "storage-manager",
			resource:  "Account",

			expectRes: "storage-manager.account.#",
		},
		{
			name: "resource is empty",

			publisher: "transcribe-manager",
			resource:  "",

			expectRes: "transcribe-manager.-.#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := PatternResource(tt.publisher, tt.resource)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_PatternInstance(t *testing.T) {

	tests := []struct {
		name string

		publisher      string
		resource       string
		subscriptionID string

		expectRes string
	}{
		{
			name: "normal",

			publisher:      "transcribe-manager",
			resource:       "transcribe",
			subscriptionID: "9f01c3d2-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: "transcribe-manager.transcribe.9f01c3d2-a1bc-11f1-92ef-60452e5e40a2.#",
		},
		{
			name: "subscription id is nil uuid",

			publisher:      "transcribe-manager",
			resource:       "transcribe",
			subscriptionID: uuid.Nil.String(),

			expectRes: "transcribe-manager.transcribe.-.#",
		},
		{
			name: "subscription id is empty",

			publisher:      "transcribe-manager",
			resource:       "transcribe",
			subscriptionID: "",

			expectRes: "transcribe-manager.transcribe.-.#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := PatternInstance(tt.publisher, tt.resource, tt.subscriptionID)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_PatternAction(t *testing.T) {

	tests := []struct {
		name string

		publisher string
		resource  string
		action    string

		expectRes string
	}{
		{
			name: "normal",

			publisher: "transcribe-manager",
			resource:  "transcribe",
			action:    "speech_interim",

			expectRes: "transcribe-manager.transcribe.*.speech_interim",
		},
		{
			name: "action has an uppercase letter",

			publisher: "storage-manager",
			resource:  "account",
			action:    "Created",

			expectRes: "storage-manager.account.*.created",
		},
		{
			name: "action is empty",

			publisher: "transcribe-manager",
			resource:  "transcribe",
			action:    "",

			expectRes: "transcribe-manager.transcribe.*.-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := PatternAction(tt.publisher, tt.resource, tt.action)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

// Test_PatternInstance_matchesRoutingKey pins the invariant that the pattern builders and
// RoutingKey normalize identically -- a binding built for a resource instance must match the key
// the publisher actually generates for it.
func Test_PatternInstance_matchesRoutingKey(t *testing.T) {

	tests := []struct {
		name string

		publisher      string
		eventType      string
		subscriptionID string

		expectResource string
	}{
		{
			name: "normal",

			publisher:      "transcribe-manager",
			eventType:      "transcript_created",
			subscriptionID: "9f01c3d2-a1bc-11f1-92ef-60452e5e40a2",

			expectResource: "transcript",
		},
		{
			name: "uppercase event type",

			publisher:      "storage-manager",
			eventType:      "Account_created",
			subscriptionID: "9f01c3d2-a1bc-11f1-92ef-60452e5e40a2",

			expectResource: "Account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := RoutingKey(tt.publisher, tt.eventType, tt.subscriptionID)
			pattern := PatternInstance(tt.publisher, tt.expectResource, tt.subscriptionID)

			prefix := pattern[:len(pattern)-1] // drop the trailing "#"
			if len(key) < len(prefix) || key[:len(prefix)] != prefix {
				t.Errorf("Wrong match. the key does not match the pattern. pattern: %s, key: %s", pattern, key)
			}
		})
	}
}

// Test_IsPlaceholderSubscriptionID pins the predicate publishers meter with. It must agree with
// the key RoutingKey actually generates for the same address -- an address that silently becomes a
// placeholder without incrementing the placeholder metric is the exact blind spot design §4.2
// calls out (VOIP-1404 code-review round 2, F10).
func Test_IsPlaceholderSubscriptionID(t *testing.T) {

	tests := []struct {
		name string

		subscriptionID string

		expectRes bool
	}{
		{
			name: "valid uuid",

			subscriptionID: "4a539340-a1bc-11f1-92ef-60452e5e40a2",

			expectRes: false,
		},
		{
			name: "empty",

			subscriptionID: "",

			expectRes: true,
		},
		{
			name: "nil uuid",

			subscriptionID: uuid.Nil.String(),

			expectRes: true,
		},
		{
			name: "at the length limit",

			subscriptionID: strings.Repeat("a", subscriptionIDMaxLen),

			expectRes: false,
		},
		{
			name: "over the length limit",

			subscriptionID: strings.Repeat("a", subscriptionIDMaxLen+1),

			expectRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := IsPlaceholderSubscriptionID(tt.subscriptionID)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %t, got: %t", tt.expectRes, res)
			}

			// the predicate and the generated key must never disagree.
			key := RoutingKey("call-manager", "call_created", tt.subscriptionID)
			if hasPlaceholderAddress := key == "call-manager.call.-.created"; hasPlaceholderAddress != tt.expectRes {
				t.Errorf("Wrong match. the predicate disagrees with the generated key. key: %s, predicate: %t", key, tt.expectRes)
			}
		})
	}
}

// Test_PatternInstance_lengthGuardMatchesRoutingKey pins the key<->pattern invariant across the
// length guard: a subscriber that builds an instance binding from an oversized address must get
// the same placeholder segment the publisher puts in the key, not a pattern that matches nothing.
func Test_PatternInstance_lengthGuardMatchesRoutingKey(t *testing.T) {
	subscriptionID := strings.Repeat("b", subscriptionIDMaxLen+1)

	key := RoutingKey("transcribe-manager", "transcribe_created", subscriptionID)
	pattern := PatternInstance("transcribe-manager", "transcribe", subscriptionID)

	prefix := pattern[:len(pattern)-1] // drop the trailing "#"
	if len(key) < len(prefix) || key[:len(prefix)] != prefix {
		t.Errorf("Wrong match. the key does not match the pattern. pattern: %s, key: %s", pattern, key)
	}
}

// Test_RoutingKey_staysWithinAmqpLimit pins the reason the length guard exists: AMQP rejects a
// publish whose routing key exceeds 255 bytes, so no input may produce one.
func Test_RoutingKey_staysWithinAmqpLimit(t *testing.T) {
	const amqpRoutingKeyMaxLen = 255

	// a pathological address plus a long publisher and event type.
	res := RoutingKey(
		"some-very-long-publisher-service-name",
		"resource_"+strings.Repeat("action_", 10),
		strings.Repeat("c", 4096),
	)

	if len(res) > amqpRoutingKeyMaxLen {
		t.Errorf("Wrong match. the routing key exceeds the amqp limit. len: %d, key: %s", len(res), res)
	}
}
