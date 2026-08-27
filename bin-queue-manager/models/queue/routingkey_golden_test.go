// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type queue-manager publishes today, across both resource namespaces
// (queue / queuecall), and asserts the exact key that notifyhandler generates for the real event
// data type of each publish site. The primary defect class it guards against is "the right key
// shape carrying the wrong id space": a queuecall published under its own id produces well-formed
// keys that no queue-following binding ever matches, and no runtime metric can detect it. Design
// doc §2.3 / §4.
//
// The file lives in models/queue because the table spans every publishing model package of the
// service and the queue is the resource all of them address; it is an external test package so it
// can import the sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. The `queue_created` constant in models/queue is
// DEAD (no publish site anywhere in the service -- QueueCreate does not notify) and is
// deliberately excluded -- design §4 dead-constant list. If a `queue_created` publish is ever
// added, add its row here in the same change.
package queue_test

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
)

// queueID is the single subscription address every queue-manager event of one queue must carry,
// regardless of which resource namespace the event lives in.
var queueID = uuid.FromStringOrNil("5e83b6f4-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Keeping it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model stops
// implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly; if the two ever
// diverge, this table stops reproducing what the publish path actually generates.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		return identifier.EventSubscriptionID()
	}

	m, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Could not marshal the event data. err: %v", err)
	}

	d := struct {
		ID string `json:"id"`
	}{}
	if errUnmarshal := json.Unmarshal(m, &d); errUnmarshal != nil {
		return ""
	}

	return d.ID
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameQueueManager)

	queueData := &queue.Queue{
		Identity: commonidentity.Identity{
			ID:         queueID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		RoutingMethod: queue.RoutingMethodRandom,
	}

	queuecallData := &queuecall.Queuecall{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // queuecall-id: stable, but the wrong address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		QueueID: queueID,
		Status:  queuecall.StatusWaiting,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// queue resource -- own id is the address, resolved by the default JSON fallback.
		{
			"queue_updated",
			queue.EventTypeQueueUpdated,
			queueData,
			"queue-manager.queue.5e83b6f4-0000-4000-8000-000000000001.updated",
		},
		{
			"queue_deleted",
			queue.EventTypeQueueDeleted,
			queueData,
			"queue-manager.queue.5e83b6f4-0000-4000-8000-000000000001.deleted",
		},

		// queuecall resource -- addressed by the parent queue-id, not the queuecall-id. All seven
		// lifecycle events land on the queue address.
		{
			"queuecall_created",
			queuecall.EventTypeQueuecallCreated,
			queuecallData,
			"queue-manager.queuecall.5e83b6f4-0000-4000-8000-000000000001.created",
		},
		{
			"queuecall_deleted",
			queuecall.EventTypeQueuecallDeleted,
			queuecallData,
			"queue-manager.queuecall.5e83b6f4-0000-4000-8000-000000000001.deleted",
		},
		{
			"queuecall_connecting",
			queuecall.EventTypeQueuecallConnecting,
			queuecallData,
			"queue-manager.queuecall.5e83b6f4-0000-4000-8000-000000000001.connecting",
		},
		{
			"queuecall_serviced",
			queuecall.EventTypeQueuecallServiced,
			queuecallData,
			"queue-manager.queuecall.5e83b6f4-0000-4000-8000-000000000001.serviced",
		},
		{
			"queuecall_abandoned",
			queuecall.EventTypeQueuecallAbandoned,
			queuecallData,
			"queue-manager.queuecall.5e83b6f4-0000-4000-8000-000000000001.abandoned",
		},
		{
			"queuecall_done",
			queuecall.EventTypeQueuecallDone,
			queuecallData,
			"queue-manager.queuecall.5e83b6f4-0000-4000-8000-000000000001.done",
		},
		{
			"queuecall_waiting",
			queuecall.EventTypeQueuecallWaiting,
			queuecallData,
			"queue-manager.queuecall.5e83b6f4-0000-4000-8000-000000000001.waiting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscriptionID := resolveSubscriptionID(t, tt.data)

			res := eventtopic.RoutingKey(publisher, tt.eventType, subscriptionID)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestGoldenRoutingKeysShareOneAddress pins the property the table above exists to protect: every
// event of one queue -- including the events of DIFFERENT queuecalls -- resolves to the same
// subscription address, so a consumer following that queue binds
// `queue-manager.<resource>.<queue-id>.#` per namespace and receives everything.
func TestGoldenRoutingKeysShareOneAddress(t *testing.T) {
	expect := queueID.String()

	tests := []struct {
		name string
		data any
	}{
		{"queue", &queue.Queue{Identity: commonidentity.Identity{ID: queueID}}},
		{"queuecall a", &queuecall.Queuecall{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, QueueID: queueID}},
		{"queuecall b", &queuecall.Queuecall{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, QueueID: queueID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resolveSubscriptionID(t, tt.data)
			if res != expect {
				t.Errorf("Wrong match. expect: %s, got: %s", expect, res)
			}
		})
	}
}

// TestQueueUsesDefaultSubscriptionID pins the deliberate absence of an override on Queue: its own
// id IS the address, so implementing the interface would be redundant and the default JSON `id`
// extraction must keep covering it.
func TestQueueUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &queue.Queue{Identity: commonidentity.Identity{ID: queueID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Queue must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}
}
