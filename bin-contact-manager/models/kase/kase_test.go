package kase

import (
	"encoding/json"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Test_Case_ConstructAndMarshal verifies the full Case struct (design §3.1)
// constructs with all fields and marshals without error, including the
// nullable fields (ContactID, ClosedAt, ClosedByID, PreviousCaseID) both
// set and nil.
func Test_Case_ConstructAndMarshal(t *testing.T) {
	id := uuid.FromStringOrNil("f1b2c3d4-3001-3001-3001-000000000001")
	customerID := uuid.FromStringOrNil("f1b2c3d4-3001-3001-3001-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-3001-3001-3001-000000000003")
	ownerID := uuid.FromStringOrNil("f1b2c3d4-3001-3001-3001-000000000004")
	closedByID := uuid.FromStringOrNil("f1b2c3d4-3001-3001-3001-000000000005")
	previousCaseID := uuid.FromStringOrNil("f1b2c3d4-3001-3001-3001-000000000006")
	openedAt := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	closedAt := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)

	c := &Case{
		ID:         id,
		CustomerID: customerID,

		PeerType:      commonaddress.TypeTel,
		PeerTarget:    "+15551234567",
		ReferenceType: "call",

		ContactID: &contactID,

		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   ownerID,
		},

		Status:       StatusOpen,
		OpenedAt:     &openedAt,
		ClosedAt:     &closedAt,
		ClosedReason: ClosedReasonAgentClosed,
		ClosedByType: ClosedByTypeAgent,
		ClosedByID:   &closedByID,

		PreviousCaseID: &previousCaseID,

		TMCreate: &openedAt,
		TMUpdate: &closedAt,
	}

	if c.ID != id {
		t.Errorf("wrong ID: %v", c.ID)
	}
	if c.ContactID == nil || *c.ContactID != contactID {
		t.Errorf("wrong ContactID: %v", c.ContactID)
	}
	if c.OwnerType != commonidentity.OwnerTypeAgent {
		t.Errorf("wrong OwnerType: %v", c.OwnerType)
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if raw["status"] != string(StatusOpen) {
		t.Errorf("expected status: %v, got: %v", StatusOpen, raw["status"])
	}
}

// Test_Case_NilOptionalFields verifies the common construction shape: a
// freshly-opened Case has ContactID, ClosedAt, ClosedByID, PreviousCaseID
// all nil (no prior contact match, not yet closed, first case for this peer).
func Test_Case_NilOptionalFields(t *testing.T) {
	c := &Case{
		ID:            uuid.FromStringOrNil("f1b2c3d4-3001-3001-3001-000000000007"),
		CustomerID:    uuid.FromStringOrNil("f1b2c3d4-3001-3001-3001-000000000008"),
		PeerType:      commonaddress.TypeTel,
		PeerTarget:    "+15559876543",
		ReferenceType: "conversation_message",
		Status:        StatusOpen,
	}

	if c.ContactID != nil {
		t.Errorf("expected nil ContactID, got: %v", *c.ContactID)
	}
	if c.ClosedAt != nil {
		t.Errorf("expected nil ClosedAt, got: %v", *c.ClosedAt)
	}
	if c.ClosedByID != nil {
		t.Errorf("expected nil ClosedByID, got: %v", *c.ClosedByID)
	}
	if c.PreviousCaseID != nil {
		t.Errorf("expected nil PreviousCaseID, got: %v", *c.PreviousCaseID)
	}
}

func Test_StatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_open", StatusOpen, "open"},
		{"status_closed", StatusClosed, "closed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}

func Test_ClosedReasonConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"closed_reason_agent_closed", ClosedReasonAgentClosed, "agent_closed"},
		{"closed_reason_timeout", ClosedReasonTimeout, "timeout"},
		{"closed_reason_merged", ClosedReasonMerged, "merged"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}

func Test_ClosedByTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"closed_by_type_agent", ClosedByTypeAgent, "agent"},
		{"closed_by_type_system", ClosedByTypeSystem, "system"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}
