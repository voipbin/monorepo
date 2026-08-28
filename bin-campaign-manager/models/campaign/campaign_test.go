package campaign

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Compile-time proof that *Campaign carries an explicit subscription address (VOIP-1419).
var _ eventtopic.SubscriptionIdentifier = (*Campaign)(nil)

// TestCampaignEventSubscriptionID pins the subscription address to the campaign's OWN id:
// distinct UUIDs sit on every plausible wrong-answer field, so returning any of them
// (CustomerID, FlowID, NextCampaignID) fails the test.
func TestCampaignEventSubscriptionID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())
	nextCampaignID := uuid.Must(uuid.NewV4())

	c := &Campaign{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		FlowID:         flowID,
		NextCampaignID: nextCampaignID,
	}

	res := c.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}
	if res == customerID.String() {
		t.Errorf("Campaign must not be addressed by its customer id. got: %s", res)
	}
	if res == flowID.String() {
		t.Errorf("Campaign must not be addressed by its flow id. got: %s", res)
	}
	if res == nextCampaignID.String() {
		t.Errorf("Campaign must not be addressed by its next campaign id. got: %s", res)
	}
}

func TestCampaignStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())
	outplanID := uuid.Must(uuid.NewV4())
	outdialID := uuid.Must(uuid.NewV4())
	queueID := uuid.Must(uuid.NewV4())
	nextCampaignID := uuid.Must(uuid.NewV4())

	c := Campaign{
		Type:           TypeCall,
		Execute:        ExecuteRun,
		Name:           "Test Campaign",
		Detail:         "Test campaign details",
		Status:         StatusRun,
		ServiceLevel:   80,
		EndHandle:      EndHandleStop,
		FlowID:         flowID,
		OutplanID:      outplanID,
		OutdialID:      outdialID,
		QueueID:        queueID,
		NextCampaignID: nextCampaignID,
		TMCreate: ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		TMUpdate: ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}
	c.ID = id

	if c.ID != id {
		t.Errorf("Campaign.ID = %v, expected %v", c.ID, id)
	}
	if c.Type != TypeCall {
		t.Errorf("Campaign.Type = %v, expected %v", c.Type, TypeCall)
	}
	if c.Execute != ExecuteRun {
		t.Errorf("Campaign.Execute = %v, expected %v", c.Execute, ExecuteRun)
	}
	if c.Status != StatusRun {
		t.Errorf("Campaign.Status = %v, expected %v", c.Status, StatusRun)
	}
	if c.ServiceLevel != 80 {
		t.Errorf("Campaign.ServiceLevel = %v, expected %v", c.ServiceLevel, 80)
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_call", TypeCall, "call"},
		{"type_flow", TypeFlow, "flow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestExecuteConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Execute
		expected string
	}{
		{"execute_run", ExecuteRun, "run"},
		{"execute_stop", ExecuteStop, "stop"},
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
		{"status_stop", StatusStop, "stop"},
		{"status_stopping", StatusStopping, "stopping"},
		{"status_run", StatusRun, "run"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestEndHandleConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant EndHandle
		expected string
	}{
		{"end_handle_stop", EndHandleStop, "stop"},
		{"end_handle_continue", EndHandleContinue, "continue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
