package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

// Test_rowScan_failure verifies that a row the model cannot be scanned from is reported with the
// name of the failing read, is not mistaken for a missing record, and never refreshes the cache.
func Test_rowScan_failure(t *testing.T) {
	id := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000001")

	tests := []struct {
		name      string
		fn        func(h *handler) error
		expectMsg string
	}{
		{"account get", func(h *handler) error { _, err := h.accountGetFromDB(context.Background(), id); return err }, "accountGetFromDB: could not scan row"},
		{"billing get", func(h *handler) error { _, err := h.billingGetFromDB(context.Background(), id); return err }, "billingGetFromDB: could not scan row"},
		{"billing get by reference id", func(h *handler) error {
			_, err := h.billingGetByReferenceIDFromDB(context.Background(), id)
			return err
		}, "billingGetByReferenceIDFromDB: could not scan row"},
		{"billing get by reference type and id", func(h *handler) error {
			_, err := h.billingGetByReferenceTypeAndIDFromDB(context.Background(), billing.ReferenceTypeCall, id)
			return err
		}, "billingGetByReferenceTypeAndIDFromDB: could not scan row"},
		{"billing get by idempotency key", func(h *handler) error {
			_, err := h.BillingGetByIdempotencyKey(context.Background(), id)
			return err
		}, "could not scan row"},
		{"account list", func(h *handler) error { _, err := h.AccountList(context.Background(), 10, "", nil); return err }, "AccountList: could not scan row"},
		{"billing list", func(h *handler) error { _, err := h.BillingList(context.Background(), 10, "", nil); return err }, "BillingList: could not scan row"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockUtil, _ := newFailureHandler(t)
			mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

			// a single column cannot be scanned into the model's destination list
			mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()))

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("a scan failure must not be reported as ErrNotFound. err: %v", err)
			}
			if !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("wrong error. expect to contain: %q, got: %v", tt.expectMsg, err)
			}
		})
	}
}

// Test_emptyResult_notFound verifies that a query without rows is reported as ErrNotFound.
func Test_emptyResult_notFound(t *testing.T) {
	id := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000002")

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{"account get", func(h *handler) error { _, err := h.accountGetFromDB(context.Background(), id); return err }},
		{"billing get", func(h *handler) error { _, err := h.billingGetFromDB(context.Background(), id); return err }},
		{"billing get by reference id", func(h *handler) error {
			_, err := h.billingGetByReferenceIDFromDB(context.Background(), id)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _, _ := newFailureHandler(t)
			mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))

			err := tt.fn(h)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("wrong error. expect: ErrNotFound, got: %v", err)
			}
		})
	}
}

// Test_queryError_returned verifies that a failing select is returned with its cause.
func Test_queryError_returned(t *testing.T) {
	id := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000003")

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{"billing get by reference id", func(h *handler) error {
			_, err := h.billingGetByReferenceIDFromDB(context.Background(), id)
			return err
		}},
		{"billing get by reference type and id", func(h *handler) error {
			_, err := h.billingGetByReferenceTypeAndIDFromDB(context.Background(), billing.ReferenceTypeCall, id)
			return err
		}},
		{"billing get by idempotency key", func(h *handler) error {
			_, err := h.BillingGetByIdempotencyKey(context.Background(), id)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _, _ := newFailureHandler(t)
			mock.ExpectQuery("SELECT").WillReturnError(errCause)

			err := tt.fn(h)
			if !errors.Is(err, errCause) {
				t.Errorf("wrong error. expect the injected cause, got: %v", err)
			}
		})
	}
}

// Test_cacheHit_skipsDatabase verifies that a cached record is returned without a database read.
func Test_cacheHit_skipsDatabase(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000004")
	referenceID := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000005")

	t.Run("account get", func(t *testing.T) {
		h, mock, _, mockCache := newFailureHandler(t)
		cached := &account.Account{Identity: commonidentity.Identity{ID: id}, Name: "cached account"}
		mockCache.EXPECT().AccountGet(ctx, id).Return(cached, nil)

		res, err := h.AccountGet(ctx, id)
		if err != nil {
			t.Fatalf("expected ok, got: %v", err)
		}
		if res.Name != "cached account" || res.ID != id {
			t.Errorf("wrong result. got: %+v", res)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unexpected database use: %v", errMet)
		}
	})

	t.Run("billing get", func(t *testing.T) {
		h, mock, _, mockCache := newFailureHandler(t)
		cached := &billing.Billing{Identity: commonidentity.Identity{ID: id}, ReferenceID: referenceID}
		mockCache.EXPECT().BillingGet(ctx, id).Return(cached, nil)

		res, err := h.BillingGet(ctx, id)
		if err != nil {
			t.Fatalf("expected ok, got: %v", err)
		}
		if res.ID != id || res.ReferenceID != referenceID {
			t.Errorf("wrong result. got: %+v", res)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unexpected database use: %v", errMet)
		}
	})

	t.Run("billing get by reference id", func(t *testing.T) {
		h, mock, _, mockCache := newFailureHandler(t)
		cached := &billing.Billing{Identity: commonidentity.Identity{ID: id}, ReferenceID: referenceID}
		mockCache.EXPECT().BillingGetByReferenceID(ctx, referenceID).Return(cached, nil)

		res, err := h.BillingGetByReferenceID(ctx, referenceID)
		if err != nil {
			t.Fatalf("expected ok, got: %v", err)
		}
		if res.ID != id || res.ReferenceID != referenceID {
			t.Errorf("wrong result. got: %+v", res)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unexpected database use: %v", errMet)
		}
	})
}

// Test_cacheMiss_databaseError verifies that a database failure after a cache miss is returned.
func Test_cacheMiss_databaseError(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000006")

	t.Run("account get", func(t *testing.T) {
		h, mock, _, mockCache := newFailureHandler(t)
		mockCache.EXPECT().AccountGet(ctx, id).Return(nil, errors.New("cache miss"))
		mock.ExpectQuery("SELECT").WillReturnError(errCause)

		if _, err := h.AccountGet(ctx, id); !errors.Is(err, errCause) {
			t.Errorf("wrong error. expect the injected cause, got: %v", err)
		}
	})

	t.Run("billing get by reference id", func(t *testing.T) {
		h, mock, _, mockCache := newFailureHandler(t)
		mockCache.EXPECT().BillingGetByReferenceID(ctx, id).Return(nil, errors.New("cache miss"))
		mock.ExpectQuery("SELECT").WillReturnError(errCause)

		if _, err := h.BillingGetByReferenceID(ctx, id); !errors.Is(err, errCause) {
			t.Errorf("wrong error. expect the injected cause, got: %v", err)
		}
	})
}

// Test_cacheRefresh_failure verifies that a failing cache write is returned by the cache refresh
// helpers, both when the record is passed in and when it is read back from the database first.
func Test_cacheRefresh_failure(t *testing.T) {
	ctx := context.Background()
	tmNow := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	// the in-memory database is shared by the whole package run, so the seeded rows get fresh ids
	accountID := uuid.Must(uuid.NewV4())
	billingID := uuid.Must(uuid.NewV4())

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

	// seed one account and one billing record; the cache refresh after the insert is accepted
	mockUtil.EXPECT().TimeNow().Return(&tmNow).Times(2)
	mockCache.EXPECT().AccountSet(ctx, gomock.Any()).Return(nil)
	if err := h.AccountCreate(ctx, &account.Account{Identity: commonidentity.Identity{ID: accountID}}); err != nil {
		t.Fatalf("could not seed the account: %v", err)
	}
	mockCache.EXPECT().BillingSet(ctx, gomock.Any()).Return(nil)
	if err := h.BillingCreate(ctx, &billing.Billing{
		Identity:       commonidentity.Identity{ID: billingID},
		IdempotencyKey: billingID,
		ReferenceType:  billing.ReferenceTypeCall,
		ReferenceID:    billingID,
	}); err != nil {
		t.Fatalf("could not seed the billing: %v", err)
	}

	t.Run("account update to cache", func(t *testing.T) {
		mockCache.EXPECT().AccountSet(ctx, gomock.Any()).Return(errCause)
		if err := h.accountUpdateToCache(ctx, accountID); !errors.Is(err, errCause) {
			t.Errorf("wrong error. expect the injected cause, got: %v", err)
		}
	})

	t.Run("account set to cache", func(t *testing.T) {
		mockCache.EXPECT().AccountSet(ctx, gomock.Any()).Return(errCause)
		if err := h.accountSetToCache(ctx, &account.Account{}); !errors.Is(err, errCause) {
			t.Errorf("wrong error. expect the injected cause, got: %v", err)
		}
	})

	t.Run("billing update to cache", func(t *testing.T) {
		mockCache.EXPECT().BillingSet(ctx, gomock.Any()).Return(errCause)
		if err := h.billingUpdateToCache(ctx, billingID); !errors.Is(err, errCause) {
			t.Errorf("wrong error. expect the injected cause, got: %v", err)
		}
	})

	t.Run("billing set to cache", func(t *testing.T) {
		mockCache.EXPECT().BillingSet(ctx, gomock.Any()).Return(errCause)
		if err := h.billingSetToCache(ctx, &billing.Billing{}); !errors.Is(err, errCause) {
			t.Errorf("wrong error. expect the injected cause, got: %v", err)
		}
	})
}

// Test_AccountSetStatus_stored verifies that the status is written and then read back for the cache.
func Test_AccountSetStatus_stored(t *testing.T) {
	ctx := context.Background()
	id := uuid.Must(uuid.NewV4())
	tmCreate := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	tmUpdate := time.Date(2026, 3, 4, 6, 6, 7, 0, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

	mockUtil.EXPECT().TimeNow().Return(&tmCreate)
	mockCache.EXPECT().AccountSet(ctx, gomock.Any()).Return(nil)
	if err := h.AccountCreate(ctx, &account.Account{Identity: commonidentity.Identity{ID: id}, Status: account.StatusActive}); err != nil {
		t.Fatalf("could not seed the account: %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&tmUpdate)
	mockCache.EXPECT().AccountSet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, a *account.Account) error {
		if a.ID != id || a.Status != account.StatusFrozen {
			t.Errorf("the cache must receive the updated account. got: %+v", a)
		}
		return nil
	})
	if err := h.AccountSetStatus(ctx, id, account.StatusFrozen); err != nil {
		t.Fatalf("expected ok, got: %v", err)
	}

	res, err := h.accountGetFromDB(ctx, id)
	if err != nil {
		t.Fatalf("could not read the account back: %v", err)
	}
	if res.Status != account.StatusFrozen {
		t.Errorf("wrong status. expect: %s, got: %s", account.StatusFrozen, res.Status)
	}
	if res.TMUpdate == nil || !res.TMUpdate.Equal(tmUpdate) {
		t.Errorf("wrong update time. expect: %v, got: %v", tmUpdate, res.TMUpdate)
	}
}

// Test_CalculateTokenCreditDeduction_unknownMode verifies that an unsupported cost mode deducts nothing.
func Test_CalculateTokenCreditDeduction_unknownMode(t *testing.T) {
	res := CalculateTokenCreditDeduction(1000, 5, billing.CostInfo{
		Mode:          billing.CostMode(99),
		TokenPerUnit:  10,
		CreditPerUnit: 20,
	})
	if res.TokenDeducted != 0 || res.CreditDeducted != 0 {
		t.Errorf("an unsupported mode must deduct nothing. got: %+v", res)
	}
}
