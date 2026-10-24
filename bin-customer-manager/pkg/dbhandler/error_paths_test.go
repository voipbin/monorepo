package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
)

// newClosedDBHandler returns a handler whose database connection is already
// closed. It uses a private in-memory database, not the shared test database.
func newClosedDBHandler(t *testing.T, cache cachehandler.CacheHandler) *handler {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open the private database. err: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("could not close the private database. err: %v", err)
	}

	return &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
}

func Test_closedDatabase(t *testing.T) {
	customerID := uuid.FromStringOrNil("3f6a9b10-0000-4000-8000-000000000001")
	accesskeyID := uuid.FromStringOrNil("3f6a9b10-0000-4000-8000-000000000002")
	curTime := time.Date(2024, 4, 18, 3, 22, 17, 0, time.UTC)

	tests := []struct {
		name         string
		cacheMissFor string // "customer" or "accesskey" when the call reads the cache first
		expectPhrase string
		call         func(h *handler) error
	}{
		{
			name:         "customer create",
			expectPhrase: "could not execute query. CustomerCreate",
			call: func(h *handler) error {
				return h.CustomerCreate(context.Background(), &customer.Customer{})
			},
		},
		{
			name:         "customer get",
			cacheMissFor: "customer",
			expectPhrase: "could not query. customerGetFromDB",
			call: func(h *handler) error {
				_, err := h.CustomerGet(context.Background(), customerID)
				return err
			},
		},
		{
			name:         "customer list",
			expectPhrase: "could not query. CustomerGets",
			call: func(h *handler) error {
				_, err := h.CustomerList(context.Background(), 10, "", map[customer.Field]any{})
				return err
			},
		},
		{
			name:         "customer update",
			expectPhrase: "CustomerUpdate: exec failed",
			call: func(h *handler) error {
				return h.CustomerUpdate(context.Background(), customerID, map[customer.Field]any{customer.FieldName: "name"})
			},
		},
		{
			name:         "customer freeze",
			expectPhrase: "CustomerFreeze",
			call: func(h *handler) error {
				return h.CustomerFreeze(context.Background(), customerID)
			},
		},
		{
			name:         "customer recover",
			expectPhrase: "CustomerRecover",
			call: func(h *handler) error {
				return h.CustomerRecover(context.Background(), customerID)
			},
		},
		{
			name:         "customer list frozen expired",
			expectPhrase: "CustomerListFrozenExpired",
			call: func(h *handler) error {
				_, err := h.CustomerListFrozenExpired(context.Background(), curTime)
				return err
			},
		},
		{
			name:         "customer anonymize",
			expectPhrase: "CustomerAnonymizePII",
			call: func(h *handler) error {
				return h.CustomerAnonymizePII(context.Background(), customerID, "anon", "anon@example.com")
			},
		},
		{
			name:         "customer delete",
			expectPhrase: "CustomerDelete: exec failed",
			call: func(h *handler) error {
				return h.CustomerDelete(context.Background(), customerID)
			},
		},
		{
			name:         "customer hard delete",
			expectPhrase: "CustomerHardDelete: exec failed",
			call: func(h *handler) error {
				return h.CustomerHardDelete(context.Background(), customerID)
			},
		},
		{
			name:         "accesskey create",
			expectPhrase: "AccesskeyCreate",
			call: func(h *handler) error {
				return h.AccesskeyCreate(context.Background(), &accesskey.Accesskey{})
			},
		},
		{
			name:         "accesskey get",
			cacheMissFor: "accesskey",
			expectPhrase: "could not query. accesskeyGetFromDB",
			call: func(h *handler) error {
				_, err := h.AccesskeyGet(context.Background(), accesskeyID)
				return err
			},
		},
		{
			name:         "accesskey list",
			expectPhrase: "could not query. AccesskeyGets",
			call: func(h *handler) error {
				_, err := h.AccesskeyList(context.Background(), 10, "", map[accesskey.Field]any{})
				return err
			},
		},
		{
			name:         "accesskey update",
			expectPhrase: "AccesskeyUpdate: exec failed",
			call: func(h *handler) error {
				return h.AccesskeyUpdate(context.Background(), accesskeyID, map[accesskey.Field]any{accesskey.FieldName: "name"})
			},
		},
		{
			name:         "accesskey delete",
			expectPhrase: "AccesskeyDelete: exec failed",
			call: func(h *handler) error {
				return h.AccesskeyDelete(context.Background(), accesskeyID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the mock cache has no expectation other than the miss below:
			// a failed database call must not write the cache
			mockCache := cachehandler.NewMockCacheHandler(mc)
			switch tt.cacheMissFor {
			case "customer":
				mockCache.EXPECT().CustomerGet(gomock.Any(), customerID).Return(nil, fmt.Errorf("cache miss"))
			case "accesskey":
				mockCache.EXPECT().AccesskeyGet(gomock.Any(), accesskeyID).Return(nil, fmt.Errorf("cache miss"))
			}

			h := newClosedDBHandler(t, mockCache)

			err := tt.call(h)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), tt.expectPhrase) {
				t.Errorf("Wrong match. expect the error to contain %q, got: %v", tt.expectPhrase, err)
			}
			if !strings.Contains(err.Error(), "database is closed") {
				t.Errorf("Wrong match. expect the driver error to be kept in the message, got: %v", err)
			}
			if errors.Unwrap(err) == nil {
				t.Errorf("Wrong match. expect a wrapped error, got: %v", err)
			}
		})
	}
}

func Test_CustomerDelete_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no cache expectation: a delete that matched no row must not touch the cache
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          dbTest,
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	err := h.CustomerDelete(context.Background(), uuid.FromStringOrNil("3f6a9b10-0000-4000-8000-0000000000a1"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
}

func Test_AccesskeyDelete_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          dbTest,
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	err := h.AccesskeyDelete(context.Background(), uuid.FromStringOrNil("3f6a9b10-0000-4000-8000-0000000000a2"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
}

func Test_customerCacheWriteIsBestEffort(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("3f6a9b10-0000-4000-8000-0000000000b1")
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
	if err := h.CustomerCreate(ctx, &customer.Customer{ID: id, Name: "before"}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	defer func() { _ = h.CustomerHardDelete(ctx, id) }()

	// the cache write after an update fails, but the update itself succeeded
	mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(fmt.Errorf("cache error"))
	if err := h.CustomerUpdate(ctx, id, map[customer.Field]any{customer.FieldName: "after"}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	// the read path still returns the row when the cache write fails
	mockCache.EXPECT().CustomerGet(ctx, id).Return(nil, fmt.Errorf("cache miss"))
	mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(fmt.Errorf("cache error"))
	res, err := h.CustomerGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Name != "after" {
		t.Errorf("Wrong match. expect: after, got: %v", res.Name)
	}

	// the internal helper reports the cache error to its caller
	mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(fmt.Errorf("cache error"))
	if err := h.customerUpdateToCache(ctx, id); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_accesskeyCacheWriteIsBestEffort(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// a fresh id per run: the shared test database keeps soft-deleted rows
	id := uuid.Must(uuid.NewV4())
	t.Cleanup(func() {
		_, _ = dbTest.Exec("DELETE FROM customer_accesskeys WHERE id = ?", id.Bytes())
	})
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	a := &accesskey.Accesskey{Name: "before"}
	a.ID = id
	mockCache.EXPECT().AccesskeySet(ctx, gomock.Any()).Return(nil)
	if err := h.AccesskeyCreate(ctx, a); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	mockCache.EXPECT().AccesskeySet(ctx, gomock.Any()).Return(fmt.Errorf("cache error"))
	if err := h.AccesskeyUpdate(ctx, id, map[accesskey.Field]any{accesskey.FieldName: "after"}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mockCache.EXPECT().AccesskeyGet(ctx, id).Return(nil, fmt.Errorf("cache miss"))
	mockCache.EXPECT().AccesskeySet(ctx, gomock.Any()).Return(fmt.Errorf("cache error"))
	res, err := h.AccesskeyGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Name != "after" {
		t.Errorf("Wrong match. expect: after, got: %v", res.Name)
	}

	mockCache.EXPECT().AccesskeySet(ctx, gomock.Any()).Return(fmt.Errorf("cache error"))
	if err := h.accesskeyUpdateToCache(ctx, id); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}

	mockCache.EXPECT().AccesskeySet(ctx, gomock.Any()).Return(nil)
	if err := h.AccesskeyDelete(ctx, id); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_NewHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	res := NewHandler(dbTest, mockCache)
	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler, got: %T", res)
	}
	if h.db != dbTest || h.cache != mockCache || h.utilHandler == nil {
		t.Errorf("Wrong match. expect all dependencies to be set, got: %+v", h)
	}
}
