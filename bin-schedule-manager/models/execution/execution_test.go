package execution

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
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

// Execution overrides the subscription address of the global topic exchange (VOIP-1404/1405).
// The assertion pins the POINTER receiver: notifyhandler asserts on the dynamic type of the event
// data, which is always a pointer, so a value receiver would silently never be picked up.
var _ eventtopic.SubscriptionIdentifier = (*Execution)(nil)

func TestExecutionEventSubscriptionID(t *testing.T) {
	scheduleID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name      string
		execution *Execution
		expect    string
	}{
		{
			"success run",
			&Execution{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				ScheduleID:  scheduleID,
				TriggerType: TriggerTypeCron,
				Status:      StatusSuccess,
			},
			scheduleID.String(),
		},
		{
			"failed run",
			&Execution{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				ScheduleID:  scheduleID,
				TriggerType: TriggerTypeManual,
				Status:      StatusFailed,
			},
			scheduleID.String(),
		},
		{
			"empty schedule id",
			&Execution{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.execution.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestExecutionEventSubscriptionIDIsNotOwnID pins the property the override exists for: every run
// of one schedule resolves to the SAME address, and that address is never the execution's own id.
// A schedule fires repeatedly and mints a new execution row each time, so a consumer binds
// `schedule-manager.execution.<schedule-id>.#` once and follows every run -- succeeded and failed
// alike.
func TestExecutionEventSubscriptionIDIsNotOwnID(t *testing.T) {
	scheduleID := uuid.Must(uuid.NewV4())

	first := &Execution{
		Identity:   commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		ScheduleID: scheduleID,
		Status:     StatusSuccess,
	}
	second := &Execution{
		Identity:   commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		ScheduleID: scheduleID,
		Status:     StatusFailed,
	}

	if first.ID == second.ID {
		t.Fatalf("Execution ids are expected to differ per run. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != scheduleID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", scheduleID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the execution own id. id: %s", first.ID)
	}
}
