package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/models/outdialtargetcall"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
)

var (
	errInjected = errors.New("injected failure")

	// unsupported is a value that can be neither marshaled for a column nor used as a filter.
	unsupported = []any{make(chan int)}
)

// newTestID returns a fresh id: the sqlite test database is shared and keeps its rows between runs
func newTestID() uuid.UUID { return uuid.Must(uuid.NewV4()) }

const testToken = "2026-01-01 00:00:00.000000"

func Test_ErrorPaths_Exec(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b0a6d7e-2c11-11f1-9d0e-0f1d2c3b4a59")

	tests := []struct {
		name  string
		regex string
		fn    func(h *handler) error
	}{
		{"outdial create", "(?i)insert into outdial_outdials", func(h *handler) error {
			return h.OutdialCreate(ctx, &outdial.Outdial{Identity: commonidentity.Identity{ID: id}})
		}},
		{"outdial update", "(?i)update outdial_outdials", func(h *handler) error {
			return h.OutdialUpdate(ctx, id, map[outdial.Field]any{outdial.FieldName: "n"})
		}},
		{"outdial delete", "(?i)update outdial_outdials", func(h *handler) error { return h.OutdialDelete(ctx, id) }},
		{"outdialtarget create", "(?i)insert into outdial_outdialtargets", func(h *handler) error {
			return h.OutdialTargetCreate(ctx, &outdialtarget.OutdialTarget{ID: id})
		}},
		{"outdialtarget update", "(?i)update outdial_outdialtargets", func(h *handler) error {
			return h.OutdialTargetUpdate(ctx, id, map[outdialtarget.Field]any{outdialtarget.FieldName: "n"})
		}},
		{"outdialtarget delete", "(?i)update outdial_outdialtargets", func(h *handler) error { return h.OutdialTargetDelete(ctx, id) }},
		{"outdialtargetcall create", "(?i)insert into outdial_outdialtargetcalls", func(h *handler) error {
			return h.OutdialTargetCallCreate(ctx, &outdialtargetcall.OutdialTargetCall{Identity: commonidentity.Identity{ID: id}})
		}},
		{"outdialtargetcall update", "(?i)update outdial_outdialtargetcalls", func(h *handler) error {
			return h.OutdialTargetCallUpdate(ctx, id, map[outdialtargetcall.Field]any{outdialtargetcall.FieldStatus: "done"})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the helper cache mock has no expectation for the writes, so a cache call fails the test
			h, mock := newMockHandler(t, mc)
			mock.ExpectExec(tt.regex).WillReturnError(errInjected)

			err := tt.fn(h)
			if !errors.Is(err, errInjected) {
				t.Errorf("expected the exec failure in the error chain, got: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unmet expectations: %v", errMet)
			}
		})
	}
}

func Test_ErrorPaths_Query(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b0a6d7e-2c12-11f1-9d0e-0f1d2c3b4a59")

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{"get outdial", func(h *handler) error { _, err := h.outdialGetFromDB(ctx, id); return err }},
		{"get outdialtarget", func(h *handler) error { _, err := h.outdialTargetGetFromDB(ctx, id); return err }},
		{"get outdialtargetcall", func(h *handler) error { _, err := h.outdialTargetCallGetFromDB(ctx, id); return err }},
		{"get outdialtargetcall by reference id", func(h *handler) error {
			_, err := h.OutdialTargetCallGetByReferenceID(ctx, id)
			return err
		}},
		{"get outdialtargetcall by activeflow id", func(h *handler) error {
			_, err := h.OutdialTargetCallGetByActiveflowID(ctx, id)
			return err
		}},
		{"list outdial", func(h *handler) error {
			_, err := h.OutdialList(ctx, testToken, 10, map[outdial.Field]any{})
			return err
		}},
		{"list outdialtarget", func(h *handler) error {
			_, err := h.OutdialTargetList(ctx, testToken, 10, map[outdialtarget.Field]any{})
			return err
		}},
		{"list outdialtargetcall", func(h *handler) error {
			_, err := h.OutdialTargetCallList(ctx, testToken, 10, map[outdialtargetcall.Field]any{})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mock := newMockHandler(t, mc)
			mock.ExpectQuery("(?i)select").WillReturnError(errInjected)

			err := tt.fn(h)
			if !errors.Is(err, errInjected) {
				t.Errorf("expected the query failure in the error chain, got: %v", err)
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("a query failure must not be reported as ErrNotFound. err: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unmet expectations: %v", errMet)
			}
		})
	}
}

func Test_ErrorPaths_NotFoundAndScan(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b0a6d7e-2c13-11f1-9d0e-0f1d2c3b4a59")

	tests := []struct {
		name string
		fn   func(h *handler) error
		// scanName is the text expected in the scan error, empty for the functions that
		// return the error of the row conversion unchanged.
		scanName string
	}{
		{"outdial", func(h *handler) error { _, err := h.outdialGetFromDB(ctx, id); return err }, "outdialGetFromDB"},
		{"outdialtarget", func(h *handler) error { _, err := h.outdialTargetGetFromDB(ctx, id); return err }, "outdialTargetGetFromDB"},
		{"outdialtargetcall", func(h *handler) error { _, err := h.outdialTargetCallGetFromDB(ctx, id); return err }, "outdialTargetCallGetFromDB"},
		{"outdialtargetcall by reference id", func(h *handler) error {
			_, err := h.OutdialTargetCallGetByReferenceID(ctx, id)
			return err
		}, "outdialTargetCallGetFromRow"},
		{"outdialtargetcall by activeflow id", func(h *handler) error {
			_, err := h.OutdialTargetCallGetByActiveflowID(ctx, id)
			return err
		}, "outdialTargetCallGetFromRow"},
	}

	for _, tt := range tests {
		t.Run(tt.name+" not found", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mock := newMockHandler(t, mc)
			mock.ExpectQuery("(?i)select").WillReturnRows(sqlmock.NewRows([]string{"id"}))

			if err := tt.fn(h); !errors.Is(err, ErrNotFound) {
				t.Errorf("expected ErrNotFound, got: %v", err)
			}
		})

		t.Run(tt.name+" scan error", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// one column where the model needs many: the scan fails
			h, mock := newMockHandler(t, mc)
			mock.ExpectQuery("(?i)select").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()))

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected a scan error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("a scan failure must not be reported as ErrNotFound. err: %v", err)
			}
			if !strings.Contains(err.Error(), tt.scanName) || !strings.Contains(err.Error(), "scan") {
				t.Errorf("expected a scan error naming %s, got: %v", tt.scanName, err)
			}
		})
	}
}

func Test_ErrorPaths_List(t *testing.T) {
	ctx := context.Background()

	listFns := []struct {
		name string
		fn   func(h *handler, token string, unsupportedFilter bool) ([]any, error)
	}{
		{"outdial", func(h *handler, token string, bad bool) ([]any, error) {
			filters := map[outdial.Field]any{}
			if bad {
				filters[outdial.FieldName] = unsupported
			}
			res, err := h.OutdialList(ctx, token, 10, filters)
			return make([]any, len(res)), err
		}},
		{"outdialtarget", func(h *handler, token string, bad bool) ([]any, error) {
			filters := map[outdialtarget.Field]any{}
			if bad {
				filters[outdialtarget.FieldName] = unsupported
			}
			res, err := h.OutdialTargetList(ctx, token, 10, filters)
			return make([]any, len(res)), err
		}},
		{"outdialtargetcall", func(h *handler, token string, bad bool) ([]any, error) {
			filters := map[outdialtargetcall.Field]any{}
			if bad {
				filters[outdialtargetcall.FieldStatus] = unsupported
			}
			res, err := h.OutdialTargetCallList(ctx, token, 10, filters)
			return make([]any, len(res)), err
		}},
	}

	for _, tt := range listFns {
		t.Run(tt.name+" unsupported filter", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no query is expected: sqlmock would fail an unexpected one
			h, mock := newMockHandler(t, mc)

			_, err := tt.fn(h, testToken, true)
			if err == nil || !strings.Contains(err.Error(), "could not apply filters") {
				t.Errorf("expected an apply filters error, got: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unmet expectations: %v", errMet)
			}
		})

		t.Run(tt.name+" empty token uses the current time", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the helper time handler returns testToken as the current time
			h, mock := newMockHandler(t, mc)
			mock.ExpectQuery("(?i)select").WithArgs(testToken).WillReturnRows(sqlmock.NewRows([]string{"id"}))

			res, err := tt.fn(h, "", false)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if len(res) != 0 {
				t.Errorf("expected an empty list, got %d items", len(res))
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("the current time was not bound as the token: %v", errMet)
			}
		})

		t.Run(tt.name+" scan error", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mock := newMockHandler(t, mc)
			mock.ExpectQuery("(?i)select").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow([]byte("x")))

			if _, err := tt.fn(h, testToken, false); err == nil || !strings.Contains(err.Error(), "could not get data") {
				t.Errorf("expected a could not get data error, got: %v", err)
			}
		})
	}
}

func Test_ErrorPaths_Update(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b0a6d7e-2c14-11f1-9d0e-0f1d2c3b4a59")

	tests := []struct {
		name  string
		empty func(h *handler) error
		bad   func(h *handler) error
	}{
		{"outdial",
			func(h *handler) error { return h.OutdialUpdate(ctx, id, map[outdial.Field]any{}) },
			func(h *handler) error {
				return h.OutdialUpdate(ctx, id, map[outdial.Field]any{outdial.FieldName: unsupported})
			}},
		{"outdialtarget",
			func(h *handler) error { return h.OutdialTargetUpdate(ctx, id, map[outdialtarget.Field]any{}) },
			func(h *handler) error {
				return h.OutdialTargetUpdate(ctx, id, map[outdialtarget.Field]any{outdialtarget.FieldName: unsupported})
			}},
		{"outdialtargetcall",
			func(h *handler) error { return h.OutdialTargetCallUpdate(ctx, id, map[outdialtargetcall.Field]any{}) },
			func(h *handler) error {
				return h.OutdialTargetCallUpdate(ctx, id, map[outdialtargetcall.Field]any{outdialtargetcall.FieldStatus: unsupported})
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name+" empty fields", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// neither SQL nor a cache call is expected
			h, mock := newMockHandler(t, mc)
			if err := tt.empty(h); err != nil {
				t.Errorf("expected nil for empty fields, got: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unmet expectations: %v", errMet)
			}
		})

		t.Run(tt.name+" fields that cannot be prepared", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mock := newMockHandler(t, mc)
			err := tt.bad(h)
			if err == nil || !strings.Contains(err.Error(), "prepare fields failed") {
				t.Errorf("expected a prepare fields error, got: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unmet expectations: %v", errMet)
			}
		})
	}
}

// Test_CacheBranches runs the cache branches against the sqlite test database
// with its own cache mock. Every case uses ids of its own.
func Test_CacheBranches(t *testing.T) {
	ctx := context.Background()

	newHandler := func(mc *gomock.Controller) (*handler, *cachehandler.MockCacheHandler) {
		mockCache := cachehandler.NewMockCacheHandler(mc)
		return &handler{db: dbTest, cache: mockCache, utilHandler: utilhandler.NewUtilHandler()}, mockCache
	}

	t.Run("outdial: the writes succeed when the cache refresh fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCache := newHandler(mc)
		id := newTestID()

		mockCache.EXPECT().OutdialSet(ctx, gomock.Any()).Return(errInjected).Times(3)
		if err := h.OutdialCreate(ctx, &outdial.Outdial{Identity: commonidentity.Identity{ID: id}, Name: "a"}); err != nil {
			t.Errorf("create: expected nil, got: %v", err)
		}
		if err := h.OutdialUpdate(ctx, id, map[outdial.Field]any{outdial.FieldName: "b"}); err != nil {
			t.Errorf("update: expected nil, got: %v", err)
		}
		if err := h.OutdialDelete(ctx, id); err != nil {
			t.Errorf("delete: expected nil, got: %v", err)
		}
	})

	t.Run("outdial: get hit, miss and cache write failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCache := newHandler(mc)
		id := newTestID()

		cached := &outdial.Outdial{Identity: commonidentity.Identity{ID: id}, Name: "cached"}
		mockCache.EXPECT().OutdialGet(ctx, id).Return(cached, nil)
		res, err := h.OutdialGet(ctx, id)
		if err != nil || res != cached {
			t.Errorf("hit: expected the cached object, got: %v, %v", res, err)
		}

		mockCache.EXPECT().OutdialSet(ctx, gomock.Any()).Return(nil)
		if errCreate := h.OutdialCreate(ctx, &outdial.Outdial{Identity: commonidentity.Identity{ID: id}, Name: "stored"}); errCreate != nil {
			t.Fatalf("create: %v", errCreate)
		}

		mockCache.EXPECT().OutdialGet(ctx, id).Return(nil, errInjected)
		mockCache.EXPECT().OutdialSet(ctx, gomock.Any()).Return(errInjected)
		res, err = h.OutdialGet(ctx, id)
		if err != nil || res == nil || res.Name != "stored" {
			t.Errorf("miss with a failing cache write: expected the stored row, got: %v, %v", res, err)
		}

		missing := newTestID()
		mockCache.EXPECT().OutdialGet(ctx, missing).Return(nil, errInjected)
		if _, err = h.OutdialGet(ctx, missing); !errors.Is(err, ErrNotFound) {
			t.Errorf("miss without a row: expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("outdialtarget: the writes succeed when the cache refresh fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCache := newHandler(mc)
		id := newTestID()

		mockCache.EXPECT().OutdialTargetSet(ctx, gomock.Any()).Return(errInjected).Times(3)
		if err := h.OutdialTargetCreate(ctx, &outdialtarget.OutdialTarget{ID: id, Name: "a"}); err != nil {
			t.Errorf("create: expected nil, got: %v", err)
		}
		if err := h.OutdialTargetUpdate(ctx, id, map[outdialtarget.Field]any{outdialtarget.FieldName: "b"}); err != nil {
			t.Errorf("update: expected nil, got: %v", err)
		}
		if err := h.OutdialTargetDelete(ctx, id); err != nil {
			t.Errorf("delete: expected nil, got: %v", err)
		}
	})

	t.Run("outdialtarget: get hit, miss and cache write failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCache := newHandler(mc)
		id := newTestID()

		cached := &outdialtarget.OutdialTarget{ID: id, Name: "cached"}
		mockCache.EXPECT().OutdialTargetGet(ctx, id).Return(cached, nil)
		res, err := h.OutdialTargetGet(ctx, id)
		if err != nil || res != cached {
			t.Errorf("hit: expected the cached object, got: %v, %v", res, err)
		}

		mockCache.EXPECT().OutdialTargetSet(ctx, gomock.Any()).Return(nil)
		if errCreate := h.OutdialTargetCreate(ctx, &outdialtarget.OutdialTarget{ID: id, Name: "stored"}); errCreate != nil {
			t.Fatalf("create: %v", errCreate)
		}

		mockCache.EXPECT().OutdialTargetGet(ctx, id).Return(nil, errInjected)
		mockCache.EXPECT().OutdialTargetSet(ctx, gomock.Any()).Return(errInjected)
		res, err = h.OutdialTargetGet(ctx, id)
		if err != nil || res == nil || res.Name != "stored" {
			t.Errorf("miss with a failing cache write: expected the stored row, got: %v, %v", res, err)
		}

		missing := newTestID()
		mockCache.EXPECT().OutdialTargetGet(ctx, missing).Return(nil, errInjected)
		if _, err = h.OutdialTargetGet(ctx, missing); !errors.Is(err, ErrNotFound) {
			t.Errorf("miss without a row: expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("outdialtargetcall: the writes succeed when the cache refresh fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCache := newHandler(mc)
		id := newTestID()

		mockCache.EXPECT().OutdialTargetCallSet(ctx, gomock.Any()).Return(errInjected).Times(2)
		if err := h.OutdialTargetCallCreate(ctx, &outdialtargetcall.OutdialTargetCall{Identity: commonidentity.Identity{ID: id}}); err != nil {
			t.Errorf("create: expected nil, got: %v", err)
		}
		if err := h.OutdialTargetCallUpdate(ctx, id, map[outdialtargetcall.Field]any{outdialtargetcall.FieldStatus: "done"}); err != nil {
			t.Errorf("update: expected nil, got: %v", err)
		}
	})

	t.Run("outdialtargetcall: each cache write failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCache := newHandler(mc)

		call := &outdialtargetcall.OutdialTargetCall{
			Identity:     commonidentity.Identity{ID: newTestID()},
			ActiveflowID: newTestID(),
			ReferenceID:  newTestID(),
		}

		gomock.InOrder(
			mockCache.EXPECT().OutdialTargetCallSet(ctx, call).Return(errInjected),
			mockCache.EXPECT().OutdialTargetCallSet(ctx, call).Return(nil),
			mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, call).Return(errInjected),
			mockCache.EXPECT().OutdialTargetCallSet(ctx, call).Return(nil),
			mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, call).Return(nil),
			mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, call).Return(errInjected),
		)
		for i := 0; i < 3; i++ {
			if err := h.outdialTargetCallSetToCache(ctx, call); !errors.Is(err, errInjected) {
				t.Errorf("case %d: expected the cache failure, got: %v", i, err)
			}
		}
	})

	t.Run("outdialtargetcall: get hit, miss and cache write failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCache := newHandler(mc)

		id := newTestID()
		activeflowID := newTestID()
		referenceID := newTestID()

		cached := &outdialtargetcall.OutdialTargetCall{Identity: commonidentity.Identity{ID: id}, TryCount: 7}
		mockCache.EXPECT().OutdialTargetCallGet(ctx, id).Return(cached, nil)
		res, err := h.OutdialTargetCallGet(ctx, id)
		if err != nil || res != cached {
			t.Errorf("hit: expected the cached object, got: %v, %v", res, err)
		}
		mockCache.EXPECT().OutdialTargetCallGetByReferenceID(ctx, referenceID).Return(cached, nil)
		if res, err = h.OutdialTargetCallGetByReferenceID(ctx, referenceID); err != nil || res != cached {
			t.Errorf("by reference id hit: expected the cached object, got: %v, %v", res, err)
		}
		mockCache.EXPECT().OutdialTargetCallGetByActiveflowID(ctx, activeflowID).Return(cached, nil)
		if res, err = h.OutdialTargetCallGetByActiveflowID(ctx, activeflowID); err != nil || res != cached {
			t.Errorf("by activeflow id hit: expected the cached object, got: %v, %v", res, err)
		}

		mockCache.EXPECT().OutdialTargetCallSet(ctx, gomock.Any()).Return(nil)
		mockCache.EXPECT().OutdialTargetCallSetByActiveflowID(ctx, gomock.Any()).Return(nil)
		mockCache.EXPECT().OutdialTargetCallSetByReferenceID(ctx, gomock.Any()).Return(nil)
		stored := &outdialtargetcall.OutdialTargetCall{
			Identity:     commonidentity.Identity{ID: id},
			ActiveflowID: activeflowID,
			ReferenceID:  referenceID,
			TryCount:     3,
		}
		if errCreate := h.OutdialTargetCallCreate(ctx, stored); errCreate != nil {
			t.Fatalf("create: %v", errCreate)
		}

		// the cache misses and every write to it fails: the stored row is still returned
		mockCache.EXPECT().OutdialTargetCallGet(ctx, id).Return(nil, errInjected)
		mockCache.EXPECT().OutdialTargetCallSet(ctx, gomock.Any()).Return(errInjected)
		if res, err = h.OutdialTargetCallGet(ctx, id); err != nil || res == nil || res.TryCount != 3 {
			t.Errorf("miss with a failing cache write: expected the stored row, got: %v, %v", res, err)
		}

		mockCache.EXPECT().OutdialTargetCallGetByReferenceID(ctx, referenceID).Return(nil, errInjected)
		mockCache.EXPECT().OutdialTargetCallSet(ctx, gomock.Any()).Return(errInjected)
		if res, err = h.OutdialTargetCallGetByReferenceID(ctx, referenceID); err != nil || res == nil || res.ID != id {
			t.Errorf("by reference id miss: expected the stored row, got: %v, %v", res, err)
		}

		mockCache.EXPECT().OutdialTargetCallGetByActiveflowID(ctx, activeflowID).Return(nil, errInjected)
		mockCache.EXPECT().OutdialTargetCallSet(ctx, gomock.Any()).Return(errInjected)
		if res, err = h.OutdialTargetCallGetByActiveflowID(ctx, activeflowID); err != nil || res == nil || res.ID != id {
			t.Errorf("by activeflow id miss: expected the stored row, got: %v, %v", res, err)
		}

		missing := newTestID()
		mockCache.EXPECT().OutdialTargetCallGet(ctx, missing).Return(nil, errInjected)
		if _, err = h.OutdialTargetCallGet(ctx, missing); !errors.Is(err, ErrNotFound) {
			t.Errorf("miss without a row: expected ErrNotFound, got: %v", err)
		}
	})
}
