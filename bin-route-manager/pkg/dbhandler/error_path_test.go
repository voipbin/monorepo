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
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/cachehandler"
)

const closedDBMessage = "sql: database is closed"

// newClosedDBHandler returns a handler whose database connection is already closed. The connection
// is separate from the shared sqlite database of the package, so other tests are not affected.
func newClosedDBHandler(t *testing.T, mc *gomock.Controller) (*handler, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open the database. err: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("could not close the database. err: %v", err)
	}

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		utilHandler: mockUtil,
		db:          db,
		cache:       mockCache,
	}

	return h, mockUtil, mockCache
}

// newLiveTestHandler returns a handler on the shared sqlite database.
func newLiveTestHandler(mc *gomock.Controller) (*handler, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	return h, mockUtil, mockCache
}

// cleanupRows removes the rows of the given table before and after the test, because the sqlite
// database is shared by all tests of the package and the ids of other tests are fixed.
func cleanupRows(t *testing.T, table string, ids ...uuid.UUID) {
	t.Helper()

	del := func() {
		for _, id := range ids {
			if _, err := dbTest.Exec("DELETE FROM "+table+" WHERE id = ?", id.Bytes()); err != nil {
				t.Errorf("could not clean up the row. table: %s, id: %s, err: %v", table, id, err)
			}
		}
	}
	del()
	t.Cleanup(del)
}

// assertClosedDBError checks that the error wraps the closed database failure and carries the
// expected function context.
func assertClosedDBError(t *testing.T, err error, expectContext string) {
	t.Helper()

	if err == nil {
		t.Fatalf("Wrong match. expect: error containing %q, got: nil", expectContext)
	}
	if !strings.Contains(err.Error(), expectContext) {
		t.Errorf("Wrong match. expect: message containing %q, got: %v", expectContext, err)
	}
	if !strings.Contains(err.Error(), closedDBMessage) {
		t.Errorf("Wrong match. expect: message containing %q, got: %v", closedDBMessage, err)
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("Wrong match. expect: wrapped error, got: %v", err)
	}
}

func Test_Provider_ClosedDB(t *testing.T) {
	id := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	now := timePtr(time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC))

	tests := []struct {
		name string

		run func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error

		expectContext string
	}{
		{
			name: "create fails to execute",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				mockUtil.EXPECT().TimeNow().Return(now)
				return h.ProviderCreate(ctx, &provider.Provider{ID: id})
			},
			expectContext: "could not execute query. ProviderCreate",
		},
		{
			name: "update fails to execute",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				mockUtil.EXPECT().TimeNow().Return(now)
				return h.ProviderUpdate(ctx, id, map[provider.Field]any{provider.FieldName: "name"})
			},
			expectContext: "ProviderUpdate: exec failed",
		},
		{
			name: "delete fails to execute",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				mockUtil.EXPECT().TimeNow().Return(now)
				return h.ProviderDelete(ctx, id)
			},
			expectContext: "ProviderDelete: exec failed",
		},
		{
			name: "get from db fails to query",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				_, err := h.providerGetFromDB(ctx, id)
				return err
			},
			expectContext: "could not query. providerGetFromDB",
		},
		{
			name: "get falls through the cache miss and the db error is returned",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				// the cache must not be filled after the db failure (no ProviderSet expectation)
				mockCache.EXPECT().ProviderGet(ctx, id).Return(nil, fmt.Errorf("cache miss"))
				_, err := h.ProviderGet(ctx, id)
				return err
			},
			expectContext: "could not query. providerGetFromDB",
		},
		{
			name: "list fails to query",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				_, err := h.ProviderList(ctx, "2021-01-01T00:00:00.000000Z", 10, map[provider.Field]any{})
				return err
			},
			expectContext: "could not query. ProviderGets",
		},
		{
			name: "list with an empty token asks the util handler and fails to query",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				mockUtil.EXPECT().TimeGetCurTime().Return("2021-01-01T00:00:00.000000Z")
				_, err := h.ProviderList(ctx, "", 10, map[provider.Field]any{})
				return err
			},
			expectContext: "could not query. ProviderGets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockUtil, mockCache := newClosedDBHandler(t, mc)

			err := tt.run(h, context.Background(), mockUtil, mockCache)
			assertClosedDBError(t, err, tt.expectContext)
		})
	}
}

func Test_Route_ClosedDB(t *testing.T) {
	id := uuid.FromStringOrNil("c2000000-0000-0000-0000-000000000001")
	now := timePtr(time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC))

	tests := []struct {
		name string

		run func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error

		expectContext string
	}{
		{
			name: "create fails to execute",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				mockUtil.EXPECT().TimeNow().Return(now)
				return h.RouteCreate(ctx, &route.Route{ID: id})
			},
			expectContext: "could not execute query. RouteCreate",
		},
		{
			name: "update fails to execute",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				mockUtil.EXPECT().TimeNow().Return(now)
				return h.RouteUpdate(ctx, id, map[route.Field]any{route.FieldName: "name"})
			},
			expectContext: "RouteUpdate: exec failed",
		},
		{
			name: "delete fails to execute",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				mockUtil.EXPECT().TimeNow().Return(now)
				return h.RouteDelete(ctx, id)
			},
			expectContext: "RouteDelete: exec failed",
		},
		{
			name: "get from db fails to query",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				_, err := h.routeGetFromDB(ctx, id)
				return err
			},
			expectContext: "could not query. routeGetFromDB",
		},
		{
			name: "get falls through the cache miss and the db error is returned",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				// the cache must not be filled after the db failure (no RouteSet expectation)
				mockCache.EXPECT().RouteGet(ctx, id).Return(nil, fmt.Errorf("cache miss"))
				_, err := h.RouteGet(ctx, id)
				return err
			},
			expectContext: "could not query. routeGetFromDB",
		},
		{
			name: "list fails to query",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				_, err := h.RouteList(ctx, "2021-01-01T00:00:00.000000Z", 10, map[route.Field]any{})
				return err
			},
			expectContext: "could not query. RouteGets",
		},
		{
			name: "list with an empty token asks the util handler and fails to query",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler) error {
				mockUtil.EXPECT().TimeGetCurTime().Return("2021-01-01T00:00:00.000000Z")
				_, err := h.RouteList(ctx, "", 10, map[route.Field]any{})
				return err
			},
			expectContext: "could not query. RouteGets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockUtil, mockCache := newClosedDBHandler(t, mc)

			err := tt.run(h, context.Background(), mockUtil, mockCache)
			assertClosedDBError(t, err, tt.expectContext)
		})
	}
}

func Test_ProviderCall_ClosedDB(t *testing.T) {
	id := uuid.FromStringOrNil("c3000000-0000-0000-0000-000000000001")
	now := timePtr(time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC))

	tests := []struct {
		name string

		run func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler) error

		expectContext string
	}{
		{
			name: "create fails to execute",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler) error {
				mockUtil.EXPECT().TimeNow().Return(now)
				return h.ProviderCallCreate(ctx, &providercall.ProviderCall{ID: id})
			},
			expectContext: "could not execute query. ProviderCallCreate",
		},
		{
			name: "delete fails to execute",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler) error {
				mockUtil.EXPECT().TimeNow().Return(now)
				return h.ProviderCallDelete(ctx, id)
			},
			expectContext: "ProviderCallDelete: exec failed",
		},
		{
			name: "get fails to query",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler) error {
				_, err := h.ProviderCallGet(ctx, id)
				return err
			},
			expectContext: "could not query. providerCallGetFromDB",
		},
		{
			name: "list fails to query",
			run: func(h *handler, ctx context.Context, mockUtil *utilhandler.MockUtilHandler) error {
				_, err := h.ProviderCallList(ctx, "2021-01-01T00:00:00.000000Z", 10, map[providercall.Field]any{})
				return err
			},
			expectContext: "could not query. ProviderCallList",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockUtil, _ := newClosedDBHandler(t, mc)

			err := tt.run(h, context.Background(), mockUtil)
			assertClosedDBError(t, err, tt.expectContext)
		})
	}
}

func Test_ProviderGetFromDB_ErrNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _ := newLiveTestHandler(mc)

	res, err := h.providerGetFromDB(context.Background(), uuid.FromStringOrNil("c1000000-0000-0000-0000-0000000000ff"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_RouteGetFromDB_ErrNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _ := newLiveTestHandler(mc)

	res, err := h.routeGetFromDB(context.Background(), uuid.FromStringOrNil("c2000000-0000-0000-0000-0000000000ff"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_ProviderList_InvalidFilter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _ := newLiveTestHandler(mc)
	ctx := context.Background()

	// a filter value type that cannot be bound is rejected while applying the filters
	_, err := h.ProviderList(ctx, "2021-01-01T00:00:00.000000Z", 10, map[provider.Field]any{provider.FieldName: struct{}{}})
	if err == nil || !strings.Contains(err.Error(), "could not apply filters. ProviderGets") {
		t.Errorf("Wrong match. expect: could not apply filters error, got: %v", err)
	}

	// a column that does not exist fails at query time
	_, err = h.ProviderList(ctx, "2021-01-01T00:00:00.000000Z", 10, map[provider.Field]any{provider.Field("no_such_column"): "x"})
	if err == nil || !strings.Contains(err.Error(), "could not query. ProviderGets") {
		t.Errorf("Wrong match. expect: could not query error, got: %v", err)
	}
}

func Test_RouteList_InvalidFilter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _ := newLiveTestHandler(mc)
	ctx := context.Background()

	_, err := h.RouteList(ctx, "2021-01-01T00:00:00.000000Z", 10, map[route.Field]any{route.FieldName: struct{}{}})
	if err == nil || !strings.Contains(err.Error(), "could not apply filters. RouteGets") {
		t.Errorf("Wrong match. expect: could not apply filters error, got: %v", err)
	}

	_, err = h.RouteList(ctx, "2021-01-01T00:00:00.000000Z", 10, map[route.Field]any{route.Field("no_such_column"): "x"})
	if err == nil || !strings.Contains(err.Error(), "could not query. RouteGets") {
		t.Errorf("Wrong match. expect: could not query error, got: %v", err)
	}
}

func Test_ProviderList_EmptyToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockCache := newLiveTestHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000010")
	cleanupRows(t, providersTable, id)

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
	mockCache.EXPECT().ProviderSet(ctx, gomock.Any()).Return(nil)
	if err := h.ProviderCreate(ctx, &provider.Provider{ID: id, Name: "empty-token-provider"}); err != nil {
		t.Fatalf("could not seed. err: %v", err)
	}

	// the stored creation time is used as the boundary token, so the row must be excluded when the
	// comparison is strict.
	var stored string
	if err := dbTest.QueryRow("SELECT tm_create FROM route_providers WHERE id = ?", id.Bytes()).Scan(&stored); err != nil {
		t.Fatalf("could not read the stored creation time. err: %v", err)
	}

	filters := map[provider.Field]any{provider.FieldName: "empty-token-provider"}
	tests := []struct {
		name     string
		token    string
		expectID bool
	}{
		{"token later than the row includes it", "2999-01-01T00:00:00.000000Z", true},
		{"token earlier than the row excludes it", "1999-01-01T00:00:00.000000Z", false},
		{"token equal to the stored creation time excludes it", stored, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.token)
			res, err := h.ProviderList(ctx, "", 100, filters)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			found := len(res) == 1 && res[0].ID == id
			if found != tt.expectID || (!tt.expectID && len(res) != 0) {
				t.Errorf("Wrong match. expect found: %v, got: %v", tt.expectID, res)
			}
		})
	}
}

func Test_RouteList_EmptyToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockCache := newLiveTestHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("c2000000-0000-0000-0000-000000000010")
	customerID := uuid.FromStringOrNil("c2000000-0000-0000-0000-0000000000a0")
	cleanupRows(t, routesTable, id)

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
	mockCache.EXPECT().RouteSet(ctx, gomock.Any()).Return(nil)
	if err := h.RouteCreate(ctx, &route.Route{ID: id, CustomerID: customerID}); err != nil {
		t.Fatalf("could not seed. err: %v", err)
	}

	var stored string
	if err := dbTest.QueryRow("SELECT tm_create FROM route_routes WHERE id = ?", id.Bytes()).Scan(&stored); err != nil {
		t.Fatalf("could not read the stored creation time. err: %v", err)
	}

	filters := map[route.Field]any{route.FieldCustomerID: customerID}
	tests := []struct {
		name     string
		token    string
		expectID bool
	}{
		{"token later than the row includes it", "2999-01-01T00:00:00.000000Z", true},
		{"token earlier than the row excludes it", "1999-01-01T00:00:00.000000Z", false},
		{"token equal to the stored creation time excludes it", stored, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.token)
			res, err := h.RouteList(ctx, "", 100, filters)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			found := len(res) == 1 && res[0].ID == id
			if found != tt.expectID || (!tt.expectID && len(res) != 0) {
				t.Errorf("Wrong match. expect found: %v, got: %v", tt.expectID, res)
			}
		})
	}
}

func Test_ProviderUpdateToCache(t *testing.T) {
	id := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000020")
	cacheErr := fmt.Errorf("cache is down")

	t.Run("missing row returns not found without touching the cache", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, _ := newLiveTestHandler(mc)
		cleanupRows(t, providersTable, id)

		if err := h.providerUpdateToCache(context.Background(), id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
		}
	})

	t.Run("cache set failure is returned by the update helper but ignored by create", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockUtil, mockCache := newLiveTestHandler(mc)
		ctx := context.Background()
		cleanupRows(t, providersTable, id)

		mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
		mockCache.EXPECT().ProviderSet(ctx, gomock.Any()).Return(cacheErr)
		if err := h.ProviderCreate(ctx, &provider.Provider{ID: id}); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		mockCache.EXPECT().ProviderSet(ctx, gomock.Any()).Return(cacheErr)
		if err := h.providerUpdateToCache(ctx, id); !errors.Is(err, cacheErr) {
			t.Errorf("Wrong match. expect: %v, got: %v", cacheErr, err)
		}
	})
}

func Test_RouteUpdateToCache(t *testing.T) {
	id := uuid.FromStringOrNil("c2000000-0000-0000-0000-000000000020")
	cacheErr := fmt.Errorf("cache is down")

	t.Run("missing row returns not found without touching the cache", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, _ := newLiveTestHandler(mc)
		cleanupRows(t, routesTable, id)

		if err := h.routeUpdateToCache(context.Background(), id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
		}
	})

	t.Run("cache set failure is returned by the update helper but ignored by create", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockUtil, mockCache := newLiveTestHandler(mc)
		ctx := context.Background()
		cleanupRows(t, routesTable, id)

		mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
		mockCache.EXPECT().RouteSet(ctx, gomock.Any()).Return(cacheErr)
		if err := h.RouteCreate(ctx, &route.Route{ID: id}); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		mockCache.EXPECT().RouteSet(ctx, gomock.Any()).Return(cacheErr)
		if err := h.routeUpdateToCache(ctx, id); !errors.Is(err, cacheErr) {
			t.Errorf("Wrong match. expect: %v, got: %v", cacheErr, err)
		}
	})
}
