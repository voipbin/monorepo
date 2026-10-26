package dbhandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/cachehandler"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// newMockedHandler returns a handler whose database is a sqlmock, so that the result sets
// can be shaped freely. The cache is a gomock cache.
func newMockedHandler(t *testing.T, mc *gomock.Controller) (*handler, sqlmock.Sqlmock, *cachehandler.MockCacheHandler) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create the sqlmock. err: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       mockCache,
	}

	return h, mock, mockCache
}

func Test_pipecatcallGetFromDB_rowErrors(t *testing.T) {
	id := uuid.FromStringOrNil("5a1e0c1e-6a52-4b6e-9d0a-2f7a1c000001")

	tests := []struct {
		name string

		rows func() *sqlmock.Rows

		expectNotFound bool
	}{
		{
			name: "scan fails on a row with the wrong columns",
			rows: func() *sqlmock.Rows {
				return sqlmock.NewRows([]string{"unknown_column"}).AddRow("x")
			},
		},
		{
			name: "iteration error before the first row is reported",
			rows: func() *sqlmock.Rows {
				return sqlmock.NewRows([]string{"id"}).AddRow([]byte("x")).RowError(0, errors.New("iteration failed"))
			},
		},
		{
			name: "no row is not found",
			rows: func() *sqlmock.Rows {
				return sqlmock.NewRows([]string{"id"})
			},
			expectNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mock, _ := newMockedHandler(t, mc)
			mock.ExpectQuery("SELECT").WillReturnRows(tt.rows())

			res, err := h.pipecatcallGetFromDB(context.Background(), id)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if tt.expectNotFound != errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. expect not found: %v, got: %v", tt.expectNotFound, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func Test_PipecatcallGet_scanFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("5a1e0c1e-6a52-4b6e-9d0a-2f7a1c000002")
	h, mock, mockCache := newMockedHandler(t, mc)

	// the cache misses and the database row can not be scanned; nothing is written to the cache
	mockCache.EXPECT().PipecatcallGet(gomock.Any(), id).Return(nil, errors.New("cache miss"))
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"unknown_column"}).AddRow("x"))

	res, err := h.PipecatcallGet(context.Background(), id)
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: error and nil, got: %v, %v", err, res)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: an error other than not found, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_PipecatcallCreate_cacheRefreshReadFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("5a1e0c1e-6a52-4b6e-9d0a-2f7a1c000003")
	h, mock, _ := newMockedHandler(t, mc)

	// the insert succeeds and the read for the cache refresh fails: the create still succeeds and
	// nothing is written to the cache (the gomock cache has no expectation).
	mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT").WillReturnError(errors.New("read failed"))

	if err := h.PipecatcallCreate(context.Background(), &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{ID: id},
	}); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_PipecatcallDelete_rowsAffectedFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	id := uuid.FromStringOrNil("5a1e0c1e-6a52-4b6e-9d0a-2f7a1c000004")
	h, mock, _ := newMockedHandler(t, mc)

	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected unsupported")))

	err := h.PipecatcallDelete(context.Background(), id)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: an error other than not found, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
