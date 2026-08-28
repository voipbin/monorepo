// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404/1405).
//
// It covers EVERY event type agent-manager publishes today and asserts the exact key that
// notifyhandler generates for the real event data type of each publish site. The primary defect
// class it guards against is "the right key shape carrying the wrong id space": an event published
// under an id that is not the resource's subscription address produces well-formed keys that no
// instance binding ever matches, and no runtime metric can detect it. Design doc §2.4 / §4.
//
// agent-manager is a default-fallback service: every published payload is an `*agent.Agent`, whose
// own id IS the subscription address, so NO type in this service carries an
// eventtopic.SubscriptionIdentifier override. That absence is asserted explicitly below -- adding
// one later would silently re-address the whole service.
//
// The file lives in models/agent because agent is the only publishing model package of the service
// and the resource all of its events address; it is an external test package so it can import
// sibling packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior, not a specification of what the events ought to
// be. All four constants in models/agent/event.go are LIVE (agent_created / agent_deleted /
// agent_status_updated via PublishWebhookEvent, agent_updated via both PublishEvent and
// PublishWebhookEvent) -- there is no dead constant to exclude here. When a publish path is added
// or removed, update this table in the same change.
package agent_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// agentID is the single subscription address every agent-manager event of one agent must carry.
var agentID = uuid.FromStringOrNil("3a17b5c4-0000-4000-8000-000000000001")

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (1404 design §4.2 / §5.2): the opt-in interface first, then -- ONLY when no override exists --
// the top-level "id" of the marshaled payload. Keeping it here rather than reaching into
// notifyhandler internals is deliberate -- the golden table must fail when a model STARTS or STOPS
// implementing the interface, which is exactly what this two-step reproduction detects.
//
// The early return below is the load-bearing half: an override that EXISTS is authoritative even
// when it yields "" or uuid.Nil, so the JSON fallback must never run behind it. This matches
// notifyhandler.resolveSubscriptionOverride's hasOverride semantics exactly; if the two ever
// diverge, this table stops reproducing what the publish path actually generates.
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
	publisher := string(commonoutline.ServiceNameAgentManager)

	agentData := &agent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Username: "test-agent",
		Status:   agent.StatusAvailable,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// agent resource -- own id is the address, resolved by the default JSON fallback.
		{
			"agent_created",
			agent.EventTypeAgentCreated,
			agentData,
			"agent-manager.agent.3a17b5c4-0000-4000-8000-000000000001.created",
		},
		{
			"agent_updated",
			agent.EventTypeAgentUpdated,
			agentData,
			"agent-manager.agent.3a17b5c4-0000-4000-8000-000000000001.updated",
		},
		{
			"agent_deleted",
			agent.EventTypeAgentDeleted,
			agentData,
			"agent-manager.agent.3a17b5c4-0000-4000-8000-000000000001.deleted",
		},
		// agent_status_updated splits mechanically on the FIRST underscore, so the resource stays
		// `agent` and the action keeps the `status_` prefix. That is intentional -- see
		// eventtopic.RoutingKey -- and it keeps status events inside the same
		// `agent-manager.agent.<agent-id>.#` binding as the lifecycle events above.
		{
			"agent_status_updated",
			agent.EventTypeAgentStatusUpdated,
			agentData,
			"agent-manager.agent.3a17b5c4-0000-4000-8000-000000000001.status_updated",
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

// TestGoldenRoutingKeyStatusUpdatedDecomposition pins the resource/action split of the one event
// type whose name could plausibly be read as a `agent_status` resource. A consumer following one
// agent binds `agent-manager.agent.<agent-id>.#`; if the split ever moved the resource segment to
// `agent_status`, that binding would silently stop delivering status events while every key still
// looked well-formed.
func TestGoldenRoutingKeyStatusUpdatedDecomposition(t *testing.T) {
	publisher := string(commonoutline.ServiceNameAgentManager)
	data := &agent.Agent{Identity: commonidentity.Identity{ID: agentID}}

	key := eventtopic.RoutingKey(publisher, agent.EventTypeAgentStatusUpdated, resolveSubscriptionID(t, data))

	if expect := "agent-manager.agent." + agentID.String() + ".status_updated"; key != expect {
		t.Errorf("Wrong match. expect: %s, got: %s", expect, key)
	}

	// Segment-level decomposition: resource is `agent`, address is the agent id, action carries the
	// whole `status_updated` tail.
	segments := strings.Split(key, ".")
	if len(segments) != 4 {
		t.Fatalf("Wrong match. a routing key has exactly 4 segments. got: %d (%s)", len(segments), key)
	}
	if segments[1] != "agent" {
		t.Errorf("Wrong match. the resource segment must be `agent`, not `agent_status`. got: %s", segments[1])
	}
	if segments[2] != agentID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", agentID.String(), segments[2])
	}
	if segments[3] != "status_updated" {
		t.Errorf("Wrong match. expect: status_updated, got: %s", segments[3])
	}

	// The instance binding a consumer would declare for this agent must actually cover this key --
	// `<publisher>.agent.<agent-id>.#` matches iff the first three segments agree.
	pattern := eventtopic.PatternInstance(publisher, "agent", agentID.String())
	if prefix := strings.TrimSuffix(pattern, "#"); !strings.HasPrefix(key, prefix) {
		t.Errorf("The agent instance binding %s does not cover the key %s.", pattern, key)
	}
}

// TestAgentUsesDefaultSubscriptionID pins the deliberate absence of an override on Agent: its own
// id IS the address, so implementing the interface would be redundant and the default JSON `id`
// extraction must keep covering it. agent-manager has no override type at all (design §2.4).
func TestAgentUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &agent.Agent{Identity: commonidentity.Identity{ID: agentID}}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Agent must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}
}

// TestGoldenRoutingKeysUseOwnID pins the property the table exists to protect: the address every
// agent-manager event resolves to is the agent's OWN id, taken from the marshaled payload's
// top-level `id`. A payload that loses that field (or gains an override returning something else)
// collapses to the `-` placeholder or a foreign address, and this assertion is what catches it.
func TestGoldenRoutingKeysUseOwnID(t *testing.T) {
	data := &agent.Agent{Identity: commonidentity.Identity{ID: agentID, CustomerID: uuid.Must(uuid.NewV4())}}

	res := resolveSubscriptionID(t, data)
	if res != agentID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", agentID.String(), res)
	}

	if eventtopic.IsPlaceholderSubscriptionID(res) {
		t.Errorf("Agent must never resolve to the placeholder address. got: %s", res)
	}
}
