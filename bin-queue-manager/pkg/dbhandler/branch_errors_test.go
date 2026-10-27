package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/cachehandler"
)

var errCacheBranchTest = errors.New("simulated cache failure")

// failingValuer is a filter value that cannot be turned into a query argument.
type failingValuer struct{}

func (failingValuer) Value() (driver.Value, error) { return nil, errCacheBranchTest }

func Test_setToCache_failure(t *testing.T) {
	ctx := context.Background()

	t.Run("queue", func(t *testing.T) {
		h, _, mockCache := newCtxTestHandler(t)
		q := &queue.Queue{Identity: commonidentity.Identity{ID: testQueueID}}
		mockCache.EXPECT().QueueSet(ctx, q).Return(errCacheBranchTest)

		if err := h.queueSetToCache(ctx, q); !errors.Is(err, errCacheBranchTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errCacheBranchTest, err)
		}
	})

	t.Run("queuecall", func(t *testing.T) {
		h, _, mockCache := newCtxTestHandler(t)
		qc := &queuecall.Queuecall{Identity: commonidentity.Identity{ID: testQueuecallID}}
		mockCache.EXPECT().QueuecallSet(ctx, qc).Return(errCacheBranchTest)

		if err := h.queuecallSetToCache(ctx, qc); !errors.Is(err, errCacheBranchTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errCacheBranchTest, err)
		}
	})
}

// Test_updateToCache_cacheFailure verifies that a failing cache write is reported by the
// refresh helpers after the row was read back from the database.
func Test_updateToCache_cacheFailure(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockCache := cachehandler.NewMockCacheHandler(mc)
		h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

		// a fresh id keeps the shared in-memory database free of collisions between runs
		id := uuid.Must(uuid.NewV4())
		t.Cleanup(func() { _, _ = dbTest.Exec("DELETE FROM queue_queues WHERE id = ?", id.Bytes()) })
		mockUtil.EXPECT().TimeNow().Return(&now)
		mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).Return(nil)
		if err := h.QueueCreate(ctx, &queue.Queue{Identity: commonidentity.Identity{ID: id}}); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).Return(errCacheBranchTest)
		if err := h.queueUpdateToCache(ctx, id); !errors.Is(err, errCacheBranchTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errCacheBranchTest, err)
		}
	})

	t.Run("queuecall", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockCache := cachehandler.NewMockCacheHandler(mc)
		h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

		id := uuid.Must(uuid.NewV4())
		t.Cleanup(func() { _, _ = dbTest.Exec("DELETE FROM queue_queuecalls WHERE id = ?", id.Bytes()) })
		mockUtil.EXPECT().TimeNow().Return(&now)
		mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil)
		if err := h.QueuecallCreate(ctx, &queuecall.Queuecall{Identity: commonidentity.Identity{ID: id}}); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(errCacheBranchTest)
		if err := h.queuecallUpdateToCache(ctx, id); !errors.Is(err, errCacheBranchTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errCacheBranchTest, err)
		}
	})
}

// Test_list_unusableFilter verifies that a filter value the query builder cannot
// express is rejected before any statement is sent.
func Test_list_unusableFilter(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		run func(h *handler) error

		expectInErr string
	}{
		{
			name: "queue list, unsupported value type",
			run: func(h *handler) error {
				_, err := h.QueueList(ctx, 10, "2026-01-01 00:00:00.000000", map[queue.Field]any{queue.FieldName: struct{}{}})
				return err
			},
			expectInErr: "could not apply filters",
		},
		{
			name: "queuecall list, unsupported value type",
			run: func(h *handler) error {
				_, err := h.QueuecallList(ctx, 10, "2026-01-01 00:00:00.000000", map[queuecall.Field]any{queuecall.FieldStatus: struct{}{}})
				return err
			},
			expectInErr: "could not apply filters",
		},
		{
			name: "queue list, value that cannot be an argument",
			run: func(h *handler) error {
				_, err := h.QueueList(ctx, 10, "2026-01-01 00:00:00.000000", map[queue.Field]any{queue.FieldName: commondatabasehandler.NotEq{Value: failingValuer{}}})
				return err
			},
			expectInErr: "could not build query",
		},
		{
			name: "queuecall list, value that cannot be an argument",
			run: func(h *handler) error {
				_, err := h.QueuecallList(ctx, 10, "2026-01-01 00:00:00.000000", map[queuecall.Field]any{queuecall.FieldStatus: commondatabasehandler.NotEq{Value: failingValuer{}}})
				return err
			},
			expectInErr: "could not build query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxTestHandler(t)

			err := tt.run(h)
			if err == nil || !strings.Contains(err.Error(), tt.expectInErr) {
				t.Errorf("Wrong match. expect an error containing %q, got: %v", tt.expectInErr, err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("Wrong match. %v", errMet)
			}
		})
	}
}

// Test_update_unencodableValue verifies that a field value that cannot be encoded
// is rejected before any statement is sent.
func Test_update_unencodableValue(t *testing.T) {
	ctx := context.Background()

	t.Run("QueueUpdate", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)

		err := h.QueueUpdate(ctx, testQueueID, map[queue.Field]any{queue.FieldName: []any{make(chan int)}})
		if err == nil || !strings.Contains(err.Error(), "prepare fields failed") {
			t.Errorf("Wrong match. expect a prepare error, got: %v", err)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("Wrong match. %v", errMet)
		}
	})

	t.Run("QueuecallUpdate", func(t *testing.T) {
		h, mock, _ := newCtxTestHandler(t)

		err := h.QueuecallUpdate(ctx, testQueuecallID, map[queuecall.Field]any{queuecall.FieldStatus: []any{make(chan int)}})
		if err == nil || !strings.Contains(err.Error(), "prepare fields failed") {
			t.Errorf("Wrong match. expect a prepare error, got: %v", err)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("Wrong match. %v", errMet)
		}
	})
}
