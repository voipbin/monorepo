package conferencecall

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestConferencecallStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	conferenceID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	tmCreate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	cc := Conferencecall{
		ActiveflowID:  activeflowID,
		ConferenceID:  conferenceID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusJoining,
		TMCreate:      &tmCreate,
		TMUpdate:      &tmUpdate,
		TMDelete:      nil,
	}
	cc.ID = id

	if cc.ID != id {
		t.Errorf("Conferencecall.ID = %v, expected %v", cc.ID, id)
	}
	if cc.ActiveflowID != activeflowID {
		t.Errorf("Conferencecall.ActiveflowID = %v, expected %v", cc.ActiveflowID, activeflowID)
	}
	if cc.ConferenceID != conferenceID {
		t.Errorf("Conferencecall.ConferenceID = %v, expected %v", cc.ConferenceID, conferenceID)
	}
	if cc.ReferenceType != ReferenceTypeCall {
		t.Errorf("Conferencecall.ReferenceType = %v, expected %v", cc.ReferenceType, ReferenceTypeCall)
	}
	if cc.Status != StatusJoining {
		t.Errorf("Conferencecall.Status = %v, expected %v", cc.Status, StatusJoining)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_call", ReferenceTypeCall, "call"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
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
		{"status_joining", StatusJoining, "joining"},
		{"status_joined", StatusJoined, "joined"},
		{"status_leaving", StatusLeaving, "leaving"},
		{"status_leaved", StatusLeaved, "leaved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestConvertWebhookMessage(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	conferenceID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	tmCreate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	tmDelete := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	cc := &Conferencecall{
		ActiveflowID:  activeflowID,
		ConferenceID:  conferenceID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusJoined,
		TMCreate:      &tmCreate,
		TMUpdate:      &tmUpdate,
		TMDelete:      &tmDelete,
	}
	cc.ID = id
	cc.CustomerID = customerID

	webhook := cc.ConvertWebhookMessage()

	if webhook.ID != id {
		t.Errorf("WebhookMessage.ID = %v, expected %v", webhook.ID, id)
	}
	if webhook.CustomerID != customerID {
		t.Errorf("WebhookMessage.CustomerID = %v, expected %v", webhook.CustomerID, customerID)
	}
	if webhook.ActiveflowID != activeflowID {
		t.Errorf("WebhookMessage.ActiveflowID = %v, expected %v", webhook.ActiveflowID, activeflowID)
	}
	if webhook.ConferenceID != conferenceID {
		t.Errorf("WebhookMessage.ConferenceID = %v, expected %v", webhook.ConferenceID, conferenceID)
	}
	if webhook.ReferenceType != ReferenceTypeCall {
		t.Errorf("WebhookMessage.ReferenceType = %v, expected %v", webhook.ReferenceType, ReferenceTypeCall)
	}
	if webhook.ReferenceID != referenceID {
		t.Errorf("WebhookMessage.ReferenceID = %v, expected %v", webhook.ReferenceID, referenceID)
	}
	if webhook.Status != StatusJoined {
		t.Errorf("WebhookMessage.Status = %v, expected %v", webhook.Status, StatusJoined)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	conferenceID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	tmCreate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	cc := &Conferencecall{
		ActiveflowID:  activeflowID,
		ConferenceID:  conferenceID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusJoining,
		TMCreate:      &tmCreate,
		TMUpdate:      nil,
		TMDelete:      nil,
	}
	cc.ID = id
	cc.CustomerID = customerID

	data, err := cc.CreateWebhookEvent()
	if err != nil {
		t.Errorf("CreateWebhookEvent failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("CreateWebhookEvent returned empty data")
	}
}

func TestConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		wantErr  bool
		validate func(t *testing.T, result map[Field]any)
	}{
		{
			name: "convert status field",
			input: map[string]string{
				"status": "joined",
			},
			wantErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if len(result) == 0 {
					t.Error("Expected non-empty result")
				}
			},
		},
		{
			name: "convert reference_type field",
			input: map[string]string{
				"reference_type": "call",
			},
			wantErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if len(result) == 0 {
					t.Error("Expected non-empty result")
				}
			},
		},
		{
			name:    "empty input",
			input:   map[string]string{},
			wantErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if len(result) != 0 {
					t.Errorf("Expected empty result, got %d fields", len(result))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringMapToFieldMap(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertStringMapToFieldMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Conferencecall overrides the subscription address of the global topic exchange (VOIP-1404/1405).
// The assertion pins the POINTER type: the event data reaches notifyhandler as a POINTER and the
// assertion matches the dynamic type; a VALUE of this pointer-receiver type would fail the
// assertion (the exact pipecat defect this ticket fixed).
var _ eventtopic.SubscriptionIdentifier = (*Conferencecall)(nil)

func TestConferencecallEventSubscriptionID(t *testing.T) {
	conferenceID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name           string
		conferencecall *Conferencecall
		expect         string
	}{
		{
			"normal",
			&Conferencecall{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				ConferenceID: conferenceID,
				Status:       StatusJoined,
			},
			conferenceID.String(),
		},
		{
			"empty conference id",
			&Conferencecall{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.conferencecall.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestConferencecallEventSubscriptionIDIsNotOwnID pins the property the override exists for:
// every conferencecall of one conference resolves to the SAME address, and that address is never
// the conferencecall's own id. Two distinct participants of one conference must converge, so a
// consumer binds `conference-manager.conferencecall.<conference-id>.#` once and follows the whole
// conference session.
func TestConferencecallEventSubscriptionIDIsNotOwnID(t *testing.T) {
	conferenceID := uuid.Must(uuid.NewV4())

	first := &Conferencecall{
		Identity:     commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		ConferenceID: conferenceID,
	}
	second := &Conferencecall{
		Identity:     commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		ConferenceID: conferenceID,
	}

	if first.ID == second.ID {
		t.Fatalf("Conferencecall ids are expected to differ. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != conferenceID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", conferenceID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the conferencecall own id. id: %s", first.ID)
	}
}
