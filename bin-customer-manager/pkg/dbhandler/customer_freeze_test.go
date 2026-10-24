package dbhandler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
)

// seedCustomer inserts a customer with the given status through a lenient
// handler so the handler under test can run with strict (expectation-less) mocks.
func seedCustomer(t *testing.T, status customer.Status) uuid.UUID {
	t.Helper()

	mc := gomock.NewController(t)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	curTime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()
	mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

	id, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("could not generate uuid: %v", err)
	}
	if err := h.CustomerCreate(context.Background(), &customer.Customer{
		ID:     id,
		Name:   "freeze test",
		Email:  "freeze@example.com",
		Status: status,
	}); err != nil {
		t.Fatalf("could not create customer: %v", err)
	}

	return id
}

func Test_CustomerFreeze(t *testing.T) {
	scheduled := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		initialStatus customer.Status

		expectErr    bool
		expectStatus customer.Status
	}{
		{"active customer is frozen", customer.StatusActive, false, customer.StatusFrozen},
		{"already frozen customer is not matched", customer.StatusFrozen, true, customer.StatusFrozen},
		{"deleted customer is not matched", customer.StatusDeleted, true, customer.StatusDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := seedCustomer(t, tt.initialStatus)

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(&scheduled)
			if !tt.expectErr {
				mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			}

			err := h.CustomerFreeze(ctx, id)
			if tt.expectErr {
				if !errors.Is(err, ErrNotFound) {
					t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
				}
			} else if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.customerGetFromDB(ctx, id)
			if err != nil {
				t.Fatalf("could not get customer: %v", err)
			}
			if res.Status != tt.expectStatus {
				t.Errorf("Wrong match. expect status: %v, got: %v", tt.expectStatus, res.Status)
			}
			if tt.expectErr {
				if res.TMDeletionScheduled != nil {
					t.Errorf("Wrong match. expect deletion schedule untouched, got: %v", res.TMDeletionScheduled)
				}
			} else {
				if res.TMDeletionScheduled == nil || !res.TMDeletionScheduled.Equal(scheduled) {
					t.Errorf("Wrong match. expect deletion schedule: %v, got: %v", scheduled, res.TMDeletionScheduled)
				}
				if res.TMUpdate == nil || !res.TMUpdate.Equal(scheduled) {
					t.Errorf("Wrong match. expect tm_update: %v, got: %v", scheduled, res.TMUpdate)
				}
			}
		})
	}
}

func Test_CustomerFreeze_notExist(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: cachehandler.NewMockCacheHandler(mc)}

	curTime := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&curTime)

	err := h.CustomerFreeze(context.Background(), uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000aa"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

// Test_CustomerFreeze_Concurrent calls CustomerFreeze for one active customer
// from many goroutines. The sqlite harness has a single connection, so the
// statements are serialized; what this proves is that the status guard in the
// WHERE clause lets exactly one call win and every other call gets ErrNotFound.
func Test_CustomerFreeze_Concurrent(t *testing.T) {
	id := seedCustomer(t, customer.StatusActive)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	scheduled := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&scheduled).Times(20)
	// the cache is refreshed only by the winner
	mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	const n = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	winners, losers := 0, 0
	var unexpected []error

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			callErr := h.CustomerFreeze(ctx, id)

			mu.Lock()
			defer mu.Unlock()
			switch {
			case callErr == nil:
				winners++
			case errors.Is(callErr, ErrNotFound):
				losers++
			default:
				unexpected = append(unexpected, callErr)
			}
		}()
	}
	wg.Wait()

	if len(unexpected) != 0 {
		t.Fatalf("Expected no unexpected errors, got: %v", unexpected)
	}
	if winners != 1 || losers != n-1 {
		t.Errorf("Wrong match. expect: 1 winner and %d losers, got: %d winners and %d losers", n-1, winners, losers)
	}
}

func Test_CustomerRecover(t *testing.T) {
	freezeTime := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	recoverTime := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		initialStatus customer.Status

		expectErr    bool
		expectStatus customer.Status
	}{
		{"frozen customer is recovered", customer.StatusFrozen, false, customer.StatusActive},
		{"active customer is not matched", customer.StatusActive, true, customer.StatusActive},
		{"deleted customer is not matched", customer.StatusDeleted, true, customer.StatusDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := seedCustomer(t, tt.initialStatus)
			ctx := context.Background()

			// give the row a deletion schedule so that recovery has something to clear
			if _, err := dbTest.ExecContext(ctx, "update customer_customers set tm_deletion_scheduled = ? where id = ?", freezeTime, id.Bytes()); err != nil {
				t.Fatalf("could not set the deletion schedule: %v", err)
			}

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

			mockUtil.EXPECT().TimeNow().Return(&recoverTime)
			if !tt.expectErr {
				mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			}

			err := h.CustomerRecover(ctx, id)
			if tt.expectErr {
				if !errors.Is(err, ErrNotFound) {
					t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
				}
			} else if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.customerGetFromDB(ctx, id)
			if err != nil {
				t.Fatalf("could not get customer: %v", err)
			}
			if res.Status != tt.expectStatus {
				t.Errorf("Wrong match. expect status: %v, got: %v", tt.expectStatus, res.Status)
			}
			if tt.expectErr {
				if res.TMDeletionScheduled == nil {
					t.Errorf("Wrong match. expect deletion schedule untouched, got: nil")
				}
			} else {
				if res.TMDeletionScheduled != nil {
					t.Errorf("Wrong match. expect deletion schedule cleared, got: %v", res.TMDeletionScheduled)
				}
				if res.TMUpdate == nil || !res.TMUpdate.Equal(recoverTime) {
					t.Errorf("Wrong match. expect tm_update: %v, got: %v", recoverTime, res.TMUpdate)
				}
			}
		})
	}
}

func Test_CustomerListFrozenExpired(t *testing.T) {
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()

	freezeAt := func(at time.Time) uuid.UUID {
		id := seedCustomer(t, customer.StatusActive)

		mc := gomock.NewController(t)
		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockCache := cachehandler.NewMockCacheHandler(mc)
		mockUtil.EXPECT().TimeNow().Return(&at).AnyTimes()
		mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
		if err := h.CustomerFreeze(ctx, id); err != nil {
			t.Fatalf("could not freeze customer: %v", err)
		}
		return id
	}

	before := freezeAt(base.Add(-time.Second))
	equal := freezeAt(base)
	after := freezeAt(base.Add(time.Second))
	soft := freezeAt(base.Add(-time.Hour))
	if _, err := dbTest.ExecContext(ctx, "update customer_customers set tm_delete = ? where id = ?", base, soft.Bytes()); err != nil {
		t.Fatalf("could not soft delete: %v", err)
	}
	active := seedCustomer(t, customer.StatusActive)
	if _, err := dbTest.ExecContext(ctx, "update customer_customers set tm_deletion_scheduled = ? where id = ?", base.Add(-time.Hour), active.Bytes()); err != nil {
		t.Fatalf("could not set the deletion schedule: %v", err)
	}

	mc := gomock.NewController(t)
	defer mc.Finish()
	h := handler{utilHandler: utilhandler.NewMockUtilHandler(mc), db: dbTest, cache: cachehandler.NewMockCacheHandler(mc)}

	res, err := h.CustomerListFrozenExpired(ctx, base)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the sqlite database is shared between tests, so check membership of the
	// rows created here instead of comparing the whole list.
	got := map[uuid.UUID]bool{}
	for _, c := range res {
		got[c.ID] = true
		if c.Status != customer.StatusFrozen {
			t.Errorf("Wrong match. expect only frozen customers, got: %v", c.Status)
		}
	}

	expect := map[string]struct {
		id      uuid.UUID
		present bool
	}{
		"scheduled before the cutoff is listed":   {before, true},
		"scheduled exactly at the cutoff is kept": {equal, false},
		"scheduled after the cutoff is kept":      {after, false},
		"soft deleted customer is skipped":        {soft, false},
		"non frozen customer is skipped":          {active, false},
	}
	for name, e := range expect {
		if got[e.id] != e.present {
			t.Errorf("Wrong match. %s: expect listed: %v, got: %v", name, e.present, got[e.id])
		}
	}
}
