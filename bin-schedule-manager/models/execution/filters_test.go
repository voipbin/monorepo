package execution

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestFieldStruct(t *testing.T) {
	tests := []struct {
		name string

		field FieldStruct

		expectScheduleID uuid.UUID
		expectStatus     Status
		expectDeleted    bool
	}{
		{
			name: "field_struct_with_all_fields",

			field: FieldStruct{
				ScheduleID: uuid.FromStringOrNil("d2b0a3f2-6f14-11f0-8a52-eb6cf3e2d552"),
				Status:     StatusRunning,
				Deleted:    false,
			},

			expectScheduleID: uuid.FromStringOrNil("d2b0a3f2-6f14-11f0-8a52-eb6cf3e2d552"),
			expectStatus:     StatusRunning,
			expectDeleted:    false,
		},
		{
			name: "field_struct_with_deleted_true",

			field: FieldStruct{
				ScheduleID: uuid.FromStringOrNil("d2e4c0e4-6f14-11f0-9138-9f6df8a2b552"),
				Status:     StatusAbandoned,
				Deleted:    true,
			},

			expectScheduleID: uuid.FromStringOrNil("d2e4c0e4-6f14-11f0-9138-9f6df8a2b552"),
			expectStatus:     StatusAbandoned,
			expectDeleted:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.ScheduleID != tt.expectScheduleID {
				t.Errorf("Wrong ScheduleID. expect: %s, got: %s", tt.expectScheduleID, tt.field.ScheduleID)
			}
			if tt.field.Status != tt.expectStatus {
				t.Errorf("Wrong Status. expect: %s, got: %s", tt.expectStatus, tt.field.Status)
			}
			if tt.field.Deleted != tt.expectDeleted {
				t.Errorf("Wrong Deleted. expect: %v, got: %v", tt.expectDeleted, tt.field.Deleted)
			}
		})
	}
}
