package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/cachehandler"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// newClosedDB returns a sqlite database that is already closed, so that every
// statement fails. It uses a private in-memory DSN that is different from the
// shared dbTest used by TestMain.
func newClosedDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file:voip1647closed?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("could not open the sqlite db. err: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("could not close the sqlite db. err: %v", err)
	}

	return db
}

func Test_NewHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	res := NewHandler(dbTest, mockCache)

	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong type. expect: *handler, got: %T", res)
	}
	if h.db != dbTest {
		t.Errorf("Wrong db. expect: the given db, got: %v", h.db)
	}
	if h.cache != mockCache {
		t.Errorf("Wrong cache. expect: the given cache, got: %v", h.cache)
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong utilHandler. expect: not nil, got: nil")
	}
}

func Test_PipecatcallGet_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("c1647001-0000-11f0-a000-000000000001")
	mockCache.EXPECT().PipecatcallGet(ctx, id).Return(nil, errors.New("cache miss"))

	res, err := h.PipecatcallGet(ctx, id)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong error. expect: %v, got: %v", ErrNotFound, err)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}

func Test_PipecatcallGet_cacheHit(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	// the db is closed: a cache hit must be returned without touching the db.
	h := handler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          newClosedDB(t),
		cache:       mockCache,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("c1647001-0000-11f0-a000-000000000002")
	cached := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{ID: id},
		HostID:   "cached-host",
	}
	mockCache.EXPECT().PipecatcallGet(ctx, id).Return(cached, nil)

	res, err := h.PipecatcallGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong error. expect: nil, got: %v", err)
	}
	if res != cached {
		t.Errorf("Wrong result. expect: the cached object, got: %v", res)
	}
}

func Test_PipecatcallGet_cacheSetFailureIgnored(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	// a random id keeps the test repeatable on the shared sqlite database (-count > 1).
	id := uuid.Must(uuid.NewV4())
	curTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	// create the record; the cache write failure must not fail the create.
	mockUtil.EXPECT().TimeNow().Return(&curTime)
	mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any()).Return(errors.New("cache down"))
	if err := h.PipecatcallCreate(ctx, &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{ID: id},
		HostID:   "db-host",
	}); err != nil {
		t.Fatalf("Wrong create error. expect: nil, got: %v", err)
	}

	// cache miss, db hit, cache write failure ignored.
	mockCache.EXPECT().PipecatcallGet(ctx, id).Return(nil, errors.New("cache miss"))
	mockCache.EXPECT().PipecatcallSet(ctx, gomock.Any()).Return(errors.New("cache down"))

	res, err := h.PipecatcallGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong error. expect: nil, got: %v", err)
	}
	if res.ID != id || res.HostID != "db-host" {
		t.Errorf("Wrong result. expect: id %s host db-host, got: id %s host %s", id, res.ID, res.HostID)
	}
}

func Test_PipecatcallUpdate_emptyFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no mock expectations: any util, cache or db call fails the test.
	h := handler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          newClosedDB(t),
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	id := uuid.FromStringOrNil("c1647001-0000-11f0-a000-000000000004")
	if err := h.PipecatcallUpdate(context.Background(), id, map[pipecatcall.Field]any{}); err != nil {
		t.Errorf("Wrong error. expect: nil, got: %v", err)
	}
	if err := h.PipecatcallUpdate(context.Background(), id, nil); err != nil {
		t.Errorf("Wrong error with nil fields. expect: nil, got: %v", err)
	}
}

func Test_PipecatcallDelete_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	// no cache expectation: a missing record must not touch the cache.
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	curTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&curTime)

	id := uuid.FromStringOrNil("c1647001-0000-11f0-a000-000000000005")
	if err := h.PipecatcallDelete(context.Background(), id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong error. expect: %v, got: %v", ErrNotFound, err)
	}
}

func Test_Pipecatcall_closedDB(t *testing.T) {
	id := uuid.FromStringOrNil("c1647001-0000-11f0-a000-000000000006")
	curTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		expectTimeNow bool
		expectCache   bool
		expectErrText string
		notNotFound   bool

		call func(h *handler) error
	}{
		{
			name:          "create exec failure",
			expectTimeNow: true,
			expectErrText: "could not execute query. PipecatcallCreate",
			call: func(h *handler) error {
				return h.PipecatcallCreate(context.Background(), &pipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{ID: id},
				})
			},
		},
		{
			name:          "get query failure after cache miss",
			expectCache:   true,
			expectErrText: "could not query. pipecatcallGetFromDB",
			notNotFound:   true,
			call: func(h *handler) error {
				_, err := h.PipecatcallGet(context.Background(), id)
				return err
			},
		},
		{
			name:          "update exec failure",
			expectTimeNow: true,
			expectErrText: "PipecatcallUpdate: exec failed",
			call: func(h *handler) error {
				return h.PipecatcallUpdate(context.Background(), id, map[pipecatcall.Field]any{
					pipecatcall.FieldHostID: "new-host",
				})
			},
		},
		{
			name:          "delete exec failure",
			expectTimeNow: true,
			expectErrText: "PipecatcallDelete: exec failed",
			notNotFound:   true,
			call: func(h *handler) error {
				return h.PipecatcallDelete(context.Background(), id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{
				utilHandler: mockUtil,
				db:          newClosedDB(t),
				cache:       mockCache,
			}

			if tt.expectTimeNow {
				mockUtil.EXPECT().TimeNow().Return(&curTime)
			}
			if tt.expectCache {
				mockCache.EXPECT().PipecatcallGet(gomock.Any(), id).Return(nil, errors.New("cache miss"))
			}

			err := tt.call(h)
			if err == nil {
				t.Fatalf("Wrong error. expect: an error, got: nil")
			}
			if !strings.Contains(err.Error(), tt.expectErrText) {
				t.Errorf("Wrong error text. expect contains: %q, got: %v", tt.expectErrText, err)
			}
			if tt.notNotFound && errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong error. a db failure must not be reported as ErrNotFound, got: %v", err)
			}
		})
	}
}
