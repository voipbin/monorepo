package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
)

// seedExpiredCustomer inserts a customer with the given state and sets tm_update
// (and tm_delete when legacy) by raw UPDATE, because CustomerCreate leaves
// tm_update NULL. The times are UTC like the values the handlers write.
func seedExpiredCustomer(t *testing.T, status customer.Status, emailVerified bool, legacy bool, tmUpdate time.Time) uuid.UUID {
	t.Helper()

	id := seedCustomerWith(t, status, emailVerified, legacy)
	if _, err := dbTest.ExecContext(context.Background(), "update customer_customers set tm_update = ? where id = ?", tmUpdate, id.Bytes()); err != nil {
		t.Fatalf("could not set tm_update: %v", err)
	}

	return id
}

func Test_CustomerAnonymizeExpiredUnverified(t *testing.T) {
	anonTime := time.Date(2026, 5, 2, 3, 4, 5, 0, time.UTC)
	oldUpdate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		status        customer.Status
		emailVerified bool
		legacy        bool

		expectAnonymized bool
	}{
		{"expired unverified customer is anonymized", customer.StatusExpired, false, false, true},
		{"expired verified customer is not matched", customer.StatusExpired, true, false, false},
		{"expired legacy customer with tm_delete is not matched", customer.StatusExpired, false, true, false},
		{"initial customer is not matched", customer.StatusInitial, false, false, false},
		{"active customer is not matched", customer.StatusActive, true, false, false},
		{"frozen customer is not matched", customer.StatusFrozen, false, false, false},
		{"deleted customer is not matched", customer.StatusDeleted, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := seedExpiredCustomer(t, tt.status, tt.emailVerified, tt.legacy, oldUpdate)

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(&anonTime)
			if tt.expectAnonymized {
				mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			}

			err := h.CustomerAnonymizeExpiredUnverified(ctx, id, "deleted_user_test", "deleted_test@removed.example")
			if tt.expectAnonymized {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			} else if !errors.Is(err, ErrNotFound) {
				t.Fatalf("Wrong match. expect: ErrNotFound, got: %v", err)
			}

			res, err := h.customerGetFromDB(ctx, id)
			if err != nil {
				t.Fatalf("could not get customer: %v", err)
			}

			if !tt.expectAnonymized {
				if res.Status != tt.status || res.Name == "deleted_user_test" {
					t.Errorf("Wrong match. expect the row untouched, got status: %v, name: %v", res.Status, res.Name)
				}
				return
			}

			if res.Status != customer.StatusDeleted {
				t.Errorf("Wrong match. expect status: %v, got: %v", customer.StatusDeleted, res.Status)
			}
			if res.Name != "deleted_user_test" || res.Email != "deleted_test@removed.example" {
				t.Errorf("Wrong match. expect anonymized name and email, got: %v, %v", res.Name, res.Email)
			}
			if res.PhoneNumber != "" || res.Address != "" || res.WebhookURI != "" {
				t.Errorf("Wrong match. expect phone, address and webhook uri cleared, got: %v, %v, %v", res.PhoneNumber, res.Address, res.WebhookURI)
			}
			if res.TMDelete == nil || !res.TMDelete.Equal(anonTime) {
				t.Errorf("Wrong match. expect tm_delete: %v, got: %v", anonTime, res.TMDelete)
			}
			if res.TMUpdate == nil || !res.TMUpdate.Equal(anonTime) {
				t.Errorf("Wrong match. expect tm_update: %v, got: %v", anonTime, res.TMUpdate)
			}
		})
	}
}

func Test_CustomerAnonymizeExpiredUnverified_secondCallIsNotMatched(t *testing.T) {
	anonTime := time.Date(2026, 5, 2, 3, 4, 5, 0, time.UTC)
	id := seedExpiredCustomer(t, customer.StatusExpired, false, false, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	mockUtil.EXPECT().TimeNow().Return(&anonTime).Times(2)
	mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil).Times(1)

	if err := h.CustomerAnonymizeExpiredUnverified(ctx, id, "a", "a@removed.example"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := h.CustomerAnonymizeExpiredUnverified(ctx, id, "a", "a@removed.example"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func Test_CustomerListExpiredUnverified(t *testing.T) {
	cutoff := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()

	before := seedExpiredCustomer(t, customer.StatusExpired, false, false, cutoff.Add(-time.Second))
	older := seedExpiredCustomer(t, customer.StatusExpired, false, false, cutoff.Add(-time.Hour))
	oldest := seedExpiredCustomer(t, customer.StatusExpired, false, false, cutoff.Add(-24*time.Hour))
	equal := seedExpiredCustomer(t, customer.StatusExpired, false, false, cutoff)
	after := seedExpiredCustomer(t, customer.StatusExpired, false, false, cutoff.Add(time.Second))
	verified := seedExpiredCustomer(t, customer.StatusExpired, true, false, cutoff.Add(-time.Hour))
	legacy := seedExpiredCustomer(t, customer.StatusExpired, false, true, cutoff.Add(-time.Hour))
	initial := seedExpiredCustomer(t, customer.StatusInitial, false, false, cutoff.Add(-time.Hour))
	active := seedExpiredCustomer(t, customer.StatusActive, true, false, cutoff.Add(-time.Hour))
	frozen := seedExpiredCustomer(t, customer.StatusFrozen, false, false, cutoff.Add(-time.Hour))
	noUpdate := seedCustomerWith(t, customer.StatusExpired, false, false) // tm_update stays NULL

	mc := gomock.NewController(t)
	defer mc.Finish()
	h := handler{utilHandler: utilhandler.NewMockUtilHandler(mc), db: dbTest, cache: cachehandler.NewMockCacheHandler(mc)}

	// the sqlite database is shared between tests, so check membership and the
	// relative order of the rows created here instead of the whole list.
	res, err := h.CustomerListExpiredUnverified(ctx, cutoff, 1000)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	position := map[uuid.UUID]int{}
	for i, c := range res {
		position[c.ID] = i
		if c.Status != customer.StatusExpired || c.EmailVerified || c.TMDelete != nil {
			t.Errorf("Wrong match. expect only expired unverified rows without tm_delete, got: %v, %v, %v", c.Status, c.EmailVerified, c.TMDelete)
		}
	}

	expect := map[string]struct {
		id      uuid.UUID
		present bool
	}{
		"one second before the cutoff is listed": {before, true},
		"one hour before the cutoff is listed":   {older, true},
		"one day before the cutoff is listed":    {oldest, true},
		"exactly at the cutoff is kept":          {equal, false},
		"after the cutoff is kept":               {after, false},
		"verified customer is skipped":           {verified, false},
		"legacy customer with tm_delete skipped": {legacy, false},
		"initial customer is skipped":            {initial, false},
		"active customer is skipped":             {active, false},
		"frozen customer is skipped":             {frozen, false},
		"row without tm_update is skipped":       {noUpdate, false},
	}
	for name, e := range expect {
		if _, ok := position[e.id]; ok != e.present {
			t.Errorf("Wrong match. %s: expect present: %v, got: %v", name, e.present, ok)
		}
	}

	if position[oldest] >= position[older] || position[older] >= position[before] {
		t.Errorf("Wrong match. expect ascending tm_update order, got positions: %d, %d, %d", position[oldest], position[older], position[before])
	}
}

func Test_CustomerListExpiredUnverified_limit(t *testing.T) {
	// older than every other seeded row in the shared database, so these rows
	// are the first ones in the ascending order.
	base := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()

	first := seedExpiredCustomer(t, customer.StatusExpired, false, false, base)
	second := seedExpiredCustomer(t, customer.StatusExpired, false, false, base.Add(time.Second))
	third := seedExpiredCustomer(t, customer.StatusExpired, false, false, base.Add(2*time.Second))

	mc := gomock.NewController(t)
	defer mc.Finish()
	h := handler{utilHandler: utilhandler.NewMockUtilHandler(mc), db: dbTest, cache: cachehandler.NewMockCacheHandler(mc)}

	res, err := h.CustomerListExpiredUnverified(ctx, base.Add(24*time.Hour), 2)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("Wrong match. expect 2 rows, got: %d", len(res))
	}
	if res[0].ID != first || res[1].ID != second {
		t.Errorf("Wrong match. expect the two oldest rows in order, got: %v, %v", res[0].ID, res[1].ID)
	}
	for _, c := range res {
		if c.ID == third {
			t.Errorf("Wrong match. expect the third row to be cut off by the limit")
		}
	}
}
