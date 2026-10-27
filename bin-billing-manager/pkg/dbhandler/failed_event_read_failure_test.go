package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
)

// Test_FailedEventListPendingRetry_readFailures verifies that every failure of the retry-queue read
// is returned to the caller with its cause instead of an empty list.
func Test_FailedEventListPendingRetry_readFailures(t *testing.T) {
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	tests := []struct {
		name         string
		prepare      func(mock sqlmock.Sqlmock)
		expectMsg    string
		expectCause  error
		expectUnwrap bool
	}{
		{
			name:         "query fails",
			prepare:      func(mock sqlmock.Sqlmock) { mock.ExpectQuery("SELECT").WillReturnError(errCause) },
			expectMsg:    "FailedEventListPendingRetry: could not query",
			expectCause:  errCause,
			expectUnwrap: true,
		},
		{
			name: "a row cannot be scanned",
			prepare: func(mock sqlmock.Sqlmock) {
				// a single column cannot be scanned into the model's destination list
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow([]byte{1}))
			},
			expectMsg: "FailedEventListPendingRetry: could not scan row",
		},
		{
			name: "row iteration fails",
			prepare: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow([]byte{1}).RowError(0, errRowsBroken)
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			expectMsg:    "FailedEventListPendingRetry: row iteration error",
			expectCause:  errRowsBroken,
			expectUnwrap: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _, _ := newFailureHandler(t)
			tt.prepare(mock)

			res, err := h.FailedEventListPendingRetry(context.Background(), now)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if res != nil {
				t.Errorf("no partial result may be returned. got: %v", res)
			}
			if !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("wrong error. expect to contain: %q, got: %v", tt.expectMsg, err)
			}
			if tt.expectUnwrap && !errors.Is(err, tt.expectCause) {
				t.Errorf("the cause must be kept. expect: %v, got: %v", tt.expectCause, err)
			}
		})
	}
}

// Test_BillingGetByIdempotencyKey_notFound verifies that an unknown key is reported as ErrNotFound.
func Test_BillingGetByIdempotencyKey_notFound(t *testing.T) {
	h, mock, _, _ := newFailureHandler(t)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := h.BillingGetByIdempotencyKey(context.Background(), uuid.FromStringOrNil("f2000000-0000-0000-0000-000000000001"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("wrong error. expect: ErrNotFound, got: %v", err)
	}
}
