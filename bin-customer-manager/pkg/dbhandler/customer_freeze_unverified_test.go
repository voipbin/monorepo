package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
)

func Test_CustomerFreeze_UnverifiedNotMatched(t *testing.T) {
	scheduled := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		status customer.Status
	}{
		{"initial customer is not matched", customer.StatusInitial},
		{"expired customer is not matched", customer.StatusExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := seedCustomerWith(t, tt.status, false, false)

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(&scheduled)

			if err := h.CustomerFreeze(ctx, id); !errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
			}
		})
	}
}

func Test_CustomerFreezeAllowUnverified(t *testing.T) {
	scheduled := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		initialStatus customer.Status
		emailVerified bool
		legacy        bool

		expectErr    bool
		expectStatus customer.Status
	}{
		{"active customer is frozen", customer.StatusActive, true, false, false, customer.StatusFrozen},
		{"initial customer is frozen", customer.StatusInitial, false, false, false, customer.StatusFrozen},
		{"expired customer is frozen", customer.StatusExpired, false, false, false, customer.StatusFrozen},
		{"expired legacy customer with tm_delete is not matched", customer.StatusExpired, false, true, true, customer.StatusExpired},
		{"initial customer with tm_delete is not matched", customer.StatusInitial, false, true, true, customer.StatusInitial},
		{"already frozen customer is not matched", customer.StatusFrozen, true, false, true, customer.StatusFrozen},
		{"deleted customer is not matched", customer.StatusDeleted, true, false, true, customer.StatusDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := seedCustomerWith(t, tt.initialStatus, tt.emailVerified, tt.legacy)

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

			err := h.CustomerFreezeAllowUnverified(ctx, id)
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
			} else if res.TMDeletionScheduled == nil || !res.TMDeletionScheduled.Equal(scheduled) {
				t.Errorf("Wrong match. expect deletion schedule: %v, got: %v", scheduled, res.TMDeletionScheduled)
			}
		})
	}
}

func Test_CustomerRecover_UnverifiedNotMatched(t *testing.T) {
	recoverTime := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	id := seedCustomerWith(t, customer.StatusFrozen, false, false)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	mockUtil.EXPECT().TimeNow().Return(&recoverTime)

	if err := h.CustomerRecover(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}

	res, err := h.customerGetFromDB(ctx, id)
	if err != nil {
		t.Fatalf("could not get customer: %v", err)
	}
	if res.Status != customer.StatusFrozen {
		t.Errorf("Wrong match. expect status: %v, got: %v", customer.StatusFrozen, res.Status)
	}
}
