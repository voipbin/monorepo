package kase

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
)

// CaseTagEvent and CaseContactEvent override the subscription address of the global topic exchange
// (VOIP-1404/1405). Both assertions pin the POINTER type: the event data reaches notifyhandler as
// a POINTER and the assertion matches the dynamic type; a VALUE of this pointer-receiver type
// would fail the assertion (the exact pipecat defect this ticket fixed).
var (
	_ eventtopic.SubscriptionIdentifier = (*CaseTagEvent)(nil)
	_ eventtopic.SubscriptionIdentifier = (*CaseContactEvent)(nil)
)

func Test_CaseTagEvent_EventSubscriptionID(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-4001-4001-4001-000000000001")

	tests := []struct {
		name   string
		event  *CaseTagEvent
		expect string
	}{
		{
			"normal",
			&CaseTagEvent{
				CaseID: caseID,
				TagID:  uuid.FromStringOrNil("f1b2c3d4-4001-4001-4001-000000000002"),
			},
			caseID.String(),
		},
		{
			"empty case id",
			&CaseTagEvent{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.event.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// Test_CaseTagEvent_EventSubscriptionIDIsNotTagID pins that the address is the case axis, not
// the other id in the payload. A tag id is shared by every case that carries the tag, so
// addressing by it would fan unrelated cases into one stream.
func Test_CaseTagEvent_EventSubscriptionIDIsNotTagID(t *testing.T) {
	tagID := uuid.FromStringOrNil("f1b2c3d4-4002-4002-4002-000000000009")

	added := &CaseTagEvent{
		CaseID: uuid.FromStringOrNil("f1b2c3d4-4002-4002-4002-000000000001"),
		TagID:  tagID,
	}
	removed := &CaseTagEvent{
		CaseID: uuid.FromStringOrNil("f1b2c3d4-4002-4002-4002-000000000002"),
		TagID:  tagID,
	}

	if added.EventSubscriptionID() == added.TagID.String() {
		t.Errorf("Subscription address must not be the tag id. tag_id: %s", added.TagID)
	}
	if added.EventSubscriptionID() == removed.EventSubscriptionID() {
		t.Errorf("Two different cases must not share one subscription address. address: %s", added.EventSubscriptionID())
	}
}

func Test_CaseContactEvent_EventSubscriptionID(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-4003-4003-4003-000000000001")

	tests := []struct {
		name   string
		event  *CaseContactEvent
		expect string
	}{
		{
			"attach",
			&CaseContactEvent{
				CaseID:    caseID,
				ContactID: uuid.FromStringOrNil("f1b2c3d4-4003-4003-4003-000000000002"),
			},
			caseID.String(),
		},
		{
			// Detach publishes contact_id == uuid.Nil from the same site; the address must not
			// degrade with it.
			"detach",
			&CaseContactEvent{
				CaseID:    caseID,
				ContactID: uuid.Nil,
			},
			caseID.String(),
		},
		{
			"empty case id",
			&CaseContactEvent{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.event.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// Test_CaseEventPayloadJSONKeys pins the payload compatibility contract of VOIP-1405 §3.3: the
// structs replaced map literals, and the JSON key SET must be identical so fanout consumers
// observe no semantic change. Key ORDER is deliberately not asserted (maps marshal key-sorted,
// structs in declaration order).
//
// The absence of a top-level "id" is asserted too: it is the reason these two types MUST carry an
// override -- without one, the default resolution would find nothing and publish every case tag /
// contact event under the `-` placeholder address.
func Test_CaseEventPayloadJSONKeys(t *testing.T) {
	tests := []struct {
		name   string
		data   any
		expect []string
	}{
		{
			"case tag event",
			&CaseTagEvent{
				CaseID: uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000001"),
				TagID:  uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000002"),
			},
			[]string{"case_id", "tag_id"},
		},
		{
			"case contact event",
			&CaseContactEvent{
				CaseID:    uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000003"),
				ContactID: uuid.FromStringOrNil("f1b2c3d4-4004-4004-4004-000000000004"),
			},
			[]string{"case_id", "contact_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatalf("Could not marshal the event data. err: %v", err)
			}

			raw := map[string]any{}
			if errUnmarshal := json.Unmarshal(m, &raw); errUnmarshal != nil {
				t.Fatalf("Could not unmarshal the event data. err: %v", errUnmarshal)
			}

			if len(raw) != len(tt.expect) {
				t.Errorf("Wrong key count. expect: %d, got: %d (%v)", len(tt.expect), len(raw), raw)
			}
			for _, key := range tt.expect {
				if _, ok := raw[key]; !ok {
					t.Errorf("Missing key. key: %s, payload: %v", key, raw)
				}
			}
			if _, ok := raw["id"]; ok {
				t.Errorf("Payload must not carry a top-level id. payload: %v", raw)
			}
		})
	}
}

func Test_CaseEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expect   string
	}{
		{"case_tag_added", EventTypeCaseTagAdded, "case_tag_added"},
		{"case_tag_removed", EventTypeCaseTagRemoved, "case_tag_removed"},
		{"case_contact_attributed", EventTypeCaseContactAttributed, "case_contact_attributed"},
		{"case_contact_detached", EventTypeCaseContactDetached, "case_contact_detached"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, tt.constant)
			}
		})
	}
}

// Test_CaseUsesDefaultSubscriptionID pins the deliberate absence of an override on Case itself:
// its own id IS the address every case-scoped event converges on, so implementing the interface
// would be redundant and the default JSON `id` extraction must keep covering it.
func Test_CaseUsesDefaultSubscriptionID(t *testing.T) {
	var data any = &Case{ID: uuid.FromStringOrNil("f1b2c3d4-4005-4005-4005-000000000001")}

	if _, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		t.Errorf("Case must not implement SubscriptionIdentifier. its own id is the subscription address.")
	}
}
