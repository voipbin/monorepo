package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/cachehandler"
)

// newShortRows returns a single row that has fewer columns than the tag model needs,
// which makes the row scan fail.
func newShortRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id"}).AddRow([]byte("x"))
}

func Test_tagGetFromDB_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT").WillReturnRows(newShortRows())

	h := handler{db: db}
	res, err := h.tagGetFromDB(context.Background(), uuid.FromStringOrNil("11111111-50d7-11ec-a6b1-8f9671a9e70e"))
	if err == nil {
		t.Fatalf("Expected an error, got nil. res: %v", res)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("A scan error must not be reported as ErrNotFound. err: %v", err)
	}
	if !strings.Contains(err.Error(), "could not scan the row. tagGetFromDB") {
		t.Errorf("Wrong error. got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected a nil result on error. got: %v", res)
	}
}

func Test_TagList_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT").WillReturnRows(newShortRows())

	h := handler{db: db}
	res, err := h.TagList(context.Background(), 10, utilhandler.TimeGetCurTime(), map[tag.Field]any{})
	if err == nil {
		t.Fatalf("Expected an error, got nil. res: %v", res)
	}
	if !strings.Contains(err.Error(), "could not scan the row. TagList") {
		t.Errorf("Wrong error. got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected a nil result on error. got: %v", res)
	}
}

func Test_tagUpdateToCache_Error(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-50d7-11ec-a6b1-8f9671a9e70e")
	cacheErr := errors.New("cache write failed")

	tests := []struct {
		name string

		rows      func() *sqlmock.Rows
		expectSet bool

		expectErr    error
		expectNotErr error
	}{
		{
			name:         "db_read_fails_before_the_cache_is_touched",
			rows:         func() *sqlmock.Rows { return sqlmock.NewRows(commondatabasehandler.GetDBFields(&tag.Tag{})) },
			expectSet:    false,
			expectErr:    ErrNotFound,
			expectNotErr: cacheErr,
		},
		{
			name: "cache_write_error_is_returned",
			rows: func() *sqlmock.Rows {
				columns := commondatabasehandler.GetDBFields(&tag.Tag{})
				return sqlmock.NewRows(columns).AddRow(make([]driver.Value, len(columns))...)
			},
			expectSet:    true,
			expectErr:    cacheErr,
			expectNotErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create sqlmock: %v", err)
			}
			defer func() { _ = db.Close() }()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{db: db, cache: mockCache}
			ctx := context.Background()

			mock.ExpectQuery("SELECT").WillReturnRows(tt.rows())
			if tt.expectSet {
				mockCache.EXPECT().TagSet(ctx, gomock.Any()).Return(cacheErr)
			}

			err = h.tagUpdateToCache(ctx, id)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
			if errors.Is(err, tt.expectNotErr) {
				t.Errorf("Unexpected error cause. got: %v", err)
			}
		})
	}
}
