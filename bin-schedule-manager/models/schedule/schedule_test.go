package schedule

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Schedule carries an explicit subscription address on the global topic exchange
// (VOIP-1404/1405, mandatory since VOIP-1419). The assertion pins the POINTER type: the event
// data reaches notifyhandler as a pointer and the interface check matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Schedule)(nil)

// TestScheduleEventSubscriptionID asserts the subscription address is the schedule's OWN id —
// not the customer id, the other uuid-typed field a wrong implementation could plausibly
// return. Both uuids are distinct, so returning the wrong field fails loudly (mutation check).
func TestScheduleEventSubscriptionID(t *testing.T) {
	scheduleID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	data := &Schedule{
		Identity: commonidentity.Identity{
			ID:         scheduleID,
			CustomerID: customerID,
		},
		Type:    TypeRPC,
		Cron:    "*/5 * * * *",
		Enabled: true,
	}

	res := data.EventSubscriptionID()
	if res != scheduleID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", scheduleID.String(), res)
	}
	if res == customerID.String() {
		t.Errorf("Schedule must not be addressed by its customer id. got: %s", res)
	}
}

func TestSchedule(t *testing.T) {
	tests := []struct {
		name string

		scheduleName string
		detail       string
		scheduleType Type
		cron         string
		targetQueue  string
		targetURI    string
		targetMethod string
		targetData   json.RawMessage
		timeoutMS    int
		retryMax     int
		enabled      bool
	}{
		{
			name: "creates_schedule_with_all_fields",

			scheduleName: "number-renew",
			detail:       "daily number renewal",
			scheduleType: TypeRPC,
			cron:         "0 2 * * *",
			targetQueue:  "bin-manager.number-manager.request",
			targetURI:    "/v1/numbers/renew",
			targetMethod: "POST",
			targetData:   json.RawMessage(`{"days":3}`),
			timeoutMS:    300000,
			retryMax:     0,
			enabled:      true,
		},
		{
			name: "creates_schedule_with_empty_fields",

			scheduleName: "",
			detail:       "",
			scheduleType: "",
			cron:         "",
			targetQueue:  "",
			targetURI:    "",
			targetMethod: "",
			targetData:   nil,
			timeoutMS:    0,
			retryMax:     0,
			enabled:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Schedule{
				Name:         tt.scheduleName,
				Detail:       tt.detail,
				Type:         tt.scheduleType,
				Cron:         tt.cron,
				TargetQueue:  tt.targetQueue,
				TargetURI:    tt.targetURI,
				TargetMethod: tt.targetMethod,
				TargetData:   tt.targetData,
				TimeoutMS:    tt.timeoutMS,
				RetryMax:     tt.retryMax,
				Enabled:      tt.enabled,
			}

			if s.Name != tt.scheduleName {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.scheduleName, s.Name)
			}
			if s.Detail != tt.detail {
				t.Errorf("Wrong Detail. expect: %s, got: %s", tt.detail, s.Detail)
			}
			if s.Type != tt.scheduleType {
				t.Errorf("Wrong Type. expect: %s, got: %s", tt.scheduleType, s.Type)
			}
			if s.Cron != tt.cron {
				t.Errorf("Wrong Cron. expect: %s, got: %s", tt.cron, s.Cron)
			}
			if s.TargetQueue != tt.targetQueue {
				t.Errorf("Wrong TargetQueue. expect: %s, got: %s", tt.targetQueue, s.TargetQueue)
			}
			if s.Enabled != tt.enabled {
				t.Errorf("Wrong Enabled. expect: %v, got: %v", tt.enabled, s.Enabled)
			}
		})
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name string

		constant Type
		expected string
	}{
		{
			name: "type_rpc",

			constant: TypeRPC,
			expected: "rpc",
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
