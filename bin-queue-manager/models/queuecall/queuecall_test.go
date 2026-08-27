package queuecall

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestQueuecallStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	queueID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	serviceAgentID := uuid.Must(uuid.NewV4())

	qc := Queuecall{
		QueueID:          queueID,
		ReferenceType:    ReferenceTypeCall,
		ReferenceID:      referenceID,
		Status:           StatusWaiting,
		ServiceAgentID:   serviceAgentID,
		TimeoutWait:      60000,
		TimeoutService:   300000,
		DurationWaiting:  15000,
		DurationService:  120000,
	}
	qc.ID = id
	qc.CustomerID = customerID

	if qc.ID != id {
		t.Errorf("Queuecall.ID = %v, expected %v", qc.ID, id)
	}
	if qc.CustomerID != customerID {
		t.Errorf("Queuecall.CustomerID = %v, expected %v", qc.CustomerID, customerID)
	}
	if qc.QueueID != queueID {
		t.Errorf("Queuecall.QueueID = %v, expected %v", qc.QueueID, queueID)
	}
	if qc.ReferenceType != ReferenceTypeCall {
		t.Errorf("Queuecall.ReferenceType = %v, expected %v", qc.ReferenceType, ReferenceTypeCall)
	}
	if qc.ReferenceID != referenceID {
		t.Errorf("Queuecall.ReferenceID = %v, expected %v", qc.ReferenceID, referenceID)
	}
	if qc.Status != StatusWaiting {
		t.Errorf("Queuecall.Status = %v, expected %v", qc.Status, StatusWaiting)
	}
	if qc.ServiceAgentID != serviceAgentID {
		t.Errorf("Queuecall.ServiceAgentID = %v, expected %v", qc.ServiceAgentID, serviceAgentID)
	}
	if qc.TimeoutWait != 60000 {
		t.Errorf("Queuecall.TimeoutWait = %v, expected %v", qc.TimeoutWait, 60000)
	}
	if qc.TimeoutService != 300000 {
		t.Errorf("Queuecall.TimeoutService = %v, expected %v", qc.TimeoutService, 300000)
	}
	if qc.DurationWaiting != 15000 {
		t.Errorf("Queuecall.DurationWaiting = %v, expected %v", qc.DurationWaiting, 15000)
	}
	if qc.DurationService != 120000 {
		t.Errorf("Queuecall.DurationService = %v, expected %v", qc.DurationService, 120000)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	if ReferenceTypeCall != "call" {
		t.Errorf("ReferenceTypeCall = %v, expected %v", ReferenceTypeCall, "call")
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_initiating", StatusInitiating, "initiating"},
		{"status_waiting", StatusWaiting, "waiting"},
		{"status_connecting", StatusConnecting, "connecting"},
		{"status_kicking", StatusKicking, "kicking"},
		{"status_service", StatusService, "service"},
		{"status_done", StatusDone, "done"},
		{"status_abandoned", StatusAbandoned, "abandoned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

// Queuecall overrides the subscription address of the global topic exchange (VOIP-1404/1405).
// The assertion pins the POINTER receiver: notifyhandler asserts on the dynamic type of the event
// data, which is always a pointer, so a value receiver would silently never be picked up.
var _ eventtopic.SubscriptionIdentifier = (*Queuecall)(nil)

func TestQueuecallEventSubscriptionID(t *testing.T) {
	queueID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name      string
		queuecall *Queuecall
		expect    string
	}{
		{
			"normal",
			&Queuecall{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				QueueID: queueID,
				Status:  StatusWaiting,
			},
			queueID.String(),
		},
		{
			"empty queue id",
			&Queuecall{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.queuecall.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestQueuecallEventSubscriptionIDIsNotOwnID pins the property the override exists for: every
// queuecall of one queue resolves to the SAME address, and that address is never the queuecall's
// own id. Two distinct queuecalls of one queue must converge, so a consumer binds
// `queue-manager.queuecall.<queue-id>.#` once and follows the whole queue.
func TestQueuecallEventSubscriptionIDIsNotOwnID(t *testing.T) {
	queueID := uuid.Must(uuid.NewV4())

	first := &Queuecall{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		QueueID:  queueID,
	}
	second := &Queuecall{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		QueueID:  queueID,
	}

	if first.ID == second.ID {
		t.Fatalf("Queuecall ids are expected to differ. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != queueID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", queueID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the queuecall own id. id: %s", first.ID)
	}
}
