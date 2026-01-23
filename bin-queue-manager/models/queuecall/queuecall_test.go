package queuecall

import (
	"testing"

	"github.com/gofrs/uuid"
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
