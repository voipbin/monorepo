package dbhandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/models/failedevent"
)

// Test_Update_unsupportedFieldValue verifies that an update carrying a value the database layer
// cannot convert is rejected before any statement is sent.
func Test_Update_unsupportedFieldValue(t *testing.T) {
	id := uuid.FromStringOrNil("f3000000-0000-0000-0000-000000000001")
	// a slice that cannot be encoded as JSON cannot be stored in a column
	unsupported := []any{make(chan int)}

	tests := []struct {
		name      string
		fn        func(h *handler) error
		expectMsg string
	}{
		{"account", func(h *handler) error {
			return h.AccountUpdate(context.Background(), id, map[account.Field]any{account.FieldName: unsupported})
		}, "AccountUpdate: could not prepare fields"},
		{"billing", func(h *handler) error {
			return h.BillingUpdate(context.Background(), id, map[billing.Field]any{billing.FieldStatus: unsupported})
		}, "BillingUpdate: could not prepare fields"},
		{"failed event", func(h *handler) error {
			return h.FailedEventUpdate(context.Background(), id, map[failedevent.Field]any{failedevent.FieldStatus: unsupported})
		}, "FailedEventUpdate: could not prepare fields"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no statement is expected on the mock: sqlmock fails the call if one is sent
			h, _, mockUtil, _ := newFailureHandler(t)
			mockUtil.EXPECT().TimeNow().Return(nil)

			err := tt.fn(h)
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("wrong error. expect to contain: %q, got: %v", tt.expectMsg, err)
			}
		})
	}
}

// Test_List_unsupportedFilterValue verifies that a list with a filter value the database layer
// cannot apply is rejected before any statement is sent.
func Test_List_unsupportedFilterValue(t *testing.T) {
	tests := []struct {
		name      string
		fn        func(h *handler) error
		expectMsg string
	}{
		{"account", func(h *handler) error {
			_, err := h.AccountList(context.Background(), 10, "", map[account.Field]any{account.FieldName: make(chan int)})
			return err
		}, "AccountList: could not apply filters"},
		{"billing", func(h *handler) error {
			_, err := h.BillingList(context.Background(), 10, "", map[billing.Field]any{billing.FieldStatus: make(chan int)})
			return err
		}, "BillingList: could not apply filters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, mockUtil, _ := newFailureHandler(t)
			mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

			err := tt.fn(h)
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("wrong error. expect to contain: %q, got: %v", tt.expectMsg, err)
			}
		})
	}
}
