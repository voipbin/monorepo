package dbhandler

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/cachehandler"
)

func Test_NewHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)

	res := NewHandler(dbTest, mockCache)

	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler, got: %T", res)
	}
	if h.db != dbTest {
		t.Errorf("Wrong match. the database handle is not kept")
	}
	if h.cache != cachehandler.CacheHandler(mockCache) {
		t.Errorf("Wrong match. the cache handler is not kept")
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. the util handler is not set")
	}
}

// Test_Update_emptyFields verifies that an update without fields is a no-op:
// no statement is sent to the database (sqlmock fails on an unexpected one).
func Test_Update_emptyFields(t *testing.T) {
	ctx := context.Background()

	t.Run("QueueUpdate", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)

		if err := h.QueueUpdate(ctx, testQueueID, map[queue.Field]any{}); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Wrong match. %v", err)
		}
	})

	t.Run("QueuecallUpdate", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)

		if err := h.QueuecallUpdate(ctx, testQueuecallID, map[queuecall.Field]any{}); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Wrong match. %v", err)
		}
	})
}

// Test_queueWrite_cacheRefreshFailure verifies that a failure of the cache
// refresh after a successful write is not reported as a failure of the write.
func Test_queueWrite_cacheRefreshFailure(t *testing.T) {
	tests := []struct {
		name string
		fn   func(ctx context.Context, h *handler, id, queuecallID uuid.UUID) error
	}{
		{"QueueAddWaitQueueCallID", func(ctx context.Context, h *handler, id, queuecallID uuid.UUID) error {
			return h.QueueAddWaitQueueCallID(ctx, id, queuecallID)
		}},
		{"QueueIncreaseTotalServicedCount", func(ctx context.Context, h *handler, id, queuecallID uuid.UUID) error {
			return h.QueueIncreaseTotalServicedCount(ctx, id, queuecallID)
		}},
		{"QueueIncreaseTotalAbandonedCount", func(ctx context.Context, h *handler, id, queuecallID uuid.UUID) error {
			return h.QueueIncreaseTotalAbandonedCount(ctx, id, queuecallID)
		}},
		{"QueueRemoveServiceQueueCall", func(ctx context.Context, h *handler, id, queuecallID uuid.UUID) error {
			return h.QueueRemoveServiceQueueCall(ctx, id, queuecallID)
		}},
		{"QueueRemoveWaitQueueCall", func(ctx context.Context, h *handler, id, queuecallID uuid.UUID) error {
			return h.QueueRemoveWaitQueueCall(ctx, id, queuecallID)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxTestHandler(t)

			mock.ExpectExec("(?i)update queue_queues").WillReturnResult(sqlmock.NewResult(0, 1))
			// the refresh reads the queue back and fails; the cache is never written.
			mock.ExpectQuery("(?i)select .* from queue_queues").WillReturnError(errTestBoom)

			if err := tt.fn(context.Background(), h, testQueueID, testQueuecallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Wrong match. %v", err)
			}
		})
	}
}
