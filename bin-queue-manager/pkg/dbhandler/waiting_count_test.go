package dbhandler

import (
	"context"
	"database/sql"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/cachehandler"
)

// Test_QueuecallSetStatusWaitingIfInitiating verifies the compare-and-swap
// against a real (sqlite) row: only an initiating queuecall moves to waiting;
// a queuecall that was already abandoned must not be resurrected, and the row
// of a lost swap stays exactly as it was.
func Test_QueuecallSetStatusWaitingIfInitiating(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, context.DeadlineExceeded).AnyTimes()

	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	create := func(t *testing.T, id uuid.UUID, status queuecall.Status) {
		t.Helper()
		tmCreate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		mockUtil.EXPECT().TimeNow().Return(&tmCreate)
		if err := h.QueuecallCreate(ctx, &queuecall.Queuecall{
			Identity: commonidentity.Identity{ID: id},
			Status:   status,
		}); err != nil {
			t.Fatalf("Could not create the queuecall for setup. err: %v", err)
		}
	}

	tests := []struct {
		name string

		id           uuid.UUID
		startStatus  queuecall.Status
		expectAffect int64
		expectStatus queuecall.Status
	}{
		{
			name:         "initiating moves to waiting",
			id:           uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000001"),
			startStatus:  queuecall.StatusInitiating,
			expectAffect: 1,
			expectStatus: queuecall.StatusWaiting,
		},
		{
			name:         "abandoned stays abandoned",
			id:           uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000002"),
			startStatus:  queuecall.StatusAbandoned,
			expectAffect: 0,
			expectStatus: queuecall.StatusAbandoned,
		},
		{
			name:         "waiting is not swapped again",
			id:           uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000003"),
			startStatus:  queuecall.StatusWaiting,
			expectAffect: 0,
			expectStatus: queuecall.StatusWaiting,
		},
		{
			name:         "service is untouched",
			id:           uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000004"),
			startStatus:  queuecall.StatusService,
			expectAffect: 0,
			expectStatus: queuecall.StatusService,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			create(t, tt.id, tt.startStatus)

			now := time.Date(2025, 6, 1, 0, 5, 0, 0, time.UTC)
			mockUtil.EXPECT().TimeNow().Return(&now)

			affected, err := h.QueuecallSetStatusWaitingIfInitiating(ctx, tt.id)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if affected != tt.expectAffect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectAffect, affected)
			}

			res, err := h.QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Fatalf("Could not get the queuecall. err: %v", err)
			}
			if res.Status != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, res.Status)
			}
			if tt.expectAffect == 0 && res.TMUpdate != nil {
				t.Errorf("Wrong match. a lost swap must not stamp tm_update. got: %v", res.TMUpdate)
			}
		})
	}

	t.Run("unknown id affects nothing", func(t *testing.T) {
		now := time.Date(2025, 6, 1, 0, 5, 0, 0, time.UTC)
		mockUtil.EXPECT().TimeNow().Return(&now)

		affected, err := h.QueuecallSetStatusWaitingIfInitiating(ctx, uuid.FromStringOrNil("e1000000-0000-0000-0000-0000000000ff"))
		if err != nil || affected != 0 {
			t.Errorf("Wrong match. expect: (0, nil), got: (%d, %v)", affected, err)
		}
	})
}

func Test_QueuecallSetStatusWaitingIfInitiating_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	closedDB, err := sql.Open("sqlite3", `file:closed_waiting_if?mode=memory&cache=private`)
	if err != nil {
		t.Fatalf("Could not open the db. err: %v", err)
	}
	_ = closedDB.Close()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	now := time.Date(2025, 6, 1, 0, 5, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now)

	h := &handler{
		utilHandler: mockUtil,
		db:          closedDB,
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	affected, err := h.QueuecallSetStatusWaitingIfInitiating(context.Background(), uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000001"))
	if err == nil || affected != 0 {
		t.Errorf("Wrong match. expect: (0, error), got: (%d, %v)", affected, err)
	}
}

func Test_QueueCountByCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().QueueGet(gomock.Any(), gomock.Any()).Return(nil, context.DeadlineExceeded).AnyTimes()

	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	customerA := uuid.FromStringOrNil("c1000000-0000-0000-0000-00000000000a")
	customerB := uuid.FromStringOrNil("c1000000-0000-0000-0000-00000000000b")
	customerEmpty := uuid.FromStringOrNil("c1000000-0000-0000-0000-00000000000c")

	create := func(id string, customerID uuid.UUID) uuid.UUID {
		t.Helper()
		qid := uuid.FromStringOrNil(id)
		tm := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
		mockUtil.EXPECT().TimeNow().Return(&tm)
		if err := h.QueueCreate(ctx, &queue.Queue{
			Identity:      commonidentity.Identity{ID: qid, CustomerID: customerID},
			RoutingMethod: queue.RoutingMethodRandom,
		}); err != nil {
			t.Fatalf("Could not create the queue for setup. err: %v", err)
		}
		return qid
	}

	create("d1000000-0000-0000-0000-000000000001", customerA)
	create("d1000000-0000-0000-0000-000000000002", customerA)
	deleted := create("d1000000-0000-0000-0000-000000000003", customerA)
	create("d1000000-0000-0000-0000-000000000004", customerB)

	tm := time.Date(2025, 6, 2, 1, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&tm)
	if err := h.QueueDelete(ctx, deleted); err != nil {
		t.Fatalf("Could not delete the queue for setup. err: %v", err)
	}

	tests := []struct {
		name       string
		customerID uuid.UUID
		expect     int
	}{
		{"deleted queues are not counted", customerA, 2},
		{"other customers' queues are not counted", customerB, 1},
		{"customer without queues", customerEmpty, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := h.QueueCountByCustomerID(ctx, tt.customerID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if count != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expect, count)
			}
		})
	}

	t.Run("query failure is returned", func(t *testing.T) {
		closedDB, err := sql.Open("sqlite3", `file:closed_count?mode=memory&cache=private`)
		if err != nil {
			t.Fatalf("Could not open the db. err: %v", err)
		}
		_ = closedDB.Close()

		hc := &handler{utilHandler: mockUtil, db: closedDB, cache: mockCache}
		count, err := hc.QueueCountByCustomerID(ctx, customerA)
		if err == nil || count != 0 {
			t.Errorf("Wrong match. expect: (0, error), got: (%d, %v)", count, err)
		}
	})
}
