package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
)

var (
	errTestBoom = errors.New("simulated database failure")

	testQueueID     = uuid.FromStringOrNil("7a1b2c3d-1001-11f1-8a00-000000000001")
	testQueuecallID = uuid.FromStringOrNil("7a1b2c3d-1002-11f1-8a00-000000000002")
	testOtherID     = uuid.FromStringOrNil("7a1b2c3d-1003-11f1-8a00-000000000003")
)

type dbErrCase struct {
	name string
	// expect is the regexp of the statement that fails.
	expect string
	fn     func(ctx context.Context, h *handler) error
}

// dbExecErrCases are the functions that issue a single write statement and
// return only an error.
func dbExecErrCases() []dbErrCase {
	tmNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return []dbErrCase{
		{"QueueCreate", "(?i)insert into queue_queues", func(ctx context.Context, h *handler) error {
			return h.QueueCreate(ctx, &queue.Queue{Identity: commonidentity.Identity{ID: testQueueID}, Name: "q"})
		}},
		{"QueueUpdate", "(?i)update queue_queues", func(ctx context.Context, h *handler) error {
			return h.QueueUpdate(ctx, testQueueID, map[queue.Field]any{queue.FieldName: "renamed"})
		}},
		{"QueueDelete", "(?i)update queue_queues", func(ctx context.Context, h *handler) error {
			return h.QueueDelete(ctx, testQueueID)
		}},
		{"QueueAddWaitQueueCallID", "(?i)update queue_queues", func(ctx context.Context, h *handler) error {
			return h.QueueAddWaitQueueCallID(ctx, testQueueID, testQueuecallID)
		}},
		{"QueueIncreaseTotalServicedCount", "(?i)update queue_queues", func(ctx context.Context, h *handler) error {
			return h.QueueIncreaseTotalServicedCount(ctx, testQueueID, testQueuecallID)
		}},
		{"QueueIncreaseTotalAbandonedCount", "(?i)update queue_queues", func(ctx context.Context, h *handler) error {
			return h.QueueIncreaseTotalAbandonedCount(ctx, testQueueID, testQueuecallID)
		}},
		{"QueueRemoveServiceQueueCall", "(?i)update queue_queues", func(ctx context.Context, h *handler) error {
			return h.QueueRemoveServiceQueueCall(ctx, testQueueID, testQueuecallID)
		}},
		{"QueueRemoveWaitQueueCall", "(?i)update queue_queues", func(ctx context.Context, h *handler) error {
			return h.QueueRemoveWaitQueueCall(ctx, testQueueID, testQueuecallID)
		}},
		{"QueuecallCreate", "(?i)insert into queue_queuecalls", func(ctx context.Context, h *handler) error {
			return h.QueuecallCreate(ctx, &queuecall.Queuecall{Identity: commonidentity.Identity{ID: testQueuecallID}})
		}},
		{"QueuecallUpdate", "(?i)update queue_queuecalls", func(ctx context.Context, h *handler) error {
			return h.QueuecallUpdate(ctx, testQueuecallID, map[queuecall.Field]any{queuecall.FieldStatus: queuecall.StatusDone})
		}},
		{"QueuecallDelete", "(?i)update queue_queuecalls", func(ctx context.Context, h *handler) error {
			return h.QueuecallDelete(ctx, testQueuecallID)
		}},
		{"QueuecallSetStatusKicking", "(?i)update queue_queuecalls", func(ctx context.Context, h *handler) error {
			return h.QueuecallSetStatusKicking(ctx, testQueuecallID)
		}},
		{"QueuecallSetStatusConnecting", "(?i)update queue_queuecalls", func(ctx context.Context, h *handler) error {
			_, err := h.QueuecallSetStatusConnecting(ctx, testQueuecallID, testOtherID, testOtherID)
			return err
		}},
		{"QueuecallSetStatusService", "(?i)update queue_queuecalls", func(ctx context.Context, h *handler) error {
			_, err := h.QueuecallSetStatusService(ctx, testQueuecallID, 10, &tmNow)
			return err
		}},
		{"QueuecallSetStatusAbandoned", "(?i)update queue_queuecalls", func(ctx context.Context, h *handler) error {
			_, err := h.QueuecallSetStatusAbandoned(ctx, testQueuecallID, 10, &tmNow)
			return err
		}},
		{"QueuecallSetStatusDone", "(?i)update queue_queuecalls", func(ctx context.Context, h *handler) error {
			_, err := h.QueuecallSetStatusDone(ctx, testQueuecallID, 10, &tmNow)
			return err
		}},
		{"QueuecallSetStatusWaitingIfInitiating", "(?i)update queue_queuecalls", func(ctx context.Context, h *handler) error {
			_, err := h.QueuecallSetStatusWaitingIfInitiating(ctx, testQueuecallID)
			return err
		}},
		{"QueuecallSetStatusWaitingIfConnecting", "(?i)update queue_queuecalls", func(ctx context.Context, h *handler) error {
			_, err := h.QueuecallSetStatusWaitingIfConnecting(ctx, testQueuecallID)
			return err
		}},
	}
}

// Test_dbExecError verifies that a failing write statement is returned to the
// caller and that nothing is written to the cache afterwards (the cache mock
// has no expectation, so any cache call fails the test).
func Test_dbExecError(t *testing.T) {
	for _, tt := range dbExecErrCases() {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxTestHandler(t)
			mock.ExpectExec(tt.expect).WillReturnError(errTestBoom)

			err := tt.fn(context.Background(), h)
			if !errors.Is(err, errTestBoom) {
				t.Errorf("Wrong match. expect: %v, got: %v", errTestBoom, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Wrong match. expectations: %v", err)
			}
		})
	}
}

// Test_dbRowsAffectedError verifies that a failure to read the number of
// affected rows of the compare-and-swap writes is returned and nothing is
// written to the cache.
func Test_dbRowsAffectedError(t *testing.T) {
	tmNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		fn   func(ctx context.Context, h *handler) (int64, error)
	}{
		{"QueuecallSetStatusConnecting", func(ctx context.Context, h *handler) (int64, error) {
			return h.QueuecallSetStatusConnecting(ctx, testQueuecallID, testOtherID, testOtherID)
		}},
		{"QueuecallSetStatusService", func(ctx context.Context, h *handler) (int64, error) {
			return h.QueuecallSetStatusService(ctx, testQueuecallID, 10, &tmNow)
		}},
		{"QueuecallSetStatusAbandoned", func(ctx context.Context, h *handler) (int64, error) {
			return h.QueuecallSetStatusAbandoned(ctx, testQueuecallID, 10, &tmNow)
		}},
		{"QueuecallSetStatusDone", func(ctx context.Context, h *handler) (int64, error) {
			return h.QueuecallSetStatusDone(ctx, testQueuecallID, 10, &tmNow)
		}},
		{"QueuecallSetStatusWaitingIfInitiating", func(ctx context.Context, h *handler) (int64, error) {
			return h.QueuecallSetStatusWaitingIfInitiating(ctx, testQueuecallID)
		}},
		{"QueuecallSetStatusWaitingIfConnecting", func(ctx context.Context, h *handler) (int64, error) {
			return h.QueuecallSetStatusWaitingIfConnecting(ctx, testQueuecallID)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxTestHandler(t)
			mock.ExpectExec("(?i)update queue_queuecalls").WillReturnResult(sqlmock.NewErrorResult(errTestBoom))

			affected, err := tt.fn(context.Background(), h)
			if !errors.Is(err, errTestBoom) {
				t.Errorf("Wrong match. expect: %v, got: %v", errTestBoom, err)
			}
			if affected != 0 {
				t.Errorf("Wrong match. expect: 0, got: %d", affected)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Wrong match. expectations: %v", err)
			}
		})
	}
}

// Test_dbQueryError verifies that a failing read statement is returned to the
// caller without a partial result.
func Test_dbQueryError(t *testing.T) {
	tmNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		expect string
		fn     func(ctx context.Context, h *handler) error
	}{
		{"QueueCountByCustomerID", "(?i)select count", func(ctx context.Context, h *handler) error {
			n, err := h.QueueCountByCustomerID(ctx, testOtherID)
			if n != 0 {
				t.Errorf("Wrong match. expect: 0, got: %d", n)
			}
			return err
		}},
		{"queueGetFromDB", "(?i)select .* from queue_queues", func(ctx context.Context, h *handler) error {
			_, err := h.queueGetFromDB(ctx, testQueueID)
			return err
		}},
		{"QueueList", "(?i)select .* from queue_queues", func(ctx context.Context, h *handler) error {
			res, err := h.QueueList(ctx, 10, "", map[queue.Field]any{})
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			return err
		}},
		{"queuecallGetFromDB", "(?i)select .* from queue_queuecalls", func(ctx context.Context, h *handler) error {
			_, err := h.queuecallGetFromDB(ctx, testQueuecallID)
			return err
		}},
		{"QueuecallList", "(?i)select .* from queue_queuecalls", func(ctx context.Context, h *handler) error {
			res, err := h.QueuecallList(ctx, 10, "", map[queuecall.Field]any{})
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			return err
		}},
		{"QueuecallListOldestWaiting", "(?i)select .* from queue_queuecalls", func(ctx context.Context, h *handler) error {
			res, err := h.QueuecallListOldestWaiting(ctx, testQueueID, 1)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			return err
		}},
		{"QueuecallListConnectingStale", "(?i)select .* from queue_queuecalls", func(ctx context.Context, h *handler) error {
			res, err := h.QueuecallListConnectingStale(ctx, tmNow, 10)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			return err
		}},
		{"QueuecallListWaitingOldest", "(?i)select .* from queue_queuecalls", func(ctx context.Context, h *handler) error {
			res, err := h.QueuecallListWaitingOldest(ctx, 10)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxTestHandler(t)
			mock.ExpectQuery(tt.expect).WillReturnError(errTestBoom)

			err := tt.fn(context.Background(), h)
			if !errors.Is(err, errTestBoom) {
				t.Errorf("Wrong match. expect: %v, got: %v", errTestBoom, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Wrong match. expectations: %v", err)
			}
		})
	}
}

// Test_dbGetCacheMissDBError verifies that a cache miss followed by a failing
// database read returns the database error and does not fill the cache.
func Test_dbGetCacheMissDBError(t *testing.T) {
	ctx := context.Background()

	t.Run("QueueGet", func(t *testing.T) {
		h, mock, mockCache := newCtxTestHandler(t)
		mockCache.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errors.New("cache miss"))
		mock.ExpectQuery("(?i)select .* from queue_queues").WillReturnError(errTestBoom)

		res, err := h.QueueGet(ctx, testQueueID)
		if !errors.Is(err, errTestBoom) || res != nil {
			t.Errorf("Wrong match. expect: nil and %v, got: %v, %v", errTestBoom, res, err)
		}
	})

	t.Run("QueuecallGet", func(t *testing.T) {
		h, mock, mockCache := newCtxTestHandler(t)
		mockCache.EXPECT().QueuecallGet(ctx, testQueuecallID).Return(nil, errors.New("cache miss"))
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnError(errTestBoom)

		res, err := h.QueuecallGet(ctx, testQueuecallID)
		if !errors.Is(err, errTestBoom) || res != nil {
			t.Errorf("Wrong match. expect: nil and %v, got: %v, %v", errTestBoom, res, err)
		}
	})

	t.Run("QueuecallGetByReferenceID", func(t *testing.T) {
		h, mock, mockCache := newCtxTestHandler(t)
		mockCache.EXPECT().QueuecallGetByReferenceID(ctx, testOtherID).Return(nil, errors.New("cache miss"))
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnError(errTestBoom)

		res, err := h.QueuecallGetByReferenceID(ctx, testOtherID)
		if !errors.Is(err, errTestBoom) || res != nil {
			t.Errorf("Wrong match. expect: nil and %v, got: %v, %v", errTestBoom, res, err)
		}
	})
}

// Test_dbGetNotFound verifies that an empty result is reported as ErrNotFound.
func Test_dbGetNotFound(t *testing.T) {
	ctx := context.Background()
	columns := []string{"id"}

	t.Run("QueueGet", func(t *testing.T) {
		h, mock, mockCache := newCtxTestHandler(t)
		mockCache.EXPECT().QueueGet(ctx, testQueueID).Return(nil, errors.New("cache miss"))
		mock.ExpectQuery("(?i)select .* from queue_queues").WillReturnRows(sqlmock.NewRows(columns))

		if res, err := h.QueueGet(ctx, testQueueID); !errors.Is(err, ErrNotFound) || res != nil {
			t.Errorf("Wrong match. expect: nil and ErrNotFound, got: %v, %v", res, err)
		}
	})

	t.Run("QueuecallGet", func(t *testing.T) {
		h, mock, mockCache := newCtxTestHandler(t)
		mockCache.EXPECT().QueuecallGet(ctx, testQueuecallID).Return(nil, errors.New("cache miss"))
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnRows(sqlmock.NewRows(columns))

		if res, err := h.QueuecallGet(ctx, testQueuecallID); !errors.Is(err, ErrNotFound) || res != nil {
			t.Errorf("Wrong match. expect: nil and ErrNotFound, got: %v, %v", res, err)
		}
	})

	t.Run("QueuecallGetByReferenceID", func(t *testing.T) {
		h, mock, mockCache := newCtxTestHandler(t)
		mockCache.EXPECT().QueuecallGetByReferenceID(ctx, testOtherID).Return(nil, errors.New("cache miss"))
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnRows(sqlmock.NewRows(columns))

		if res, err := h.QueuecallGetByReferenceID(ctx, testOtherID); !errors.Is(err, ErrNotFound) || res != nil {
			t.Errorf("Wrong match. expect: nil and ErrNotFound, got: %v, %v", res, err)
		}
	})
}

// Test_dbGetCacheHit verifies that a cache hit is returned without touching the database.
func Test_dbGetCacheHit(t *testing.T) {
	ctx := context.Background()

	t.Run("QueueGet", func(t *testing.T) {
		h, mock, mockCache := newCtxTestHandler(t)
		cached := &queue.Queue{Identity: commonidentity.Identity{ID: testQueueID}, Name: "cached"}
		mockCache.EXPECT().QueueGet(ctx, testQueueID).Return(cached, nil)

		res, err := h.QueueGet(ctx, testQueueID)
		if err != nil || res.Name != "cached" || res.ID != testQueueID {
			t.Errorf("Wrong match. got: %v, %v", res, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Wrong match. expectations: %v", err)
		}
	})

	t.Run("QueuecallGet", func(t *testing.T) {
		h, _, mockCache := newCtxTestHandler(t)
		cached := &queuecall.Queuecall{Identity: commonidentity.Identity{ID: testQueuecallID}, Status: queuecall.StatusWaiting}
		mockCache.EXPECT().QueuecallGet(ctx, testQueuecallID).Return(cached, nil)

		res, err := h.QueuecallGet(ctx, testQueuecallID)
		if err != nil || res.Status != queuecall.StatusWaiting || res.ID != testQueuecallID {
			t.Errorf("Wrong match. got: %v, %v", res, err)
		}
	})

	t.Run("QueuecallGetByReferenceID", func(t *testing.T) {
		h, _, mockCache := newCtxTestHandler(t)
		cached := &queuecall.Queuecall{Identity: commonidentity.Identity{ID: testQueuecallID}, Status: queuecall.StatusService}
		mockCache.EXPECT().QueuecallGetByReferenceID(ctx, testOtherID).Return(cached, nil)

		res, err := h.QueuecallGetByReferenceID(ctx, testOtherID)
		if err != nil || res.Status != queuecall.StatusService || res.ID != testQueuecallID {
			t.Errorf("Wrong match. got: %v, %v", res, err)
		}
	})
}

// Test_dbScanError verifies that a row which cannot be scanned into the model is
// returned as an error instead of a partial value.
func Test_dbScanError(t *testing.T) {
	ctx := context.Background()
	badRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id"}).AddRow([]byte{0x01, 0x02})
	}

	t.Run("queueGetFromDB", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)
		mock.ExpectQuery("(?i)select .* from queue_queues").WillReturnRows(badRows())
		if res, err := h.queueGetFromDB(ctx, testQueueID); err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
		}
	})

	t.Run("QueueList", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)
		mock.ExpectQuery("(?i)select .* from queue_queues").WillReturnRows(badRows())
		if res, err := h.QueueList(ctx, 10, "", map[queue.Field]any{}); err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
		}
	})

	t.Run("queuecallGetFromDB", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnRows(badRows())
		if res, err := h.queuecallGetFromDB(ctx, testQueuecallID); err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
		}
	})

	t.Run("QueuecallGetByReferenceID", func(t *testing.T) {
		h, mock, mockCache := newCtxTestHandler(t)
		mockCache.EXPECT().QueuecallGetByReferenceID(ctx, testOtherID).Return(nil, errors.New("cache miss"))
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnRows(badRows())
		if res, err := h.QueuecallGetByReferenceID(ctx, testOtherID); err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
		}
	})

	t.Run("QueuecallList", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnRows(badRows())
		if res, err := h.QueuecallList(ctx, 10, "", map[queuecall.Field]any{}); err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
		}
	})

	t.Run("QueuecallListOldestWaiting", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnRows(badRows())
		if res, err := h.QueuecallListOldestWaiting(ctx, testQueueID, 1); err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
		}
	})

	t.Run("QueuecallListConnectingStale", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnRows(badRows())
		if res, err := h.QueuecallListConnectingStale(ctx, time.Now(), 10); err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
		}
	})

	t.Run("QueuecallListWaitingOldest", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)
		mock.ExpectQuery("(?i)select .* from queue_queuecalls").WillReturnRows(badRows())
		if res, err := h.QueuecallListWaitingOldest(ctx, 10); err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
		}
	})
}
