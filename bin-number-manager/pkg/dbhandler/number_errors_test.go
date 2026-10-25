package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/cachehandler"
)

// newPrivateDB returns a standalone in-memory sqlite database with the numbers schema.
// It is independent of the shared dbTest so a test may close it.
func newPrivateDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open db. err: %v", err)
	}
	db.SetMaxOpenConns(1)

	schema, err := os.ReadFile("../../scripts/database_scripts_test/table_numbers.sql")
	if err != nil {
		t.Fatalf("could not read schema. err: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("could not apply schema. err: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newClosedDB(t *testing.T) *sql.DB {
	t.Helper()

	db := newPrivateDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("could not close db. err: %v", err)
	}
	return db
}

func Test_NewHandler_fields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	db := newPrivateDB(t)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h, ok := NewHandler(db, mockCache).(*handler)
	if !ok {
		t.Fatalf("NewHandler did not return *handler")
	}
	if h.db != db {
		t.Errorf("db was not kept")
	}
	if h.cache != mockCache {
		t.Errorf("cache was not kept")
	}
	if h.utilHandler == nil {
		t.Errorf("utilHandler is nil")
	}
}

func Test_closedDB_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6a1f0c1e-0001-11ee-8a01-000000000001")

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{
			"create",
			func(h *handler) error {
				return h.NumberCreate(context.Background(), &number.Number{
					Identity: commonidentity.Identity{ID: id},
					Number:   "+15550000001",
				})
			},
		},
		{
			"get",
			func(h *handler) error {
				_, err := h.NumberGet(context.Background(), id)
				return err
			},
		},
		{
			"list",
			func(h *handler) error {
				_, err := h.NumberList(context.Background(), 10, "2030-01-01 00:00:00.000000", nil)
				return err
			},
		},
		{
			"update",
			func(h *handler) error {
				return h.NumberUpdate(context.Background(), id, map[number.Field]any{number.FieldName: "n"})
			},
		},
		{
			"delete",
			func(h *handler) error {
				return h.NumberDelete(context.Background(), id)
			},
		},
		{
			"get existing numbers",
			func(h *handler) error {
				_, err := h.NumberGetExistingNumbers(context.Background(), []string{"+15550000001"})
				return err
			},
		},
		{
			"count virtual",
			func(h *handler) error {
				_, err := h.NumberCountVirtualByCustomerID(context.Background(), id)
				return err
			},
		},
		{
			"gets by tm renew",
			func(h *handler) error {
				_, err := h.NumberGetsByTMRenew(context.Background(), "2030-01-01 00:00:00.000000", 10, nil, nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockCache.EXPECT().NumberGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss")).AnyTimes()

			h := &handler{
				utilHandler: utilhandler.NewUtilHandler(),
				db:          newClosedDB(t),
				cache:       mockCache,
			}

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error on a closed database, got nil")
			}
			if !strings.Contains(err.Error(), "database is closed") {
				t.Errorf("expected a closed database error, got: %v", err)
			}
		})
	}
}

func Test_NumberGet_cacheHit_skipsDB(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("6a1f0c1e-0002-11ee-8a01-000000000002")
	cached := &number.Number{
		Identity: commonidentity.Identity{ID: id},
		Number:   "+15550000002",
	}

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().NumberGet(gomock.Any(), id).Return(cached, nil)

	// a closed database proves that the database is not queried on a cache hit.
	h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newClosedDB(t), cache: mockCache}

	res, err := h.NumberGet(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(res, cached) {
		t.Errorf("wrong match. expect: %v, got: %v", cached, res)
	}
}

func Test_NumberGet_cacheMiss_fallsBackToDB(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("6a1f0c1e-0003-11ee-8a01-000000000003")

	mockCache := cachehandler.NewMockCacheHandler(mc)
	// create and get both try to fill the cache; its failure must not fail the call.
	mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).Return(errors.New("cache set failed")).Times(2)
	mockCache.EXPECT().NumberGet(gomock.Any(), id).Return(nil, errors.New("cache miss"))

	h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newPrivateDB(t), cache: mockCache}

	if err := h.NumberCreate(context.Background(), &number.Number{
		Identity: commonidentity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("6a1f0c1e-0099-11ee-8a01-000000000099")},
		Number:   "+15550000003",
		Status:   number.StatusActive,
	}); err != nil {
		t.Fatalf("create must succeed even if the cache fails. err: %v", err)
	}

	res, err := h.NumberGet(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != id || res.Number != "+15550000003" {
		t.Errorf("wrong row. id: %s, number: %s", res.ID, res.Number)
	}
}

func Test_NumberList_emptyToken_usesCurrentTime(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).Return(nil)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &handler{utilHandler: mockUtil, db: newPrivateDB(t), cache: mockCache}

	createdAt := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)
	id := uuid.FromStringOrNil("6a1f0c1e-0004-11ee-8a01-000000000004")
	mockUtil.EXPECT().TimeNow().Return(&createdAt)
	if err := h.NumberCreate(context.Background(), &number.Number{
		Identity: commonidentity.Identity{ID: id},
		Number:   "+15550000004",
	}); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// created at 2021, so a token of 2030 includes it and a token of 2020 does not.
	mockUtil.EXPECT().TimeGetCurTime().Return("2030-01-01 00:00:00.000000")
	res, err := h.NumberList(context.Background(), 10, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].ID != id {
		t.Errorf("expected exactly the created number, got %d rows", len(res))
	}

	res, err = h.NumberList(context.Background(), 10, "2020-01-01 00:00:00.000000", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected no rows before the creation time, got %d", len(res))
	}
}

func Test_unsupportedFilter_errors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          newPrivateDB(t),
		cache:       cachehandler.NewMockCacheHandler(mc),
	}
	filters := map[number.Field]any{number.FieldName: struct{ X int }{X: 1}}

	if _, err := h.NumberList(context.Background(), 10, "2030-01-01 00:00:00.000000", filters); err == nil {
		t.Errorf("NumberList: expected an error for an unsupported filter value")
	}
	if _, err := h.NumberGetsByTMRenew(context.Background(), "2030-01-01 00:00:00.000000", 10, filters, nil); err == nil {
		t.Errorf("NumberGetsByTMRenew: expected an error for an unsupported filter value")
	}
}

func Test_cacheHelpers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("6a1f0c1e-0005-11ee-8a01-000000000005")
	cacheErr := errors.New("cache failed")
	cached := &number.Number{Identity: commonidentity.Identity{ID: id}, Number: "+15550000005"}

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newPrivateDB(t), cache: mockCache}

	// by number: success and error pass through
	mockCache.EXPECT().NumberGetByNumber(gomock.Any(), "+15550000005").Return(cached, nil)
	res, err := h.NumberGetFromCacheByNumber(context.Background(), "+15550000005")
	if err != nil || !reflect.DeepEqual(res, cached) {
		t.Errorf("by number success. res: %v, err: %v", res, err)
	}

	mockCache.EXPECT().NumberGetByNumber(gomock.Any(), "+15550000006").Return(nil, cacheErr)
	if _, err := h.NumberGetFromCacheByNumber(context.Background(), "+15550000006"); !errors.Is(err, cacheErr) {
		t.Errorf("by number error. expect: %v, got: %v", cacheErr, err)
	}

	// set error pass through
	mockCache.EXPECT().NumberSet(gomock.Any(), cached).Return(cacheErr)
	if err := h.numberSetToCache(context.Background(), cached); !errors.Is(err, cacheErr) {
		t.Errorf("set error. expect: %v, got: %v", cacheErr, err)
	}

	// update to cache: the row does not exist, so the DB read fails and the cache is not touched
	if err := h.numberUpdateToCache(context.Background(), id); err == nil {
		t.Errorf("update to cache: expected an error for a missing row")
	}
}

func Test_NumberDelete_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          newPrivateDB(t),
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	err := h.NumberDelete(context.Background(), uuid.FromStringOrNil("6a1f0c1e-0006-11ee-8a01-000000000006"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
