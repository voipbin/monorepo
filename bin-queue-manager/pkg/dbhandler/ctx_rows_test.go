package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/pkg/cachehandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

func newCtxTestHandler(t *testing.T) (*handler, sqlmock.Sqlmock, *cachehandler.MockCacheHandler) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	return &handler{db: db, cache: mockCache, utilHandler: mockUtil}, mock, mockCache
}

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error: a single-row get must not report it as ErrNotFound
// and a list must not return a partial (here empty) result as a success.
func Test_RowsErr_NotMisreported(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func(h *handler) error

		// the cause is wrapped with %w by the new code only
		expectCause bool
	}{
		{"get queue", func(h *handler) error { _, err := h.queueGetFromDB(ctx, id); return err }, false},
		{"get queuecall", func(h *handler) error { _, err := h.queuecallGetFromDB(ctx, id); return err }, false},
		{"get queuecall by reference id", func(h *handler) error {
			_, err := h.QueuecallGetByReferenceID(ctx, id)
			return err
		}, true},
		{"list queue", func(h *handler) error { _, err := h.QueueList(ctx, 10, "", nil); return err }, false},
		{"list queuecall", func(h *handler) error { _, err := h.QueuecallList(ctx, 10, "", nil); return err }, false},
		{"list queuecall oldest waiting", func(h *handler) error {
			_, err := h.QueuecallListOldestWaiting(ctx, id, 10)
			return err
		}, false},
		{"list queuecall connecting stale", func(h *handler) error {
			_, err := h.QueuecallListConnectingStale(ctx, time.Now(), 10)
			return err
		}, false},
		{"list queuecall waiting oldest", func(h *handler) error {
			_, err := h.QueuecallListWaitingOldest(ctx, 10)
			return err
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache := newCtxTestHandler(t)
			mockCache.EXPECT().QueuecallGetByReferenceID(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss")).AnyTimes()

			// one row whose read fails: Next returns false and the cause is
			// only reported by rows.Err(). (zero rows would leave Err nil.)
			rows := sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()).RowError(0, errRowsBroken)
			mock.ExpectQuery("SELECT").WillReturnRows(rows)

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
			}
			if tt.expectCause && !errors.Is(err, errRowsBroken) {
				t.Fatalf("the cause must be preserved with %%w. err: %v", err)
			}
		})
	}
}

// Test_ExecContext_Canceled verifies that the queue counter updates pass the
// request context to the database. The expected exec is registered and the
// context is already canceled: database/sql checks the context before it
// reaches the driver, so the expectation must stay unconsumed. A call that
// ignores the context (db.Exec) consumes the expectation and fails this test.
// (The error text wraps the cause with %v, so the unconsumed expectation is
// the assertion that tells the two apart.)
func Test_ExecContext_Canceled(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
	queueCallID := uuid.FromStringOrNil("5b1c7d2a-1b2d-11f1-9c3e-4f8a2d6e0b11")

	tests := []struct {
		name string
		fn   func(h *handler, ctx context.Context) error
	}{
		{"QueueAddWaitQueueCallID", func(h *handler, ctx context.Context) error { return h.QueueAddWaitQueueCallID(ctx, id, queueCallID) }},
		{"QueueIncreaseTotalServicedCount", func(h *handler, ctx context.Context) error {
			return h.QueueIncreaseTotalServicedCount(ctx, id, queueCallID)
		}},
		{"QueueIncreaseTotalAbandonedCount", func(h *handler, ctx context.Context) error {
			return h.QueueIncreaseTotalAbandonedCount(ctx, id, queueCallID)
		}},
		{"QueueRemoveServiceQueueCall", func(h *handler, ctx context.Context) error {
			return h.QueueRemoveServiceQueueCall(ctx, id, queueCallID)
		}},
		{"QueueRemoveWaitQueueCall", func(h *handler, ctx context.Context) error {
			return h.QueueRemoveWaitQueueCall(ctx, id, queueCallID)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache := newCtxTestHandler(t)
			mockCache.EXPECT().QueueGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss")).AnyTimes()
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			mock.ExpectExec("update queue_queues").WillReturnResult(sqlmock.NewResult(0, 1))

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			if err := tt.fn(h, ctx); err == nil {
				t.Fatalf("expected an error for the canceled context, got nil")
			}
			if err := mock.ExpectationsWereMet(); err == nil {
				t.Fatalf("the exec was executed with a canceled context; it must not reach the driver")
			}
		})
	}
}
