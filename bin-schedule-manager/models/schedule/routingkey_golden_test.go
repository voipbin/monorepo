// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type schedule-manager publishes today, across both resource namespaces
// (schedule / execution), and asserts the exact key that notifyhandler generates for the real
// event data type of each publish site. The primary defect class it guards against is "the right
// key shape carrying the wrong id space": a per-run execution row published under its own id
// produces well-formed keys that no schedule-following binding ever matches, and no runtime
// metric can detect it. Design doc §2.3 / §4.
//
// The file lives in models/schedule because the table spans every publishing model package of the
// service and the schedule is the resource all of them address; it is an external test package so
// it can import the sibling model packages without any import-cycle risk. Note that the event
// type constants for BOTH namespaces live in models/schedule (event.go) while the execution
// payload type lives in models/execution.
//
// MAINTENANCE: this table pins CURRENT behavior. The completion publish site is DYNAMIC --
// pkg/dispatchhandler/dispatch.go:notifyExecutionCompleted picks `execution_failed` by default
// and `execution_succeeded` only when the run's status is StatusSuccess -- so BOTH branches are
// enumerated below. If a third terminal status ever gains its own event type, add its row here in
// the same change.
package schedule_test

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/models/schedule"
)

// scheduleID is the single subscription address every schedule-manager event of one schedule must
// carry, regardless of which resource namespace the event lives in.
var scheduleID = uuid.FromStringOrNil("a1d40e63-0000-4000-8000-000000000001")

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
	publisher := string(commonoutline.ServiceNameScheduleManager)

	scheduleData := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         scheduleID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Type:    schedule.TypeRPC,
		Cron:    "*/5 * * * *",
		Enabled: true,
	}

	// The two execution rows below are DIFFERENT runs of the SAME schedule -- exactly what the
	// dynamic dispatch branch produces over time.
	executionSucceededData := &execution.Execution{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // one row per run: never an address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ScheduleID:  scheduleID,
		TriggerType: execution.TriggerTypeCron,
		Status:      execution.StatusSuccess,
	}

	executionFailedData := &execution.Execution{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // a different row, same schedule address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ScheduleID:  scheduleID,
		TriggerType: execution.TriggerTypeCron,
		Status:      execution.StatusFailed,
		Error:       "rpc timeout",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// schedule resource -- own id is the address, resolved by the default JSON fallback.
		{
			"schedule_created",
			schedule.EventTypeScheduleCreated,
			scheduleData,
			"schedule-manager.schedule.a1d40e63-0000-4000-8000-000000000001.created",
		},
		{
			"schedule_updated",
			schedule.EventTypeScheduleUpdated,
			scheduleData,
			"schedule-manager.schedule.a1d40e63-0000-4000-8000-000000000001.updated",
		},
		{
			"schedule_deleted",
			schedule.EventTypeScheduleDeleted,
			scheduleData,
			"schedule-manager.schedule.a1d40e63-0000-4000-8000-000000000001.deleted",
		},

		// execution resource -- BOTH dynamic branches of notifyExecutionCompleted, addressed by
		// the parent schedule-id rather than the per-run execution id.
		{
			"execution_succeeded",
			schedule.EventTypeExecutionSucceeded,
			executionSucceededData,
			"schedule-manager.execution.a1d40e63-0000-4000-8000-000000000001.succeeded",
		},
		{
			"execution_failed",
			schedule.EventTypeExecutionFailed,
			executionFailedData,
			"schedule-manager.execution.a1d40e63-0000-4000-8000-000000000001.failed",
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
// event of one schedule -- including the rows of DIFFERENT runs -- resolves to the same
// subscription address, so a consumer following that schedule binds
// `schedule-manager.<resource>.<schedule-id>.#` per namespace and receives everything.
func TestGoldenRoutingKeysShareOneAddress(t *testing.T) {
	expect := scheduleID.String()

	tests := []struct {
		name string
		data any
	}{
		{"schedule", &schedule.Schedule{Identity: commonidentity.Identity{ID: scheduleID}}},
		{"execution run 1", &execution.Execution{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, ScheduleID: scheduleID, Status: execution.StatusSuccess}},
		{"execution run 2", &execution.Execution{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, ScheduleID: scheduleID, Status: execution.StatusFailed}},
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

// TestScheduleUsesDefaultSubscriptionID pins the deliberate absence of an override on Schedule:
// its own id IS the address, so implementing the interface would be redundant and the default
// JSON `id` extraction must keep covering it.
func TestScheduleUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &schedule.Schedule{Identity: commonidentity.Identity{ID: scheduleID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Schedule must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}
}
