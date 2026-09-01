package container

import (
	"encoding/json"
	"testing"

	"monorepo/bin-common-handler/models/eventtopic"
)

// The publish path resolves the subscription address by asserting this interface, so the
// assertion must hold at compile time, not at the first publish.
var _ eventtopic.SubscriptionIdentifier = (*Event)(nil)

func Test_EventTypeConstants(t *testing.T) {
	tests := []struct {
		name string

		eventType string

		expect string
	}{
		{
			name: "container_started",

			eventType: EventTypeContainerStarted,

			expect: "container_started",
		},
		{
			name: "container_died",

			eventType: EventTypeContainerDied,

			expect: "container_died",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.eventType != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, tt.eventType)
			}
		})
	}
}

func Test_EventTypeUniqueness(t *testing.T) {
	eventTypes := map[string]bool{
		EventTypeContainerStarted: true,
		EventTypeContainerDied:    true,
	}

	if len(eventTypes) != 2 {
		t.Errorf("Wrong match. expect: 2 unique event types, got: %d", len(eventTypes))
	}
}

func Test_ServiceConstants(t *testing.T) {
	tests := []struct {
		name string

		service string

		expect string
	}{
		{
			name: "asterisk_call",

			service: ServiceAsteriskCall,

			expect: "asterisk-call",
		},
		{
			name: "asterisk_conference",

			service: ServiceAsteriskConference,

			expect: "asterisk-conference",
		},
		{
			name: "asterisk_registrar",

			service: ServiceAsteriskRegistrar,

			expect: "asterisk-registrar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.service != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, tt.service)
			}
		})
	}
}

// Test_EventSubscriptionID is mutation-checked on purpose: two DISTINCT asterisk-ids must resolve
// to their own values, so a hardcoded return (or a return of the wrong field) cannot pass.
func Test_EventSubscriptionID(t *testing.T) {
	tests := []struct {
		name string

		event *Event

		expect string
	}{
		{
			name: "resolved_asterisk_id",

			event: &Event{
				ContainerName: "voip-asterisk-call-docker-1",
				Service:       ServiceAsteriskCall,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},

			expect: "3e:50:6b:43:bb:32",
		},
		{
			name: "resolved_asterisk_id_distinct_value",

			event: &Event{
				ContainerName: "voip-asterisk-call-docker-2",
				Service:       ServiceAsteriskCall,
				AsteriskID:    "72:ce:24:e6:51:2f",
			},

			expect: "72:ce:24:e6:51:2f",
		},
		{
			name: "unresolved_asterisk_id_degrades_to_empty",

			event: &Event{
				ContainerName: "voip-asterisk-registrar-docker-2",
				Service:       ServiceAsteriskRegistrar,
				AsteriskID:    "",
			},

			expect: "",
		},
		{
			name: "container_name_is_not_the_address",

			event: &Event{
				ContainerName: "voip-asterisk-conference-docker-1",
				Service:       ServiceAsteriskConference,
				AsteriskID:    "aa:bb:cc:dd:ee:ff",
			},

			expect: "aa:bb:cc:dd:ee:ff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := tt.event.EventSubscriptionID(); res != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, res)
			}
		})
	}
}

// Test_EventSubscriptionIDNilReceiver pins the typed-nil guard: the publish path asserts the
// interface on an `any`, and a nil *Event SATISFIES that assertion, so the method must not
// dereference blindly.
func Test_EventSubscriptionIDNilReceiver(t *testing.T) {
	var event *Event

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("EventSubscriptionID panicked on nil receiver: %v", r)
		}
	}()

	if res := event.EventSubscriptionID(); res != "" {
		t.Errorf("Wrong match. expect: empty subscription id, got: %q", res)
	}
}

// Test_EventUnresolvedIDIsPlaceholder pins the degrade path end-to-end: an unresolved id must be
// recognized as the placeholder by eventtopic, not merely be an empty string here.
func Test_EventUnresolvedIDIsPlaceholder(t *testing.T) {
	event := &Event{ContainerName: "voip-asterisk-call-docker-1", Service: ServiceAsteriskCall}

	if !eventtopic.IsPlaceholderSubscriptionID(event.EventSubscriptionID()) {
		t.Errorf("Wrong match. expect: placeholder subscription id, got: not a placeholder")
	}
}

// Test_EventJSONRoundTrip pins the wire shape. bin-call-manager unmarshals this exact payload,
// so a json-tag change here is a cross-service contract change.
func Test_EventJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string

		event *Event

		expectJSON string
	}{
		{
			name: "resolved",

			event: &Event{
				ContainerName: "voip-asterisk-call-docker-2",
				Service:       ServiceAsteriskCall,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},

			expectJSON: `{"container_name":"voip-asterisk-call-docker-2","service":"asterisk-call","asterisk_id":"3e:50:6b:43:bb:32"}`,
		},
		{
			name: "unresolved",

			event: &Event{
				ContainerName: "voip-asterisk-registrar-docker-1",
				Service:       ServiceAsteriskRegistrar,
			},

			expectJSON: `{"container_name":"voip-asterisk-registrar-docker-1","service":"asterisk-registrar","asterisk_id":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshaled, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if string(marshaled) != tt.expectJSON {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectJSON, string(marshaled))
			}

			res := &Event{}
			if errUnmarshal := json.Unmarshal(marshaled, res); errUnmarshal != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", errUnmarshal)
			}

			if *res != *tt.event {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.event, res)
			}
		})
	}
}
