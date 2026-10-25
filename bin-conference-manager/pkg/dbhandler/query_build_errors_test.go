package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
)

// failingValuer fails when the driver value is requested, which happens while squirrel builds the query.
type failingValuer struct{}

func (failingValuer) Value() (driver.Value, error) {
	return nil, errors.New("value conversion failed")
}

// Test_QueryBuildErrors verifies that a statement which cannot be built is
// reported before any statement is sent to the database.
func Test_QueryBuildErrors(t *testing.T) {
	id := uuid.FromStringOrNil("8e6d4a77-0003-11f1-ad4b-9c5f6e7a8b03")

	tests := []struct {
		name string

		call func(ctx context.Context, h *handler) error

		expectErr string
	}{
		{
			name: "conference create with a field that cannot be serialized",
			call: func(ctx context.Context, h *handler) error {
				return h.ConferenceCreate(ctx, &conference.Conference{
					Identity: commonidentity.Identity{ID: id},
					Data:     map[string]any{"bad": make(chan int)},
				})
			},
			expectErr: "could not prepare fields. ConferenceCreate",
		},
		{
			name: "conference update with a field that cannot be serialized",
			call: func(ctx context.Context, h *handler) error {
				return h.ConferenceUpdate(ctx, id, map[conference.Field]any{
					conference.FieldData: map[string]any{"bad": make(chan int)},
				})
			},
			expectErr: "ConferenceUpdate: prepare fields failed",
		},
		{
			name: "conferencecall update with a field that cannot be serialized",
			call: func(ctx context.Context, h *handler) error {
				return h.ConferencecallUpdate(ctx, id, map[conferencecall.Field]any{
					conferencecall.FieldStatus: []any{make(chan int)},
				})
			},
			expectErr: "ConferencecallUpdate: prepare fields failed",
		},
		{
			name: "conference list with a filter value that cannot be converted",
			call: func(ctx context.Context, h *handler) error {
				_, err := h.ConferenceList(ctx, 10, "", map[conference.Field]any{
					conference.FieldName: commondatabasehandler.NotEq{Value: failingValuer{}},
				})
				return err
			},
			expectErr: "could not build query",
		},
		{
			name: "conferencecall list with a filter value that cannot be converted",
			call: func(ctx context.Context, h *handler) error {
				_, err := h.ConferencecallList(ctx, 10, "", map[conferencecall.Field]any{
					conferencecall.FieldStatus: commondatabasehandler.NotEq{Value: failingValuer{}},
				})
				return err
			},
			expectErr: "could not build query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newSQLMockHandler(t)
			// no statement is expected: sqlmock fails an unexpected statement.

			err := tt.call(context.Background(), h)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error to contain: %q, got: %v", tt.expectErr, err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("Unexpected statement. err: %v", errMet)
			}
		})
	}
}
