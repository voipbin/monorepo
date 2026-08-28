package pod

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"monorepo/bin-common-handler/models/eventtopic"
)

var _ eventtopic.SubscriptionIdentifier = (*Event)(nil)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name string

		eventType     string
		expectedValue string
	}{
		{
			name: "pod_added_event_type",

			eventType:     EventTypePodAdded,
			expectedValue: "pod_added",
		},
		{
			name: "pod_updated_event_type",

			eventType:     EventTypePodUpdated,
			expectedValue: "pod_updated",
		},
		{
			name: "pod_deleted_event_type",

			eventType:     EventTypePodDeleted,
			expectedValue: "pod_deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.eventType != tt.expectedValue {
				t.Errorf("Wrong event type. expect: %s, got: %s", tt.expectedValue, tt.eventType)
			}
		})
	}
}

func TestEventTypeUniqueness(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "all_event_types_are_unique",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventTypes := map[string]bool{
				EventTypePodAdded:   true,
				EventTypePodUpdated: true,
				EventTypePodDeleted: true,
			}

			if len(eventTypes) != 3 {
				t.Errorf("Expected 3 unique event types, got: %d", len(eventTypes))
			}
		})
	}
}

// TestEventSubscriptionID pins the placeholder-by-design contract: the subscription address is
// "" for EVERY pod, no matter how identifiable the embedded pod is (uid, name — none of it may
// leak into the routing key).
func TestEventSubscriptionID(t *testing.T) {
	tests := []struct {
		name string

		event *Event

		expect string
	}{
		{
			name: "normal_pod_returns_empty_address",

			event: &Event{
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						UID:       "3f2b0d18-0000-4000-8000-000000000001",
						Name:      "asterisk-call-7c9d5f8b6d-abcde",
						Namespace: "voip",
					},
				},
			},

			expect: "",
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

// TestEventSubscriptionIDNilEmbed pins the nil-embed branch: an Event whose embedded *corev1.Pod
// is nil must still return "" WITHOUT panicking (design §3 wrapper/nil-embed requirement).
func TestEventSubscriptionIDNilEmbed(t *testing.T) {
	event := &Event{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("EventSubscriptionID panicked on nil embed: %v", r)
		}
	}()

	if res := event.EventSubscriptionID(); res != "" {
		t.Errorf("Wrong match. expect: empty subscription id, got: %q", res)
	}
}
