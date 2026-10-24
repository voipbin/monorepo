package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/cachehandler"
)

var errCacheTest = errors.New("simulated cache failure")

// newClosedDB returns a database handle that is already closed, so that every
// statement fails the way a lost connection does.
func newClosedDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", `file:closed_db_for_test?mode=memory`)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return db
}

func Test_NewHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	res, ok := NewHandler(dbTest, mockCache).(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler")
	}
	if res.db != dbTest || res.cache != mockCache || res.util == nil {
		t.Errorf("Wrong match. the dependencies were not stored as given")
	}
}

// Test_closedDB_errorsPropagate checks that a database failure is returned to the
// caller, wrapped with the failing operation, and never turned into a success or
// into ErrNotFound.
func Test_closedDB_errorsPropagate(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("7e8f9a0b-0000-0000-0000-000000000001")

	tests := []struct {
		name   string
		expect string // the operation named in the error
		fn     func(h *handler) error
	}{
		{"AccountCreate", "AccountCreate", func(h *handler) error { return h.AccountCreate(ctx, &account.Account{ID: id}) }},
		{"AccountGet", "accountGetFromDB", func(h *handler) error { _, err := h.AccountGet(ctx, id); return err }},
		{"AccountList", "AccountGets", func(h *handler) error { _, err := h.AccountList(ctx, "", 5, nil); return err }},
		{"AccountUpdate", "AccountUpdate", func(h *handler) error {
			return h.AccountUpdate(ctx, id, map[account.Field]any{account.FieldTotalFileCount: 1})
		}},
		{"AccountIncreaseFileInfo", "AccountIncreaseFile", func(h *handler) error { return h.AccountIncreaseFileInfo(ctx, id, 1, 10) }},
		{"AccountDecreaseFileInfo", "AccountDecreaseFileInfo", func(h *handler) error { return h.AccountDecreaseFileInfo(ctx, id, 1, 10) }},
		{"AccountDelete", "AccountDelete", func(h *handler) error { return h.AccountDelete(ctx, id) }},
		{"FileCreate", "FileCreate", func(h *handler) error { return h.FileCreate(ctx, &file.File{}) }},
		{"FileGet", "fileGetFromDB", func(h *handler) error { _, err := h.FileGet(ctx, id); return err }},
		{"FileList", "FileGets", func(h *handler) error { _, err := h.FileList(ctx, "", 5, nil); return err }},
		{"FileUpdate", "FileUpdate", func(h *handler) error {
			return h.FileUpdate(ctx, id, map[file.Field]any{file.FieldName: "after"})
		}},
		{"FileDelete", "FileDelete", func(h *handler) error { return h.FileDelete(ctx, id) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
			mockUtil.EXPECT().TimeGetCurTime().Return("2999-01-01 00:00:00.000000").AnyTimes()

			// the cache has no expectations except a miss: nothing may be cached when the db failed.
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockCache.EXPECT().AccountGet(gomock.Any(), gomock.Any()).Return(nil, errCacheTest).AnyTimes()
			mockCache.EXPECT().FileGet(gomock.Any(), gomock.Any()).Return(nil, errCacheTest).AnyTimes()

			h := &handler{util: mockUtil, db: newClosedDB(t), cache: mockCache}

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. a database failure must not be reported as not found, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("Wrong match. expect the error to name %q, got: %v", tt.expect, err)
			}
		})
	}
}

func Test_Get_cacheHitSkipsTheDB(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("8f9a0b1c-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{util: utilhandler.NewMockUtilHandler(mc), db: newClosedDB(t), cache: mockCache}

	cachedAccount := &account.Account{ID: id, TotalFileCount: 3}
	mockCache.EXPECT().AccountGet(ctx, id).Return(cachedAccount, nil)
	if res, err := h.AccountGet(ctx, id); err != nil || res != cachedAccount {
		t.Errorf("Wrong match. expect the cached account, got: %v/%v", res, err)
	}

	cachedFile := &file.File{Name: "cached"}
	mockCache.EXPECT().FileGet(ctx, id).Return(cachedFile, nil)
	if res, err := h.FileGet(ctx, id); err != nil || res != cachedFile {
		t.Errorf("Wrong match. expect the cached file, got: %v/%v", res, err)
	}
}

func Test_Get_cacheMissFillsTheCacheAndToleratesItsFailure(t *testing.T) {
	ctx := context.Background()

	accountID := uuid.Must(uuid.NewV4())
	fileID := uuid.Must(uuid.NewV4())
	insertTestAccount(t, accountID)
	insertTestFile(t, fileID)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{util: utilhandler.NewMockUtilHandler(mc), db: dbTest, cache: mockCache}

	mockCache.EXPECT().AccountGet(ctx, accountID).Return(nil, errCacheTest)
	mockCache.EXPECT().AccountSet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, a *account.Account) error {
		if a.ID != accountID || a.TotalFileCount != 5 || a.TotalFileSize != 50 {
			t.Errorf("Wrong match. expect the stored row to be cached, got: %+v", a)
		}
		return errCacheTest
	})
	res, err := h.AccountGet(ctx, accountID)
	if err != nil || res.ID != accountID || res.TotalFileCount != 5 {
		t.Errorf("Wrong match. expect the stored account despite the cache failure, got: %v/%v", res, err)
	}

	mockCache.EXPECT().FileGet(ctx, fileID).Return(nil, errCacheTest)
	mockCache.EXPECT().FileSet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, f *file.File) error {
		if f.ID != fileID || f.Name != "before" {
			t.Errorf("Wrong match. expect the stored row to be cached, got: %+v", f)
		}
		return errCacheTest
	})
	resFile, err := h.FileGet(ctx, fileID)
	if err != nil || resFile.ID != fileID || resFile.Name != "before" {
		t.Errorf("Wrong match. expect the stored file despite the cache failure, got: %v/%v", resFile, err)
	}

	// a row that does not exist is not found, and nothing is cached for it.
	missing := uuid.Must(uuid.NewV4())
	mockCache.EXPECT().AccountGet(ctx, missing).Return(nil, errCacheTest)
	mockCache.EXPECT().FileGet(ctx, missing).Return(nil, errCacheTest)
	if _, err := h.AccountGet(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
	if _, err := h.FileGet(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func Test_Delete_cacheFailureIsTolerated(t *testing.T) {
	ctx := context.Background()

	accountID := uuid.Must(uuid.NewV4())
	fileID := uuid.Must(uuid.NewV4())
	insertTestAccount(t, accountID)
	insertTestFile(t, fileID)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{util: mockUtil, db: dbTest, cache: mockCache}

	mockCache.EXPECT().FileDel(ctx, fileID).Return(errCacheTest)
	if err := h.FileDelete(ctx, fileID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	// deleting a row that does not exist reports not found and leaves the cache alone.
	if err := h.FileDelete(ctx, uuid.Must(uuid.NewV4())); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
	if err := h.AccountDelete(ctx, uuid.Must(uuid.NewV4())); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func Test_Update_withoutFieldsDoesNothing(t *testing.T) {
	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	// the closed database and the strict mocks prove that no statement and no cache call happens.
	h := &handler{util: utilhandler.NewMockUtilHandler(mc), db: newClosedDB(t), cache: cachehandler.NewMockCacheHandler(mc)}

	if err := h.FileUpdate(ctx, uuid.Must(uuid.NewV4()), map[file.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := h.AccountUpdate(ctx, uuid.Must(uuid.NewV4()), map[account.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
