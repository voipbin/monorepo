package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
)

// The tests in this file use a scripted driver for the failures that the
// shared in-memory database cannot produce: a failing RowsAffected, a failing
// row iteration and a row that does not match the model. Every test creates
// its own mock database, so none of them touches the shared test database.

var (
	testSQLMockCustomerID  = uuid.FromStringOrNil("3f6a9b10-0000-4000-8000-0000000000b1")
	testSQLMockAccesskeyID = uuid.FromStringOrNil("3f6a9b10-0000-4000-8000-0000000000b2")
	errTestSQLMock         = errors.New("scripted driver failure")
)

func newSQLMockHandler(t *testing.T, cache cachehandler.CacheHandler) (*handler, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create the mock database. err: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}, mock
}

// mismatchedRows returns one row with a single column, which cannot be scanned into a model.
func mismatchedRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id"}).AddRow([]byte{0x01})
}

// storedRows returns one row of the model's columns: the id and the name are set,
// every other value is NULL.
func storedRows(model any, id uuid.UUID, name string) *sqlmock.Rows {
	cols := commondatabasehandler.GetDBFields(model)
	vals := make([]driver.Value, len(cols))
	for i, col := range cols {
		switch col {
		case "id":
			vals[i] = id.Bytes()
		case "name":
			vals[i] = name
		}
	}
	return sqlmock.NewRows(cols).AddRow(vals...)
}

func Test_sqlmock_rowsAffectedError(t *testing.T) {
	tests := []struct {
		name   string
		expect string
		call   func(h *handler) error
	}{
		{
			name:   "customer freeze",
			expect: "UPDATE customer_customers",
			call: func(h *handler) error {
				return h.CustomerFreeze(context.Background(), testSQLMockCustomerID)
			},
		},
		{
			name:   "customer recover",
			expect: "UPDATE customer_customers",
			call: func(h *handler) error {
				return h.CustomerRecover(context.Background(), testSQLMockCustomerID)
			},
		},
		{
			name:   "customer anonymize",
			expect: "UPDATE customer_customers",
			call: func(h *handler) error {
				return h.CustomerAnonymizePII(context.Background(), testSQLMockCustomerID, "anon", "anon@example.com")
			},
		},
		{
			name:   "customer delete",
			expect: "UPDATE customer_customers",
			call: func(h *handler) error {
				return h.CustomerDelete(context.Background(), testSQLMockCustomerID)
			},
		},
		{
			name:   "accesskey delete",
			expect: "UPDATE customer_accesskeys",
			call: func(h *handler) error {
				return h.AccesskeyDelete(context.Background(), testSQLMockAccesskeyID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no cache expectation: a failed update must not touch the cache
			h, mock := newSQLMockHandler(t, cachehandler.NewMockCacheHandler(mc))
			mock.ExpectExec(tt.expect).WillReturnResult(sqlmock.NewErrorResult(errTestSQLMock))

			err := tt.call(h)
			if !errors.Is(err, errTestSQLMock) {
				t.Fatalf("Wrong match. expect: %v, got: %v", errTestSQLMock, err)
			}
			if !strings.Contains(err.Error(), "could not get rows affected") {
				t.Errorf("Wrong match. expect the message to name the failed step, got: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("Wrong match. expect all scripted calls to be used, got: %v", errMet)
			}
		})
	}
}

func Test_sqlmock_getErrors(t *testing.T) {
	tests := []struct {
		name         string
		useAccesskey bool
		rows         *sqlmock.Rows

		expectIs     error
		expectPhrase string
	}{
		{
			name:         "customer row iteration error",
			rows:         sqlmock.NewRows([]string{"id"}).AddRow([]byte{0x01}).RowError(0, errTestSQLMock),
			expectIs:     errTestSQLMock,
			expectPhrase: "row iteration error. customerGetFromDB",
		},
		{
			name:         "customer row does not match the model",
			rows:         mismatchedRows(),
			expectPhrase: "could not get data from row. customerGetFromDB",
		},
		{
			name:         "customer not found",
			rows:         sqlmock.NewRows([]string{"id"}),
			expectIs:     ErrNotFound,
			expectPhrase: "",
		},
		{
			name:         "accesskey row iteration error",
			useAccesskey: true,
			rows:         sqlmock.NewRows([]string{"id"}).AddRow([]byte{0x01}).RowError(0, errTestSQLMock),
			expectIs:     errTestSQLMock,
			expectPhrase: "row iteration error. accesskeyGetFromDB",
		},
		{
			name:         "accesskey row does not match the model",
			useAccesskey: true,
			rows:         mismatchedRows(),
			expectPhrase: "could not get data from row. accesskeyGetFromDB",
		},
		{
			name:         "accesskey not found",
			useAccesskey: true,
			rows:         sqlmock.NewRows([]string{"id"}),
			expectIs:     ErrNotFound,
			expectPhrase: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h, mock := newSQLMockHandler(t, mockCache)

			var err error
			if tt.useAccesskey {
				mockCache.EXPECT().AccesskeyGet(gomock.Any(), testSQLMockAccesskeyID).Return(nil, fmt.Errorf("cache miss"))
				mock.ExpectQuery("SELECT .* FROM customer_accesskeys").WillReturnRows(tt.rows)
				_, err = h.AccesskeyGet(context.Background(), testSQLMockAccesskeyID)
			} else {
				mockCache.EXPECT().CustomerGet(gomock.Any(), testSQLMockCustomerID).Return(nil, fmt.Errorf("cache miss"))
				mock.ExpectQuery("SELECT .* FROM customer_customers").WillReturnRows(tt.rows)
				_, err = h.CustomerGet(context.Background(), testSQLMockCustomerID)
			}

			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIs, err)
			}
			if !strings.Contains(err.Error(), tt.expectPhrase) {
				t.Errorf("Wrong match. expect the message to contain %q, got: %v", tt.expectPhrase, err)
			}
		})
	}
}

func Test_sqlmock_listErrors(t *testing.T) {
	curTime := time.Date(2024, 4, 18, 3, 22, 17, 0, time.UTC)

	tests := []struct {
		name   string
		table  string
		rows   *sqlmock.Rows
		call   func(h *handler) error
		expect string

		expectIs error
	}{
		{
			name:   "customer list, row does not match the model",
			table:  "customer_customers",
			rows:   mismatchedRows(),
			expect: "could not get data. CustomerGets",
			call: func(h *handler) error {
				_, err := h.CustomerList(context.Background(), 10, "2024-01-01T00:00:00.000000Z", map[customer.Field]any{})
				return err
			},
		},
		{
			name:     "customer list, row iteration error",
			table:    "customer_customers",
			rows:     sqlmock.NewRows([]string{"id"}).AddRow([]byte{0x01}).RowError(0, errTestSQLMock),
			expect:   "rows iteration error. CustomerGets",
			expectIs: errTestSQLMock,
			call: func(h *handler) error {
				_, err := h.CustomerList(context.Background(), 10, "2024-01-01T00:00:00.000000Z", map[customer.Field]any{})
				return err
			},
		},
		{
			name:   "accesskey list, row does not match the model",
			table:  "customer_accesskeys",
			rows:   mismatchedRows(),
			expect: "could not get data. AccesskeyGets",
			call: func(h *handler) error {
				_, err := h.AccesskeyList(context.Background(), 10, "2024-01-01T00:00:00.000000Z", map[accesskey.Field]any{})
				return err
			},
		},
		{
			name:     "accesskey list, row iteration error",
			table:    "customer_accesskeys",
			rows:     sqlmock.NewRows([]string{"id"}).AddRow([]byte{0x01}).RowError(0, errTestSQLMock),
			expect:   "rows iteration error. AccesskeyGets",
			expectIs: errTestSQLMock,
			call: func(h *handler) error {
				_, err := h.AccesskeyList(context.Background(), 10, "2024-01-01T00:00:00.000000Z", map[accesskey.Field]any{})
				return err
			},
		},
		{
			name:   "frozen expired list, row does not match the model",
			table:  "customer_customers",
			rows:   mismatchedRows(),
			expect: "could not get data. CustomerListFrozenExpired",
			call: func(h *handler) error {
				_, err := h.CustomerListFrozenExpired(context.Background(), curTime)
				return err
			},
		},
		{
			name:     "frozen expired list, row iteration error",
			table:    "customer_customers",
			rows:     sqlmock.NewRows([]string{"id"}).AddRow([]byte{0x01}).RowError(0, errTestSQLMock),
			expect:   "rows iteration error. CustomerListFrozenExpired",
			expectIs: errTestSQLMock,
			call: func(h *handler) error {
				_, err := h.CustomerListFrozenExpired(context.Background(), curTime)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mock := newSQLMockHandler(t, cachehandler.NewMockCacheHandler(mc))
			mock.ExpectQuery("SELECT .* FROM " + tt.table).WillReturnRows(tt.rows)

			err := tt.call(h)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("Wrong match. expect the message to contain %q, got: %v", tt.expect, err)
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}

// Test_sqlmock_cacheHit checks that a cache hit is returned as is. The scripted
// query is never consumed: a lookup in the database after a hit would use it up.
func Test_sqlmock_cacheHit(t *testing.T) {
	t.Run("customer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockCache := cachehandler.NewMockCacheHandler(mc)
		h, mock := newSQLMockHandler(t, mockCache)
		mock.ExpectQuery("SELECT .* FROM customer_customers").WillReturnRows(mismatchedRows())

		cached := &customer.Customer{Name: "cached customer"}
		mockCache.EXPECT().CustomerGet(gomock.Any(), testSQLMockCustomerID).Return(cached, nil)

		res, err := h.CustomerGet(context.Background(), testSQLMockCustomerID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res != cached {
			t.Errorf("Wrong match. expect: the cached customer, got: %v", res)
		}
		if mock.ExpectationsWereMet() == nil {
			t.Errorf("Wrong match. expect: the database is not queried after a cache hit")
		}
	})

	t.Run("accesskey", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockCache := cachehandler.NewMockCacheHandler(mc)
		h, mock := newSQLMockHandler(t, mockCache)
		mock.ExpectQuery("SELECT .* FROM customer_accesskeys").WillReturnRows(mismatchedRows())

		cached := &accesskey.Accesskey{Name: "cached accesskey"}
		mockCache.EXPECT().AccesskeyGet(gomock.Any(), testSQLMockAccesskeyID).Return(cached, nil)

		res, err := h.AccesskeyGet(context.Background(), testSQLMockAccesskeyID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res != cached {
			t.Errorf("Wrong match. expect: the cached accesskey, got: %v", res)
		}
		if mock.ExpectationsWereMet() == nil {
			t.Errorf("Wrong match. expect: the database is not queried after a cache hit")
		}
	})
}

func Test_sqlmock_updateToCache(t *testing.T) {
	errCache := errors.New("cache write failure")

	tests := []struct {
		name         string
		useAccesskey bool
		cacheErr     error
	}{
		{"customer, cache write fails", false, errCache},
		{"customer, cache write succeeds", false, nil},
		{"accesskey, cache write fails", true, errCache},
		{"accesskey, cache write succeeds", true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h, mock := newSQLMockHandler(t, mockCache)

			// the row read from the database is what must be written to the cache
			var gotID uuid.UUID
			var gotName string
			var expectID uuid.UUID
			var err error
			if tt.useAccesskey {
				expectID = testSQLMockAccesskeyID
				mock.ExpectQuery("SELECT .* FROM customer_accesskeys").WillReturnRows(storedRows(&accesskey.Accesskey{}, expectID, "stored accesskey"))
				mockCache.EXPECT().AccesskeySet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, a *accesskey.Accesskey) error {
					gotID, gotName = a.ID, a.Name
					return tt.cacheErr
				})
				err = h.accesskeyUpdateToCache(context.Background(), expectID)
			} else {
				expectID = testSQLMockCustomerID
				mock.ExpectQuery("SELECT .* FROM customer_customers").WillReturnRows(storedRows(&customer.Customer{}, expectID, "stored customer"))
				mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, c *customer.Customer) error {
					gotID, gotName = c.ID, c.Name
					return tt.cacheErr
				})
				err = h.customerUpdateToCache(context.Background(), expectID)
			}

			if !errors.Is(err, tt.cacheErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.cacheErr, err)
			}
			if gotID != expectID {
				t.Errorf("Wrong match. expect the cached id: %s, got: %s", expectID, gotID)
			}

			expectName := "stored customer"
			if tt.useAccesskey {
				expectName = "stored accesskey"
			}
			if gotName != expectName {
				t.Errorf("Wrong match. expect the cached name: %s, got: %s", expectName, gotName)
			}
		})
	}
}

// Test_sqlmock_emptyUpdate checks that an update without fields does nothing:
// the mock database has no scripted call and the cache has no expectation.
func Test_sqlmock_emptyUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mock := newSQLMockHandler(t, cachehandler.NewMockCacheHandler(mc))

	if err := h.CustomerUpdate(context.Background(), testSQLMockCustomerID, map[customer.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := h.AccesskeyUpdate(context.Background(), testSQLMockAccesskeyID, map[accesskey.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Wrong match. expect: no database call, got: %v", err)
	}
}

// Test_sqlmock_updateToCache_queryError checks that a failed read of the row is
// returned and nothing is written to the cache.
func Test_sqlmock_updateToCache_queryError(t *testing.T) {
	tests := []struct {
		name  string
		table string
		call  func(h *handler) error
	}{
		{
			name:  "customer",
			table: "customer_customers",
			call: func(h *handler) error {
				return h.customerUpdateToCache(context.Background(), testSQLMockCustomerID)
			},
		},
		{
			name:  "accesskey",
			table: "customer_accesskeys",
			call: func(h *handler) error {
				return h.accesskeyUpdateToCache(context.Background(), testSQLMockAccesskeyID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no cache expectation: the cache must not be written
			h, mock := newSQLMockHandler(t, cachehandler.NewMockCacheHandler(mc))
			mock.ExpectQuery("SELECT .* FROM " + tt.table).WillReturnError(errTestSQLMock)

			if err := tt.call(h); !errors.Is(err, errTestSQLMock) {
				t.Errorf("Wrong match. expect: %v, got: %v", errTestSQLMock, err)
			}
		})
	}
}

// Test_sqlmock_listUnsupportedFilter checks that a filter value of an unsupported
// type is rejected before the database is queried.
func Test_sqlmock_listUnsupportedFilter(t *testing.T) {
	tests := []struct {
		name   string
		expect string
		call   func(h *handler) error
	}{
		{
			name:   "customer",
			expect: "could not apply filters. CustomerGets",
			call: func(h *handler) error {
				_, err := h.CustomerList(context.Background(), 10, "2024-01-01T00:00:00.000000Z", map[customer.Field]any{customer.FieldName: struct{}{}})
				return err
			},
		},
		{
			name:   "accesskey",
			expect: "could not apply filters. AccesskeyGets",
			call: func(h *handler) error {
				_, err := h.AccesskeyList(context.Background(), 10, "2024-01-01T00:00:00.000000Z", map[accesskey.Field]any{accesskey.FieldName: struct{}{}})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the mock database has no scripted query: a query would fail the call differently
			h, _ := newSQLMockHandler(t, cachehandler.NewMockCacheHandler(mc))

			err := tt.call(h)
			if err == nil || !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("Wrong match. expect an error containing %q, got: %v", tt.expect, err)
			}
			if err != nil && !strings.Contains(err.Error(), "unsupported filter type") {
				t.Errorf("Wrong match. expect the unsupported type to be named, got: %v", err)
			}
		})
	}
}
