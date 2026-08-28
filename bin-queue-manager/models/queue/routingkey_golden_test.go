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
	"reflect"
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
// (VOIP-1419): every published event data type implements eventtopic.SubscriptionIdentifier
// explicitly -- the method's return value IS the subscription-id segment, and an empty return
// degrades to the `-` placeholder. No JSON fallback exists anymore. Keeping the mirror here
// rather than reaching into notifyhandler internals is deliberate -- the golden table must fail
// when a model's method stops returning the pinned address.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		// A non-implementing payload cannot reach the narrowed PublishEvent signature at all;
		// the helper keeps the `any` parameter for uniform table rows and resolves such data to
		// the placeholder-producing empty id.
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
	// interface still SATISFIES the assertion, and every real implementation dereferences its
	// receiver -- calling the method would panic. Production resolves such a payload to the `-`
	// placeholder.
	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
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
		// queue resource -- own id is the address, returned by its explicit EventSubscriptionID.
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
