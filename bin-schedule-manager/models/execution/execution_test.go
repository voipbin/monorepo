package execution

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestExecution(t *testing.T) {
	tests := []struct {
		name string

		scheduleID   uuid.UUID
		triggerType  TriggerType
		status       Status
		statusCode   int
		errString    string
		result       string
		attemptCount int
		durationMS   int
	}{
		{
			name: "creates_execution_with_all_fields",

			scheduleID:   uuid.FromStringOrNil("b3e6a3f2-6f14-11f0-b2ac-9f7cf3e2d552"),
			triggerType:  TriggerTypeCron,
			status:       StatusSuccess,
			statusCode:   200,
			errString:    "",
			result:       `{"renewed":3}`,
			attemptCount: 1,
			durationMS:   1523,
		},
		{
			name: "creates_failed_manual_execution",

			scheduleID:   uuid.FromStringOrNil("b41a54c8-6f14-11f0-b7cd-33cfc3b2e552"),
			triggerType:  TriggerTypeManual,
			status:       StatusFailed,
			statusCode:   500,
			errString:    "request failed",
			result:       "",
			attemptCount: 3,
			durationMS:   90000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Execution{
				ScheduleID:   tt.scheduleID,
				TriggerType:  tt.triggerType,
				Status:       tt.status,
				StatusCode:   tt.statusCode,
				Error:        tt.errString,
				Result:       tt.result,
				AttemptCount: tt.attemptCount,
				DurationMS:   tt.durationMS,
			}

			if e.ScheduleID != tt.scheduleID {
				t.Errorf("Wrong ScheduleID. expect: %s, got: %s", tt.scheduleID, e.ScheduleID)
			}
			if e.TriggerType != tt.triggerType {
				t.Errorf("Wrong TriggerType. expect: %s, got: %s", tt.triggerType, e.TriggerType)
			}
			if e.Status != tt.status {
				t.Errorf("Wrong Status. expect: %s, got: %s", tt.status, e.Status)
			}
			if e.StatusCode != tt.statusCode {
				t.Errorf("Wrong StatusCode. expect: %d, got: %d", tt.statusCode, e.StatusCode)
			}
			if e.AttemptCount != tt.attemptCount {
				t.Errorf("Wrong AttemptCount. expect: %d, got: %d", tt.attemptCount, e.AttemptCount)
			}
		})
	}
}

func TestTriggerTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant TriggerType
		expected string
	}{
		{
			name:     "trigger_type_cron",
			constant: TriggerTypeCron,
			expected: "cron",
		},
		{
			name:     "trigger_type_manual",
			constant: TriggerTypeManual,
			expected: "manual",
		},
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
		{
			name:     "status_running",
			constant: StatusRunning,
			expected: "running",
		},
		{
			name:     "status_success",
			constant: StatusSuccess,
			expected: "success",
		},
		{
			name:     "status_failed",
			constant: StatusFailed,
			expected: "failed",
		},
		{
			name:     "status_abandoned",
			constant: StatusAbandoned,
			expected: "abandoned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
