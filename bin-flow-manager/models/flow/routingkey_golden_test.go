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
// flow-manager is a DEFAULT-ID service: BOTH `flow.Flow` and `activeflow.Activeflow` deliberately
// carry NO eventtopic.SubscriptionIdentifier override, so the JSON `id` fallback covers them.
// The activeflow decision is the load-bearing one and is pinned explicitly below
// (TestActiveflowUsesDefaultSubscriptionIDNotReferenceID): `ReferenceID` looks like the natural
// parent axis, but it can be uuid.Nil, which would collapse the key to the `-` placeholder and
// inflate the placeholder rate. Own id wins -- design §2.4.
//
// The file lives in models/flow because that is the service's PRIMARY model package; it is an
// external test package (`flow_test`) so it can import the sibling activeflow package without any
// import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. A new published event type, a changed event-type
// constant, or an override added to *Flow / *Activeflow must be reflected here in the same change
// -- the table is not a specification of what the events ought to be, it is a lock on what they
// are.
package flow_test

import (
	"encoding/json"
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
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Reproducing it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model starts or
// stops implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		// typed-nil guard, mirroring notifyhandler.resolveSubscriptionOverride: a nil pointer whose
		// type implements the interface still SATISFIES the assertion, and every real implementation
		// dereferences its receiver -- calling the method would panic. Production reports "no
		// override" for such a payload, so this guard falls through to the JSON half below rather
		// than returning early; `null` carries no top-level `id` either, so both halves agree on the
		// `-` placeholder.
		if v := reflect.ValueOf(data); v.Kind() != reflect.Ptr || !v.IsNil() {
			return identifier.EventSubscriptionID()
		}
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
	publisher := string(commonoutline.ServiceNameFlowManager)

	flowData := &flow.Flow{
		Identity: commonidentity.Identity{
			ID:         flowID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Type: flow.TypeFlow,
		Name: "test flow",
	}

	// ReferenceID is set to a DIFFERENT uuid on purpose: if an override were ever added that
	// addresses activeflows by their reference, every expectation below would flip and the table
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
		// flow resource -- own id is the address, resolved by the default JSON fallback.
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

// TestFlowUsesDefaultSubscriptionID pins the deliberate ABSENCE of an override on Flow
// (design §2.4): a flow is an independent persistent resource, its own id IS the subscription
// address.
func TestFlowUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &flow.Flow{Identity: commonidentity.Identity{ID: flowID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Flow must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}

	if res := resolveSubscriptionID(t, data); res != flowID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", flowID.String(), res)
	}
}

// TestActiveflowUsesDefaultSubscriptionIDNotReferenceID pins the ONE deliberate design decision in
// this service that could plausibly have gone the other way (design §2.4): Activeflow keeps the
// DEFAULT own-id address and is explicitly NOT addressed by ReferenceID. ReferenceID resembles the
// Category-B parent axis used elsewhere (campaigncall→CampaignID, queuecall→QueueID), but it can
// be uuid.Nil, and normalizeSubscriptionID maps uuid.Nil to the `-` placeholder -- an override
// there would inflate the placeholder rate for every reference-less activeflow. This test fails
// the moment somebody "fixes" that.
func TestActiveflowUsesDefaultSubscriptionIDNotReferenceID(t *testing.T) {
	referenceID := uuid.Must(uuid.NewV4())

	var data any = &activeflow.Activeflow{
		Identity:      commonidentity.Identity{ID: activeflowID},
		ReferenceType: activeflow.ReferenceTypeCall,
		ReferenceID:   referenceID,
	}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Activeflow must not implement SubscriptionIdentifier. its own id is the subscription address, deliberately not the reference_id.")
	}

	res := resolveSubscriptionID(t, data)
	if res != activeflowID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", activeflowID.String(), res)
	}
	if res == referenceID.String() {
		t.Errorf("The activeflow subscription address must not be the reference_id. got: %s", res)
	}
}

// TestActiveflowSubscriptionIDSurvivesNilReferenceID is the empirical half of the decision above:
// with an all-Nil reference the address is still the activeflow's own id, and the resulting key
// carries no `-` placeholder. Under a ReferenceID override this same activeflow would produce
// `flow-manager.activeflow.-.created`, which no instance binding can ever match.
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
