package campaign

import (
	"testing"

	"github.com/gofrs/uuid"
)

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
		TMCreate:       "2024-01-01 00:00:00.000000",
		TMUpdate:       "2024-01-01 00:00:00.000000",
		TMDelete:       "9999-01-01T00:00:00.000000Z",
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
