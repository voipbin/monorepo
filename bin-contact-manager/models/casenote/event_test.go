package casenote

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
)

// CaseNote and CaseNoteDeletedEvent override the subscription address of the global topic
// exchange (VOIP-1404/1405). Both assertions pin the POINTER receiver: notifyhandler asserts on
// the dynamic type of the event data, which is always a pointer, so a value receiver would
// silently never be picked up.
var (
	_ eventtopic.SubscriptionIdentifier = (*CaseNote)(nil)
	_ eventtopic.SubscriptionIdentifier = (*CaseNoteDeletedEvent)(nil)
)

func Test_CaseNote_EventSubscriptionID(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-5001-5001-5001-000000000001")
	authorID := uuid.FromStringOrNil("f1b2c3d4-5001-5001-5001-000000000002")

	tests := []struct {
		name   string
		note   *CaseNote
		expect string
	}{
		{
			"normal",
			&CaseNote{
				ID:         uuid.FromStringOrNil("f1b2c3d4-5001-5001-5001-000000000003"),
				CustomerID: uuid.FromStringOrNil("f1b2c3d4-5001-5001-5001-000000000004"),
				CaseID:     caseID,
				AuthorType: AuthorTypeAgent,
				AuthorID:   &authorID,
				Text:       "customer confirmed the outage is resolved",
			},
			caseID.String(),
		},
		{
			"empty case id",
			&CaseNote{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.note.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// Test_CaseNote_EventSubscriptionIDIsNotOwnID is the mandatory "address != own id" assertion
// (design §2.3, pilot precedent TestSpeechEventSubscriptionIDIsNotOwnID). CaseNote HAS a stable
// top-level `id`, so an omitted override would resolve to that note id silently -- a well-formed
// key nobody can bind to. Two notes on one case must share one address.
func Test_CaseNote_EventSubscriptionIDIsNotOwnID(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-5002-5002-5002-000000000001")

	first := &CaseNote{
		ID:     uuid.FromStringOrNil("f1b2c3d4-5002-5002-5002-000000000002"),
		CaseID: caseID,
	}
	second := &CaseNote{
		ID:     uuid.FromStringOrNil("f1b2c3d4-5002-5002-5002-000000000003"),
		CaseID: caseID,
	}

	if first.ID == second.ID {
		t.Fatalf("Case note ids are expected to differ per note. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != caseID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", caseID, first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the case note own id. id: %s", first.ID)
	}
}

func Test_CaseNoteDeletedEvent_EventSubscriptionID(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-5003-5003-5003-000000000001")

	tests := []struct {
		name   string
		event  *CaseNoteDeletedEvent
		expect string
	}{
		{
			"normal",
			&CaseNoteDeletedEvent{
				ID:         uuid.FromStringOrNil("f1b2c3d4-5003-5003-5003-000000000002"),
				CaseID:     caseID,
				CustomerID: uuid.FromStringOrNil("f1b2c3d4-5003-5003-5003-000000000003"),
			},
			caseID.String(),
		},
		{
			"empty case id",
			&CaseNoteDeletedEvent{},
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

// Test_CaseNoteDeletedEvent_EventSubscriptionIDIsNotOwnID guards the silent-failure class called
// out in design §2.3: this payload carries a top-level `id` (the note id), so a missing override
// would produce a well-formed-but-wrong address that the placeholder metric cannot detect. The
// address must be the case id, and it must differ from the note id.
func Test_CaseNoteDeletedEvent_EventSubscriptionIDIsNotOwnID(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-5004-5004-5004-000000000001")
	noteID := uuid.FromStringOrNil("f1b2c3d4-5004-5004-5004-000000000002")

	e := &CaseNoteDeletedEvent{
		ID:         noteID,
		CaseID:     caseID,
		CustomerID: uuid.FromStringOrNil("f1b2c3d4-5004-5004-5004-000000000003"),
	}

	if e.EventSubscriptionID() != caseID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", caseID, e.EventSubscriptionID())
	}
	if e.EventSubscriptionID() == noteID.String() {
		t.Errorf("Subscription address must not be the case note own id. id: %s", noteID)
	}

	// The own id really is present in the payload -- that is what makes the missing-override case
	// silent rather than a placeholder. Pin it so the risk stays visible.
	m, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Could not marshal the event data. err: %v", err)
	}
	raw := map[string]any{}
	if errUnmarshal := json.Unmarshal(m, &raw); errUnmarshal != nil {
		t.Fatalf("Could not unmarshal the event data. err: %v", errUnmarshal)
	}
	if raw["id"] != noteID.String() {
		t.Errorf("Wrong match. expect: %s, got: %v", noteID, raw["id"])
	}
}

// Test_CaseNoteDeletedEventPayloadJSONKeys pins the payload compatibility contract of VOIP-1405
// §3.3: the struct replaced a map literal and the JSON key SET must be identical. Key ORDER is
// deliberately not asserted (maps marshal key-sorted, structs in declaration order).
func Test_CaseNoteDeletedEventPayloadJSONKeys(t *testing.T) {
	e := &CaseNoteDeletedEvent{
		ID:         uuid.FromStringOrNil("f1b2c3d4-5005-5005-5005-000000000001"),
		CaseID:     uuid.FromStringOrNil("f1b2c3d4-5005-5005-5005-000000000002"),
		CustomerID: uuid.FromStringOrNil("f1b2c3d4-5005-5005-5005-000000000003"),
	}

	m, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Could not marshal the event data. err: %v", err)
	}

	raw := map[string]any{}
	if errUnmarshal := json.Unmarshal(m, &raw); errUnmarshal != nil {
		t.Fatalf("Could not unmarshal the event data. err: %v", errUnmarshal)
	}

	expect := []string{"id", "case_id", "customer_id"}
	if len(raw) != len(expect) {
		t.Errorf("Wrong key count. expect: %d, got: %d (%v)", len(expect), len(raw), raw)
	}
	for _, key := range expect {
		if _, ok := raw[key]; !ok {
			t.Errorf("Missing key. key: %s, payload: %v", key, raw)
		}
	}
}

func Test_CaseNoteEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expect   string
	}{
		{"case_note_created", EventTypeCaseNoteCreated, "case_note_created"},
		{"case_note_deleted", EventTypeCaseNoteDeleted, "case_note_deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, tt.constant)
			}
		})
	}
}
