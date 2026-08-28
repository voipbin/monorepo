// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404 /
// VOIP-1405).
//
// It covers EVERY event type ai-manager publishes today, across all five resource namespaces
// (ai / aicall / aimessage / summary / team), and asserts the exact key that notifyhandler
// generates for the real event data type of each publish site. The primary defect class it guards
// against is "the right key shape carrying the wrong id space": a per-event id under a resource
// namespace produces well-formed keys that no instance binding ever matches, and no runtime
// metric can detect it. Design doc 1404 §4.2 / 1405 §2.2-§2.4.
//
// The file lives in models/ai because the table spans every model package of the service and `ai`
// is the service's primary resource; it is an external test package so it can import the sibling
// model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. When a publish site gains, loses, or renames an
// event type, the corresponding row must be updated in the same change -- the table is not a
// specification of what the events ought to be, it is a lock on what they are.
//
// NOTE on the `team` resource: ai-manager's team_* events are the AI-team (models/team) lifecycle.
// bin-pipecat-manager publishes an unrelated `team_member_switched` under its own publisher
// segment, so the two never collide -- the first key segment is the publisher.
package ai_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// aicallID is the single subscription address every message event of one AIcall must carry,
// regardless of whether the payload is the persisted message or a streaming fragment.
var aicallID = uuid.FromStringOrNil("7c31b0a4-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): the subscription address comes ONLY from the payload's explicit
// EventSubscriptionID method -- implementation is mandatory (enforced at compile time on the
// publish call sites), and an empty return degrades to the `-` placeholder. Keeping this
// mirror here rather than reaching into notifyhandler internals is deliberate -- the golden
// table must keep compiling and FAIL when a model stops implementing the interface, which is
// exactly what the `data any` parameter and the "" return for a non-implementing payload
// detect.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	identifier, ok := data.(eventtopic.SubscriptionIdentifier)
	if !ok {
		return ""
	}

	// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
	// interface still SATISFIES the assertion, and every real implementation dereferences its
	// receiver -- calling the method would panic. Production resolves such a payload to the
	// `-` placeholder, so this returns "".
	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return identifier.EventSubscriptionID()
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameAIManager)

	aiID := uuid.FromStringOrNil("7c31b0a4-0000-4000-8000-0000000000a1")
	aiData := &ai.AI{
		Identity: commonidentity.Identity{
			ID:         aiID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Name: "test ai",
	}

	aicallData := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         aicallID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   aiID,
		Status:         aicall.StatusProgressing,
	}

	messageData := &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // persisted, stable -- but the wrong address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		AIcallID:  aicallID,
		Direction: message.DirectionOutgoing,
		Role:      message.RoleUser,
		Content:   "hello",
	}

	intermediateData := &message.IntermediateWebhookMessage{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()), // per-delta id: never an address
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		AIcallID:  aicallID,
		Role:      message.RoleAssistant,
		Content:   "hel",
		Direction: message.DirectionIncoming,
		Sequence:  1,
	}

	summaryID := uuid.FromStringOrNil("7c31b0a4-0000-4000-8000-0000000000c1")
	summaryData := &summary.Summary{
		Identity: commonidentity.Identity{
			ID:         summaryID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ReferenceType: summary.ReferenceTypeCall,
		ReferenceID:   aicallID,
	}

	teamID := uuid.FromStringOrNil("7c31b0a4-0000-4000-8000-0000000000d1")
	teamData := &team.Team{
		Identity: commonidentity.Identity{
			ID:         teamID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Name: "test team",
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// ai resource -- own id is the address (explicit EventSubscriptionID, VOIP-1419).
		{
			"ai_created",
			ai.EventTypeCreated,
			aiData,
			"ai-manager.ai.7c31b0a4-0000-4000-8000-0000000000a1.created",
		},
		{
			"ai_updated",
			ai.EventTypeUpdated,
			aiData,
			"ai-manager.ai.7c31b0a4-0000-4000-8000-0000000000a1.updated",
		},
		{
			"ai_deleted",
			ai.EventTypeDeleted,
			aiData,
			"ai-manager.ai.7c31b0a4-0000-4000-8000-0000000000a1.deleted",
		},

		// aicall resource -- own id is the address. The event types carry two underscores, so the
		// mechanical split on the FIRST `_` yields resource `aicall` and action `status_<state>`.
		{
			"aicall_status_initializing",
			aicall.EventTypeStatusInitializing,
			aicallData,
			"ai-manager.aicall.7c31b0a4-0000-4000-8000-000000000001.status_initializing",
		},
		{
			"aicall_status_progressing",
			aicall.EventTypeStatusProgressing,
			aicallData,
			"ai-manager.aicall.7c31b0a4-0000-4000-8000-000000000001.status_progressing",
		},
		{
			"aicall_status_pausing",
			aicall.EventTypeStatusPausing,
			aicallData,
			"ai-manager.aicall.7c31b0a4-0000-4000-8000-000000000001.status_pausing",
		},
		{
			"aicall_status_resuming",
			aicall.EventTypeStatusResuming,
			aicallData,
			"ai-manager.aicall.7c31b0a4-0000-4000-8000-000000000001.status_resuming",
		},
		{
			"aicall_status_terminating",
			aicall.EventTypeStatusTerminating,
			aicallData,
			"ai-manager.aicall.7c31b0a4-0000-4000-8000-000000000001.status_terminating",
		},
		{
			"aicall_status_terminated",
			aicall.EventTypeStatusTerminated,
			aicallData,
			"ai-manager.aicall.7c31b0a4-0000-4000-8000-000000000001.status_terminated",
		},

		// aimessage resource -- addressed by the parent AIcallID, not the message's own id, for
		// BOTH the persisted message (Category B) and the streaming fragment (Category A).
		{
			"aimessage_created",
			message.EventTypeMessageCreated,
			messageData,
			"ai-manager.aimessage.7c31b0a4-0000-4000-8000-000000000001.created",
		},
		{
			"aimessage_intermediate",
			message.EventTypeMessageIntermediate,
			intermediateData,
			"ai-manager.aimessage.7c31b0a4-0000-4000-8000-000000000001.intermediate",
		},

		// summary resource -- own id is the address (explicit EventSubscriptionID, VOIP-1419).
		{
			"summary_created",
			summary.EventTypeCreated,
			summaryData,
			"ai-manager.summary.7c31b0a4-0000-4000-8000-0000000000c1.created",
		},
		{
			"summary_updated",
			summary.EventTypeUpdated,
			summaryData,
			"ai-manager.summary.7c31b0a4-0000-4000-8000-0000000000c1.updated",
		},
		{
			"summary_deleted",
			summary.EventTypeDeleted,
			summaryData,
			"ai-manager.summary.7c31b0a4-0000-4000-8000-0000000000c1.deleted",
		},

		// team resource -- own id is the address (explicit EventSubscriptionID, VOIP-1419).
		{
			"team_created",
			team.EventTypeCreated,
			teamData,
			"ai-manager.team.7c31b0a4-0000-4000-8000-0000000000d1.created",
		},
		{
			"team_updated",
			team.EventTypeUpdated,
			teamData,
			"ai-manager.team.7c31b0a4-0000-4000-8000-0000000000d1.updated",
		},
		{
			"team_deleted",
			team.EventTypeDeleted,
			teamData,
			"ai-manager.team.7c31b0a4-0000-4000-8000-0000000000d1.deleted",
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

// TestGoldenRoutingKeysMessagesShareTheAIcallAddress pins the property the aimessage rows exist to
// protect: every message event of one AIcall -- persisted or intermediate -- resolves to the same
// subscription address, so a consumer following that conversation binds
// `ai-manager.aimessage.<aicall-id>.#` and receives everything. Together with
// `ai-manager.aicall.<aicall-id>.#` that is the full two-pattern view of one AIcall.
func TestGoldenRoutingKeysMessagesShareTheAIcallAddress(t *testing.T) {
	expect := aicallID.String()

	tests := []struct {
		name string
		data any
	}{
		{
			"aicall",
			&aicall.AIcall{Identity: commonidentity.Identity{ID: aicallID}},
		},
		{
			"message",
			&message.Message{
				Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				AIcallID: aicallID,
			},
		},
		{
			"intermediate message",
			&message.IntermediateWebhookMessage{
				Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				AIcallID: aicallID,
			},
		},
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
