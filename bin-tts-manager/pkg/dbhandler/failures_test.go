package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/pkg/cachehandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/mock/gomock"
)

var errTestRow = errors.New("test row failure")

func newFailureTestHandler(t *testing.T) (*dbHandler, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create mock: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	return &dbHandler{
		db:    db,
		util:  utilhandler.NewUtilHandler(),
		cache: cachehandler.NewMockCacheHandler(mc),
	}, mock
}

func Test_speakingGetFromRow_columnMismatch(t *testing.T) {
	h, mock := newFailureTestHandler(t)

	// a result set that does not carry the columns of the model cannot be scanned.
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"unknown_column"}).AddRow("value"))
	rows, err := h.db.Query("SELECT * FROM test")
	if err != nil {
		t.Fatalf("could not query: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		t.Fatal("expected a row")
	}
	res, err := h.speakingGetFromRow(rows)
	if res != nil || err == nil || !strings.Contains(err.Error(), "could not scan the row") {
		t.Errorf("Wrong result. res: %v, err: %v", res, err)
	}
}

func Test_SpeakingGet_rowError(t *testing.T) {
	h, mock := newFailureTestHandler(t)

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow([]byte("x")).RowError(0, errTestRow))

	res, err := h.SpeakingGet(context.Background(), uuid.Must(uuid.NewV4()))
	if res != nil || !errors.Is(err, errTestRow) {
		t.Fatalf("Wrong result. res: %v, err: %v", res, err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("a row error must not be reported as not found")
	}
}

func Test_SpeakingGets_scanError(t *testing.T) {
	h, mock := newFailureTestHandler(t)

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"unknown_column"}).AddRow("value"))

	res, err := h.SpeakingGets(context.Background(), "", 10, nil)
	if res != nil || err == nil || !strings.Contains(err.Error(), "could not scan row") {
		t.Errorf("Wrong result. res: %v, err: %v", res, err)
	}
}

func Test_SpeakingGets_rowError(t *testing.T) {
	h, mock := newFailureTestHandler(t)

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow([]byte("x")).RowError(0, errTestRow))

	res, err := h.SpeakingGets(context.Background(), "", 10, nil)
	if res != nil || !errors.Is(err, errTestRow) {
		t.Errorf("Wrong result. res: %v, err: %v", res, err)
	}
}

func Test_streamingGetFromCache_errors(t *testing.T) {
	id := uuid.FromStringOrNil("b6dcbc72-5af5-11f0-b0a7-fbe8aae41529")
	errCache := errors.New("cache unavailable")

	tests := []struct {
		name string

		cacheErr error

		expectErr error
	}{
		{name: "missing key becomes not found", cacheErr: redis.Nil, expectErr: ErrNotFound},
		{name: "other cache errors are propagated", cacheErr: errCache, expectErr: errCache},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &dbHandler{cache: mockCache}

			mockCache.EXPECT().StreamingGet(gomock.Any(), id).Return(nil, tt.cacheErr)

			res, err := h.streamingGetFromCache(context.Background(), id)
			if res != nil || !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong result. expect: %v, got res: %v, err: %v", tt.expectErr, res, err)
			}
			if errors.Is(tt.expectErr, errCache) && errors.Is(err, ErrNotFound) {
				t.Errorf("a cache failure must not be reported as not found")
			}
		})
	}
}
