package transcribe

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Transcribe carries the explicit subscription address of the global topic exchange
// (VOIP-1404/1419). The assertion pins the POINTER type: the event data reaches notifyhandler
// as a pointer and the eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Transcribe)(nil)

func TestTranscribeEventSubscriptionID(t *testing.T) {
	ownID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	onEndFlowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	hostID := uuid.Must(uuid.NewV4())

	tr := &Transcribe{
		Identity: commonidentity.Identity{
			ID:         ownID,
			CustomerID: customerID,
		},
		ActiveflowID:  activeflowID,
		OnEndFlowID:   onEndFlowID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		HostID:        hostID,
	}

	res := tr.EventSubscriptionID()
	if res != ownID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", ownID.String(), res)
	}

	// Mutation checks: the subscription address is the transcribe's OWN id — none of the other
	// id-shaped fields on the struct may ever leak into the routing key.
	if res == customerID.String() {
		t.Errorf("Subscription address must not be the customer id. id: %s", customerID)
	}
	if res == activeflowID.String() {
		t.Errorf("Subscription address must not be the activeflow id. id: %s", activeflowID)
	}
	if res == onEndFlowID.String() {
		t.Errorf("Subscription address must not be the on-end-flow id. id: %s", onEndFlowID)
	}
	if res == referenceID.String() {
		t.Errorf("Subscription address must not be the reference id. id: %s", referenceID)
	}
	if res == hostID.String() {
		t.Errorf("Subscription address must not be the host id. id: %s", hostID)
	}
}

func TestTranscribeEventSubscriptionIDEmpty(t *testing.T) {
	tr := &Transcribe{}

	if res := tr.EventSubscriptionID(); res != uuid.Nil.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", uuid.Nil.String(), res)
	}
}

func TestTranscribeStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	onEndFlowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	hostID := uuid.Must(uuid.NewV4())
	streamingID1 := uuid.Must(uuid.NewV4())
	streamingID2 := uuid.Must(uuid.NewV4())

	tmCreate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	tr := Transcribe{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ActiveflowID:  activeflowID,
		OnEndFlowID:   onEndFlowID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusProgressing,
		HostID:        hostID,
		Language:      "en-US",
		Direction:     DirectionBoth,
		StreamingIDs:  []uuid.UUID{streamingID1, streamingID2},
		TMCreate:      &tmCreate,
		TMUpdate:      &tmUpdate,
		TMDelete:      nil,
	}

	if tr.ID != id {
		t.Errorf("Transcribe.ID = %v, expected %v", tr.ID, id)
	}
	if tr.CustomerID != customerID {
		t.Errorf("Transcribe.CustomerID = %v, expected %v", tr.CustomerID, customerID)
	}
	if tr.ActiveflowID != activeflowID {
		t.Errorf("Transcribe.ActiveflowID = %v, expected %v", tr.ActiveflowID, activeflowID)
	}
	if tr.OnEndFlowID != onEndFlowID {
		t.Errorf("Transcribe.OnEndFlowID = %v, expected %v", tr.OnEndFlowID, onEndFlowID)
	}
	if tr.ReferenceType != ReferenceTypeCall {
		t.Errorf("Transcribe.ReferenceType = %v, expected %v", tr.ReferenceType, ReferenceTypeCall)
	}
	if tr.ReferenceID != referenceID {
		t.Errorf("Transcribe.ReferenceID = %v, expected %v", tr.ReferenceID, referenceID)
	}
	if tr.Status != StatusProgressing {
		t.Errorf("Transcribe.Status = %v, expected %v", tr.Status, StatusProgressing)
	}
	if tr.HostID != hostID {
		t.Errorf("Transcribe.HostID = %v, expected %v", tr.HostID, hostID)
	}
	if tr.Language != "en-US" {
		t.Errorf("Transcribe.Language = %v, expected %v", tr.Language, "en-US")
	}
	if tr.Direction != DirectionBoth {
		t.Errorf("Transcribe.Direction = %v, expected %v", tr.Direction, DirectionBoth)
	}
	if len(tr.StreamingIDs) != 2 {
		t.Errorf("Transcribe.StreamingIDs length = %v, expected %v", len(tr.StreamingIDs), 2)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_unknown", ReferenceTypeUnknown, "unknown"},
		{"reference_type_recording", ReferenceTypeRecording, "recording"},
		{"reference_type_call", ReferenceTypeCall, "call"},
		{"reference_type_confbridge", ReferenceTypeConfbridge, "confbridge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDirectionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Direction
		expected string
	}{
		{"direction_both", DirectionBoth, "both"},
		{"direction_in", DirectionIn, "in"},
		{"direction_out", DirectionOut, "out"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDirectionNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    Direction
		expected Direction
	}{
		{"both_stays_both", DirectionBoth, DirectionBoth},
		{"in_stays_in", DirectionIn, DirectionIn},
		{"out_stays_out", DirectionOut, DirectionOut},
		{"empty_falls_back_to_both", Direction(""), DirectionBoth},
		{"invalid_falls_back_to_both", Direction("foo"), DirectionBoth},
		{"uppercase_falls_back_to_both", Direction("BOTH"), DirectionBoth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Normalize()
			if result != tt.expected {
				t.Errorf("Direction(%q).Normalize() = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_progressing", StatusProgressing, "progressing"},
		{"status_done", StatusDone, "done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestIsUpdatableStatus(t *testing.T) {
	tests := []struct {
		name      string
		oldStatus Status
		newStatus Status
		expected  bool
	}{
		{"progressing_to_done", StatusProgressing, StatusDone, true},
		{"progressing_to_progressing", StatusProgressing, StatusProgressing, false},
		{"done_to_done", StatusDone, StatusDone, false},
		{"done_to_progressing", StatusDone, StatusProgressing, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUpdatableStatus(tt.oldStatus, tt.newStatus)
			if result != tt.expected {
				t.Errorf("IsUpdatableStatus(%s, %s) = %v, expected %v", tt.oldStatus, tt.newStatus, result, tt.expected)
			}
		})
	}
}
