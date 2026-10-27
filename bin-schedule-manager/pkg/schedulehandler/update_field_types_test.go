package schedulehandler

import (
	"context"
	"errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"

	"monorepo/bin-schedule-manager/models/schedule"
)

// Test_validateUpdateFields_non_string_values verifies that a type, cron or
// name of the wrong Go type is rejected with the field's own reason. The
// handler has no dependencies, so reaching the name uniqueness check or the
// database would panic instead of returning.
func Test_validateUpdateFields_non_string_values(t *testing.T) {
	tests := []struct {
		name   string
		fields map[schedule.Field]any

		expectReason  string
		expectMessage string
	}{
		{"type is a number", map[schedule.Field]any{schedule.FieldType: 123}, "SCHEDULE_TYPE_INVALID", "The schedule type is invalid."},
		{"cron is a number", map[schedule.Field]any{schedule.FieldCron: 123}, "SCHEDULE_CRON_INVALID", "The cron expression is invalid."},
		{"name is a number", map[schedule.Field]any{schedule.FieldName: 123}, "SCHEDULE_NAME_EMPTY", "The schedule name must not be empty."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &scheduleHandler{}

			err := h.validateUpdateFields(context.Background(), &schedule.Schedule{}, tt.fields)

			var vErr *cerrors.VoipbinError
			if !errors.As(err, &vErr) {
				t.Fatalf("Wrong match. expect: a typed error, got: %v", err)
			}
			if vErr.Status != cerrors.StatusInvalidArgument || vErr.Reason != tt.expectReason || vErr.Message != tt.expectMessage {
				t.Errorf("Wrong match. expect: %s/%s/%q, got: %s/%s/%q", cerrors.StatusInvalidArgument, tt.expectReason, tt.expectMessage, vErr.Status, vErr.Reason, vErr.Message)
			}
		})
	}
}
