// Golden routing-key table of the global topic exchange `bin-manager.event` (VOIP-1404 /
// VOIP-1405).
//
// It covers EVERY event type call-manager publishes today, across all five resource namespaces
// (call / dtmf / confbridge / groupcall / recording), and asserts the exact key that notifyhandler
// generates for the real event data type of each publish site. The primary defect class it guards
// against is "the right key shape carrying the wrong id space": a per-event random id under a
// resource namespace produces well-formed keys that no instance binding ever matches, and no
// runtime metric can detect it. Design doc §4.2 / §8 (1404), §2.2 / §4 (1405).
//
// The file lives in models/call because the table spans every model package of the service and
// call is the resource most of them address; it is an external test package so it can import the
// sibling model packages without any import-cycle risk.
//
// MAINTENANCE: this table pins CURRENT behavior. When a new event type is published, or an
// existing one changes its data type, add/adjust the row in the same change -- the table is not a
// specification of what the events ought to be, it is a lock on what they are.
package call_test

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/dtmf"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/models/recording"
)

// Fixed addresses, one per resource namespace. call-manager does NOT collapse its namespaces onto
// a single address the way transcribe-manager does: confbridge, groupcall and recording are
// independently addressable resources whose own ids are the subscription address by design
// (1405 §2.4). Only two types (dtmf, outbound-whitelist-rejected) address the call axis instead.
var (
	callID       = uuid.FromStringOrNil("1f01c3d2-0000-4000-8000-000000000001")
	confbridgeID = uuid.FromStringOrNil("1f01c3d2-0000-4000-8000-000000000002")
	groupcallID  = uuid.FromStringOrNil("1f01c3d2-0000-4000-8000-000000000003")
	recordingID  = uuid.FromStringOrNil("1f01c3d2-0000-4000-8000-000000000004")
)

// resolveSubscriptionID mirrors the resolution notifyhandler performs on the publish path
// (VOIP-1419): the eventtopic.SubscriptionIdentifier contract is MANDATORY (own-id types
// satisfy it through the method promoted from the embedded commonidentity.Identity; dtmf and
// outbound-whitelist-rejected through explicit call-id overrides) -- there is no
// JSON fallback anymore -- and a non-implementing or nil payload resolves to "", which the
// routing-key layer collapses to the `-` placeholder. Keeping the mirror here rather than
// reaching into notifyhandler internals is deliberate: the golden table must fail when a model
// stops implementing the interface or its method starts returning the wrong id space.
//
// The parameter stays `any` (not the interface) so the table can also pin what production does
// with a payload that reaches the publish path without a usable implementation (nil interface,
// typed nil): both degrade to the same placeholder instead of panicking.
func resolveSubscriptionID(t *testing.T, data any) string {
	t.Helper()

	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		// typed-nil guard, mirroring notifyhandler: a nil pointer whose type implements the
		// interface still SATISFIES the assertion, and every real implementation dereferences its
		// receiver -- calling the method would panic. Production resolves such a payload to the
		// `-` placeholder, so this guard falls through to the empty return below instead.
		if v := reflect.ValueOf(data); v.Kind() != reflect.Pointer || !v.IsNil() {
			return identifier.EventSubscriptionID()
		}
	}

	return ""
}

func TestGoldenRoutingKeys(t *testing.T) {
	publisher := string(commonoutline.ServiceNameCallManager)

	callData := &call.Call{
		Identity: commonidentity.Identity{
			ID:         callID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		Status: call.StatusProgressing,
	}

	// the DTMF id is regenerated for every single digit event
	// (pkg/callhandler/digit.go digitNotifyDTMFEvent), so it is never an address.
	dtmfData := &dtmf.DTMF{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		CallID: callID,
		Digit:  "1",
	}

	// the whitelist rejection carries no id of its own at all -- see models/call/event.go.
	whitelistRejectedData := &call.OutboundWhitelistRejectedEvent{
		CallID:             callID,
		CustomerID:         uuid.Must(uuid.NewV4()),
		DestinationCountry: "US",
	}

	confbridgeData := &confbridge.Confbridge{
		Identity: commonidentity.Identity{
			ID:         confbridgeID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
	}

	// joined/leaved wrap the confbridge in an ANONYMOUS value embed; the EventSubscriptionID
	// promoted through that embedded Confbridge (and its commonidentity.Identity) returns the
	// embedded confbridge's id -- not the joined/leaved
	// call id.
	confbridgeJoinedData := &confbridge.EventConfbridgeJoined{
		Confbridge:   *confbridgeData,
		JoinedCallID: callID,
	}
	confbridgeLeavedData := &confbridge.EventConfbridgeLeaved{
		Confbridge:   *confbridgeData,
		LeavedCallID: callID,
	}

	groupcallData := &groupcall.Groupcall{
		Identity: commonidentity.Identity{
			ID:         groupcallID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
	}

	recordingData := &recording.Recording{
		Identity: commonidentity.Identity{
			ID:         recordingID,
			CustomerID: uuid.Must(uuid.NewV4()),
		},
		ReferenceType: recording.ReferenceTypeCall,
		ReferenceID:   callID,
	}

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		// call resource -- own id is the address, returned through the EventSubscriptionID
		// promoted from the embedded commonidentity.Identity (VOIP-1419).
		// pkg/callhandler/db.go, bridge.go, chained_call.go.
		{
			"call_created",
			call.EventTypeCallCreated,
			callData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.created",
		},
		{
			"call_updated",
			call.EventTypeCallUpdated,
			callData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.updated",
		},
		{
			"call_deleted",
			call.EventTypeCallDeleted,
			callData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.deleted",
		},
		{
			"call_hangup",
			call.EventTypeCallHangup,
			callData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.hangup",
		},

		// the 5 dynamic branches of the mapEvt status->event table in
		// pkg/callhandler/db.go dbUpdateStatus. All 5 publish the same *call.Call data, but the
		// event type -- and therefore the ACTION segment -- is chosen at runtime, so every branch
		// needs its own row. StatusHangup is deliberately absent from that map (hangup is
		// published by Hangup(), covered by the call_hangup row above).
		{
			"call_dialing (mapEvt)",
			call.EventTypeCallDialing,
			callData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.dialing",
		},
		{
			"call_ringing (mapEvt)",
			call.EventTypeCallRinging,
			callData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.ringing",
		},
		{
			"call_progressing (mapEvt)",
			call.EventTypeCallProgressing,
			callData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.progressing",
		},
		{
			"call_terminating (mapEvt)",
			call.EventTypeCallTerminating,
			callData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.terminating",
		},
		{
			"call_canceling (mapEvt)",
			call.EventTypeCallCanceling,
			callData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.canceling",
		},

		// SPECIAL CASE: the only DOT-separated event type in the monorepo
		// (`call.outbound_whitelist_rejected`). RoutingKey lowercases the type and rewrites every
		// `.` to `_` BEFORE splitting on the first `_`, so it normalizes to resource `call` +
		// action `outbound_whitelist_rejected` and lands in the same namespace as the lifecycle
		// events above -- it does NOT leak an extra key segment. The address comes from the
		// type's explicit method, which returns the parent call id (the payload has no id of
		// its own).
		{
			"call.outbound_whitelist_rejected (dot type normalization)",
			call.EventTypeCallOutboundWhitelistRejected,
			whitelistRejectedData,
			"call-manager.call.1f01c3d2-0000-4000-8000-000000000001.outbound_whitelist_rejected",
		},

		// dtmf resource -- addressed by the parent call-id via its explicit method, never by the
		// per-digit random own id.
		{
			"dtmf_received",
			dtmf.EventTypeDTMFReceived,
			dtmfData,
			"call-manager.dtmf.1f01c3d2-0000-4000-8000-000000000001.received",
		},

		// confbridge resource -- own id is the address. pkg/confbridgehandler/db.go.
		{
			"confbridge_created",
			confbridge.EventTypeConfbridgeCreated,
			confbridgeData,
			"call-manager.confbridge.1f01c3d2-0000-4000-8000-000000000002.created",
		},
		{
			"confbridge_deleted",
			confbridge.EventTypeConfbridgeDeleted,
			confbridgeData,
			"call-manager.confbridge.1f01c3d2-0000-4000-8000-000000000002.deleted",
		},
		{
			"confbridge_terminating",
			confbridge.EventTypeConfbridgeTerminating,
			confbridgeData,
			"call-manager.confbridge.1f01c3d2-0000-4000-8000-000000000002.terminating",
		},
		{
			"confbridge_terminated",
			confbridge.EventTypeConfbridgeTerminated,
			confbridgeData,
			"call-manager.confbridge.1f01c3d2-0000-4000-8000-000000000002.terminated",
		},
		{
			"confbridge_joined (anonymous embed id promotion)",
			confbridge.EventTypeConfbridgeJoined,
			confbridgeJoinedData,
			"call-manager.confbridge.1f01c3d2-0000-4000-8000-000000000002.joined",
		},
		{
			"confbridge_leaved (anonymous embed id promotion)",
			confbridge.EventTypeConfbridgeLeaved,
			confbridgeLeavedData,
			"call-manager.confbridge.1f01c3d2-0000-4000-8000-000000000002.leaved",
		},

		// groupcall resource -- own id is the address. pkg/groupcallhandler/db.go, hangup.go.
		{
			"groupcall_created",
			groupcall.EventTypeGroupcallCreated,
			groupcallData,
			"call-manager.groupcall.1f01c3d2-0000-4000-8000-000000000003.created",
		},
		{
			"groupcall_progressing",
			groupcall.EventTypeGroupcallProgressing,
			groupcallData,
			"call-manager.groupcall.1f01c3d2-0000-4000-8000-000000000003.progressing",
		},
		{
			"groupcall_hangup",
			groupcall.EventTypeGroupcallHangup,
			groupcallData,
			"call-manager.groupcall.1f01c3d2-0000-4000-8000-000000000003.hangup",
		},
		{
			"groupcall_deleted",
			groupcall.EventTypeGroupcallDeleted,
			groupcallData,
			"call-manager.groupcall.1f01c3d2-0000-4000-8000-000000000003.deleted",
		},

		// recording resource -- own id is the address, deliberately (1405 §2.4): the download and
		// stop APIs are recording-id keyed and the id returns from the start RPC, so it is
		// pre-bindable. Call-axis followers use the type-level pattern instead.
		// pkg/recordinghandler/start.go, stop.go.
		{
			"recording_started",
			recording.EventTypeRecordingStarted,
			recordingData,
			"call-manager.recording.1f01c3d2-0000-4000-8000-000000000004.started",
		},
		{
			"recording_finished",
			recording.EventTypeRecordingFinished,
			recordingData,
			"call-manager.recording.1f01c3d2-0000-4000-8000-000000000004.finished",
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

// TestGoldenRoutingKeysCallAxisShareOneAddress pins the property the two call-axis methods
// (dtmf, outbound-whitelist-rejected) exist to protect: every event that belongs to ONE call
// resolves to the same subscription address, so a
// consumer following that call binds `call-manager.call.<call-id>.#` plus
// `call-manager.dtmf.<call-id>.#` and receives everything, regardless of the per-event ids the
// payloads happen to carry.
func TestGoldenRoutingKeysCallAxisShareOneAddress(t *testing.T) {
	expect := callID.String()

	tests := []struct {
		name string
		data any
	}{
		{"call", &call.Call{Identity: commonidentity.Identity{ID: callID}}},
		{"dtmf", &dtmf.DTMF{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, CallID: callID}},
		{"outbound_whitelist_rejected", &call.OutboundWhitelistRejectedEvent{CallID: callID, CustomerID: uuid.Must(uuid.NewV4())}},
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

// TestGoldenRoutingKeysConfbridgeEventWrappersPromoteConfbridgeID pins the id space the
// joined/leaved rows above depend on. Both wrappers carry a SECOND uuid (the joined/leaved call
// id) that is a plausible-looking but wrong address; the method promoted through the embedded
// Confbridge must keep resolving
// the embedded confbridge `id` instead.
func TestGoldenRoutingKeysConfbridgeEventWrappersPromoteConfbridgeID(t *testing.T) {
	base := confbridge.Confbridge{
		Identity: commonidentity.Identity{ID: confbridgeID, CustomerID: uuid.Must(uuid.NewV4())},
	}

	joined := &confbridge.EventConfbridgeJoined{Confbridge: base, JoinedCallID: callID}
	leaved := &confbridge.EventConfbridgeLeaved{Confbridge: base, LeavedCallID: callID}

	for _, tt := range []struct {
		name string
		data any
	}{
		{"joined", joined},
		{"leaved", leaved},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := resolveSubscriptionID(t, tt.data)
			if res != confbridgeID.String() {
				t.Errorf("Wrong match. expect: %s, got: %s", confbridgeID, res)
			}
			if res == callID.String() {
				t.Errorf("Subscription address must not be the joined/leaved call id. call_id: %s", callID)
			}
		})
	}
}

// TestGoldenRoutingKeysPlaceholderFallbacks pins what happens when an address is absent. The
// EventSubscriptionID method is the ONLY source of the address -- a Nil return is authoritative, nothing
// resurrects a different id behind it -- and every case collapses to the `-` placeholder, which
// type-level bindings still match and instance bindings correctly never do.
func TestGoldenRoutingKeysPlaceholderFallbacks(t *testing.T) {
	publisher := string(commonoutline.ServiceNameCallManager)

	tests := []struct {
		name      string
		eventType string
		data      any
		expect    string
	}{
		{
			// own id present, but the method says the call-id is the address and it is Nil:
			// the own id must never be resurrected behind the method's back.
			"dtmf without call id",
			dtmf.EventTypeDTMFReceived,
			&dtmf.DTMF{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}},
			"call-manager.dtmf.-.received",
		},
		{
			"outbound_whitelist_rejected without call id",
			call.EventTypeCallOutboundWhitelistRejected,
			&call.OutboundWhitelistRejectedEvent{CustomerID: uuid.Must(uuid.NewV4())},
			"call-manager.call.-.outbound_whitelist_rejected",
		},
		{
			"call without id",
			call.EventTypeCallCreated,
			&call.Call{},
			"call-manager.call.-.created",
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
