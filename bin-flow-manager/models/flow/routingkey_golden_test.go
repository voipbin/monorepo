// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type flow-manager publishes today, across both resource namespaces:
//   - flow      -- flow_created/updated/deleted (pkg/flowhandler/db.go:174,234,272,307; :307 is a
//     second flow_updated site, same type and same data type as :234)
//   - activeflow -- activeflow_created/updated/deleted (pkg/activeflowhandler/db.go:124,238,262,387;
//     :238 and :262 are two activeflow_updated sites)
//
// The defect class it guards against is "the right key shape carrying the wrong id space": a key
// whose third segment is not the address subscribers can bind to in advance produces well-formed
// keys that no instance binding ever matches, and no runtime metric can detect it. Design doc
// §2.4 / §4.
//
// flow-manager is an OWN-ID service: BOTH `flow.Flow` and `activeflow.Activeflow` implement
// eventtopic.SubscriptionIdentifier explicitly (mandatory since VOIP-1419; an empty return
// degrades to the `-` placeholder) and return the resource's own id. The activeflow decision is
// the load-bearing one (pinned in models/activeflow's own tests and by the table below):
// `ReferenceID` looks like the natural parent axis, but it can be uuid.Nil, which would collapse
// the key to the `-` placeholder and inflate the placeholder rate. Own id wins -- design §2.4.
//
// The file lives in models/flow because that is the service's PRIMARY model package; it is an
// external test package (`flow_test`) so it can import the sibling activeflow package without any
// import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. A new published event type, a changed event-type
// constant, or a changed EventSubscriptionID implementation on *Flow / *Activeflow must be
// reflected here in the same change -- the table is not a specification of what the events ought
// to be, it is a lock on what they are.
package flow_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
)

var (
	// flowID is the subscription address of every flow_* event.
	flowID = uuid.FromStringOrNil("3c7e5a10-0000-4000-8000-000000000001")

	// activeflowID is the subscription address of every activeflow_* event -- the activeflow's
	// OWN id, deliberately not its ReferenceID (see the package comment).
	activeflowID = uuid.FromStringOrNil("3c7e5a10-0000-4000-8000-000000000002")
)

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): every published type carries an explicit, mandatory EventSubscriptionID method;
// the method's return is the address, and an empty return (or a non-implementing / nil payload)
// degrades to the `-` placeholder. Reproducing it here rather than reaching into notifyhandler
// internals is deliberate -- the golden table must fail when a model's method starts returning a
// different id space.
//
// The parameter stays `any` on purpose: a non-implementing payload resolves to "" (placeholder)
// rather than failing to compile, matching the production helper's degrade path.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
		// interface still SATISFIES the assertion, and every real implementation dereferences its
		// receiver -- calling the method would panic. Production resolves such a payload to the
		// `-` placeholder instead.
		if v := reflect.ValueOf(data); v.Kind() != reflect.Ptr || !v.IsNil() {
			return identifier.EventSubscriptionID()
		}
	}

	return ""
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameFlowManager)

	flowData := &flow.Flow{
		Identity: commonidentity.Identity{
			ID:         flowID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Type: flow.TypeFlow,
		Name: "test flow",
	}

	// ReferenceID is set to a DIFFERENT uuid on purpose: if EventSubscriptionID were ever changed
	// to address activeflows by their reference, every expectation below would flip and the table
	// would fail loudly instead of drifting silently.
	activeflowData := &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         activeflowID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		FlowID:        flowID,
		Status:        activeflow.StatusRunning,
		ReferenceType: activeflow.ReferenceTypeCall,
		ReferenceID:   uuid.Must(uuid.NewV4()),
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// flow resource -- own id is the address, returned by the explicit EventSubscriptionID.
		{
			"flow_created",
			flow.EventTypeFlowCreated,
			flowData,
			"flow-manager.flow.3c7e5a10-0000-4000-8000-000000000001.created",
		},
		{
			"flow_updated",
			flow.EventTypeFlowUpdated,
			flowData,
			"flow-manager.flow.3c7e5a10-0000-4000-8000-000000000001.updated",
		},
		{
			"flow_deleted",
			flow.EventTypeFlowDeleted,
			flowData,
			"flow-manager.flow.3c7e5a10-0000-4000-8000-000000000001.deleted",
		},

		// activeflow resource -- own id is the address (NOT ReferenceID), design §2.4.
		{
			"activeflow_created",
			activeflow.EventTypeActiveflowCreated,
			activeflowData,
			"flow-manager.activeflow.3c7e5a10-0000-4000-8000-000000000002.created",
		},
		{
			"activeflow_updated",
			activeflow.EventTypeActiveflowUpdated,
			activeflowData,
			"flow-manager.activeflow.3c7e5a10-0000-4000-8000-000000000002.updated",
		},
		{
			"activeflow_deleted",
			activeflow.EventTypeActiveflowDeleted,
			activeflowData,
			"flow-manager.activeflow.3c7e5a10-0000-4000-8000-000000000002.deleted",
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

// TestActiveflowSubscriptionIDSurvivesNilReferenceID pins the empirical half of the own-id
// decision (design §2.4): with an all-Nil reference the address is still the activeflow's own id,
// and the resulting key carries no `-` placeholder. An EventSubscriptionID returning ReferenceID
// would make this same activeflow produce `flow-manager.activeflow.-.created`, which no instance
// binding can ever match. The per-type address-choice tests (own id, not ReferenceID/FlowID) live
// in the models' own packages next to the methods.
func TestActiveflowSubscriptionIDSurvivesNilReferenceID(t *testing.T) {
	var data any = &activeflow.Activeflow{
		Identity:    commonidentity.Identity{ID: activeflowID},
		ReferenceID: uuid.Nil,
	}

	subscriptionID := resolveSubscriptionID(t, data)
	if eventtopic.IsPlaceholderSubscriptionID(subscriptionID) {
		t.Errorf("The activeflow subscription address must not collapse to the placeholder. got: %s", subscriptionID)
	}

	expect := "flow-manager.activeflow.3c7e5a10-0000-4000-8000-000000000002.created"
	res := eventtopic.RoutingKey(string(commonoutline.ServiceNameFlowManager), activeflow.EventTypeActiveflowCreated, subscriptionID)
	if res != expect {
		t.Errorf("Wrong match. expect: %s, got: %s", expect, res)
	}
}
