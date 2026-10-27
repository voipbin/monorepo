package dbhandler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/models/trunk"
)

// Test_update_unencodableValue verifies that an update whose value cannot be encoded for the
// database is rejected before any statement is sent (the sqlmock has no expectation, so a
// statement would fail the call with a different text).
func Test_update_unencodableValue(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("4a7c9e1b-6d2f-11f1-b3a5-5e6f7a8b9c01")
	// a value inside a map that has no json form
	bad := map[string]any{"case": make(chan int)}
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		fn   func(h *handler) error

		expectErr string
	}{
		{
			name: "extension",
			fn: func(h *handler) error {
				return h.ExtensionUpdate(ctx, id, map[extension.Field]any{extension.FieldDetail: bad})
			},
			expectErr: "ExtensionUpdate: prepare fields failed",
		},
		{
			name: "trunk",
			fn: func(h *handler) error {
				return h.TrunkUpdate(ctx, id, map[trunk.Field]any{trunk.FieldDetail: bad})
			},
			expectErr: "TrunkUpdate: prepare fields failed",
		},
		{
			name: "sipauth",
			fn: func(h *handler) error {
				return h.SIPAuthUpdate(ctx, id, map[sipauth.Field]any{sipauth.FieldUsername: bad})
			},
			expectErr: "SIPAuthUpdate: prepare fields failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _, mockUtil := newErrorPathHandler(t)
			mockUtil.EXPECT().TimeNow().Return(&now)

			err := tt.fn(h)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect an error containing %q, got: %v", tt.expectErr, err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("Wrong match. no statement must be sent: %v", errMet)
			}
		})
	}
}
