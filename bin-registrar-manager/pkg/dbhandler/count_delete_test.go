package dbhandler

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-registrar-manager/models/astcontact"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

func Test_CountByCustomerID(t *testing.T) {
	customerID := uuid.FromStringOrNil("77030aee-7fec-11ec-9fc4-0fa126e45204")
	errQuery := errors.New("query failed")

	tests := []struct {
		name  string
		table string
		call  func(h *handler) (int, error)
	}{
		{
			name:  "extensions",
			table: extensionsTable,
			call: func(h *handler) (int, error) {
				return h.ExtensionCountByCustomerID(context.Background(), customerID)
			},
		},
		{
			name:  "trunks",
			table: trunksTable,
			call: func(h *handler) (int, error) {
				return h.TrunkCountByCustomerID(context.Background(), customerID)
			},
		},
	}

	for _, tt := range tests {
		// the query must count only rows that are not soft-deleted and only the given customer.
		query := "^" + regexp.QuoteMeta("SELECT COUNT(*) FROM "+tt.table+" WHERE customer_id = ? AND tm_delete IS NULL") + "$"

		cases := []struct {
			name      string
			rows      *sqlmock.Rows
			queryErr  error
			wantCount int
			wantErr   bool
		}{
			{name: "count", rows: sqlmock.NewRows([]string{"count"}).AddRow(3), wantCount: 3},
			{name: "zero rows boundary", rows: sqlmock.NewRows([]string{"count"}).AddRow(0), wantCount: 0},
			{name: "query error", queryErr: errQuery, wantErr: true},
			{name: "scan error", rows: sqlmock.NewRows([]string{"count"}).AddRow("not-a-number"), wantErr: true},
		}

		for _, c := range cases {
			t.Run(tt.name+" "+c.name, func(t *testing.T) {
				db, mock, err := sqlmock.New() // default regexp matcher
				if err != nil {
					t.Fatalf("could not create sqlmock: %v", err)
				}
				defer func() { _ = db.Close() }()

				h := &handler{db: db}

				exp := mock.ExpectQuery(query).WithArgs(customerID.Bytes())
				if c.queryErr != nil {
					exp.WillReturnError(c.queryErr)
				} else {
					exp.WillReturnRows(c.rows)
				}

				count, err := tt.call(h)
				if (err != nil) != c.wantErr {
					t.Fatalf("error = %v, wantErr %v", err, c.wantErr)
				}
				if c.queryErr != nil && !errors.Is(err, c.queryErr) {
					t.Errorf("error = %v, want it to wrap the query error", err)
				}
				if count != c.wantCount {
					t.Errorf("count = %d, want %d", count, c.wantCount)
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("unfulfilled expectations: %v", err)
				}
			})
		}
	}
}

func Test_TrunkDelete_softDelete(t *testing.T) {
	id := uuid.FromStringOrNil("cbf0af5e-519e-11ee-a4c4-9f155401d234")
	now := time.Date(2023, 1, 3, 21, 35, 2, 0, time.UTC)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{db: db, utilHandler: mockUtil, cache: mockCache}

	mockUtil.EXPECT().TimeNow().Return(&now)
	// the row is soft-deleted (tm_delete set), never removed, and only the given id is touched.
	mock.ExpectExec("^"+regexp.QuoteMeta("UPDATE "+trunksTable+" SET tm_delete = ?, tm_update = ? WHERE id = ?")+"$").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), id.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// the cache refresh after the delete reads the row back; its failure is ignored by TrunkDelete.
	mock.ExpectQuery("(?is)select.*from.*" + trunksTable).WillReturnError(errors.New("refresh failed"))

	if err := h.TrunkDelete(context.Background(), id); err != nil {
		t.Errorf("TrunkDelete() error = %v, want nil even when the cache refresh fails", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func Test_TrunkDelete_execError(t *testing.T) {
	id := uuid.FromStringOrNil("cbf0af5e-519e-11ee-a4c4-9f155401d234")
	now := time.Date(2023, 1, 3, 21, 35, 2, 0, time.UTC)
	errExec := errors.New("exec failed")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	// no cache expectation: a failed delete must not touch the cache.
	h := &handler{db: db, utilHandler: mockUtil, cache: cachehandler.NewMockCacheHandler(mc)}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mock.ExpectExec("(?i)^update " + trunksTable).WillReturnError(errExec)

	if err := h.TrunkDelete(context.Background(), id); !errors.Is(err, errExec) {
		t.Errorf("TrunkDelete() error = %v, want the exec error", err)
	}
}

func Test_AstContactCacheErrors(t *testing.T) {
	errCache := errors.New("cache failed")
	endpoint := "1001@realm.example.invalid"
	contacts := []*astcontact.AstContact{{ID: "contact-a", Endpoint: endpoint}}

	tests := []struct {
		name string
		call func(h *handler, mockCache *cachehandler.MockCacheHandler) error
	}{
		{"set error", func(h *handler, mockCache *cachehandler.MockCacheHandler) error {
			mockCache.EXPECT().AstContactsSet(gomock.Any(), endpoint, contacts).Return(errCache)
			return h.AstContactsSetToCache(context.Background(), endpoint, contacts)
		}},
		{"delete error", func(h *handler, mockCache *cachehandler.MockCacheHandler) error {
			mockCache.EXPECT().AstContactsDel(gomock.Any(), endpoint).Return(errCache)
			return h.AstContactDeleteFromCache(context.Background(), endpoint)
		}},
		{"gets error", func(h *handler, mockCache *cachehandler.MockCacheHandler) error {
			mockCache.EXPECT().AstContactsGet(gomock.Any(), endpoint).Return(nil, errCache)
			_, err := h.AstContactGetsFromCache(context.Background(), endpoint)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{cache: mockCache}

			if err := tt.call(h, mockCache); !errors.Is(err, errCache) {
				t.Errorf("error = %v, want the cache error", err)
			}
		})
	}

	t.Run("success paths", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockCache := cachehandler.NewMockCacheHandler(mc)
		h := &handler{cache: mockCache}

		mockCache.EXPECT().AstContactsSet(gomock.Any(), endpoint, contacts).Return(nil)
		mockCache.EXPECT().AstContactsDel(gomock.Any(), endpoint).Return(nil)

		if err := h.AstContactsSetToCache(context.Background(), endpoint, contacts); err != nil {
			t.Errorf("AstContactsSetToCache() error = %v", err)
		}
		if err := h.AstContactDeleteFromCache(context.Background(), endpoint); err != nil {
			t.Errorf("AstContactDeleteFromCache() error = %v", err)
		}
	})
}
