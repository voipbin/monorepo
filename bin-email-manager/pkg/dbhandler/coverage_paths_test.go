package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/cachehandler"
)

// brokenValuer is a filter value whose conversion to a driver value fails.
type brokenValuer struct{}

func (brokenValuer) Value() (driver.Value, error) {
	return nil, errors.New("simulated valuer failure")
}

// brokenSqlizer is an update value that fails while the query is built.
type brokenSqlizer func() (string, []any, error)

func (f brokenSqlizer) ToSql() (string, []any, error) {
	return f()
}

// badTypeRow returns a full row whose id column holds a value that cannot be scanned into a uuid.
func badTypeRow(t *testing.T) *sqlmock.Rows {
	t.Helper()

	columns, values := fullRow(t, uuid.FromStringOrNil("5e8f0f0a-7777-4777-8777-000000000001"))
	for i, c := range columns {
		if c == "id" {
			values[i] = int64(7)
		}
	}

	return sqlmock.NewRows(columns).AddRow(values...)
}

func Test_emailGetFromDB_scanFails(t *testing.T) {
	h, mock := newMockHandler(t)
	id := uuid.FromStringOrNil("5e8f0f0a-7777-4777-8777-000000000001")

	mock.ExpectQuery("SELECT").WillReturnRows(badTypeRow(t))

	res, err := h.emailGetFromDB(context.Background(), id)
	if err == nil || !strings.Contains(err.Error(), "could not scan the row. emailGetFromRow") {
		t.Errorf("Wrong match. expect a scan error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected a nil result. got: %v", res)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("A scan failure must not be reported as not found")
	}
}

func Test_EmailList_scanFails(t *testing.T) {
	h, mock := newMockHandler(t)

	mock.ExpectQuery("SELECT").WillReturnRows(badTypeRow(t))

	res, err := h.EmailList(context.Background(), "", 10, map[email.Field]any{})
	if err == nil || !strings.Contains(err.Error(), "could not scan the row. EmailList") {
		t.Errorf("Wrong match. expect a scan error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected a nil result. got: %v", res)
	}
}

func Test_EmailList_filterErrors(t *testing.T) {
	tests := []struct {
		name string

		filters map[email.Field]any

		expectErr string
	}{
		{
			name:      "unsupported_filter_type",
			filters:   map[email.Field]any{email.FieldCustomerID: make(chan int)},
			expectErr: "could not apply filters. EmailList: unsupported filter type for customer_id",
		},
		{
			name:      "filter_value_fails_while_building_query",
			filters:   map[email.Field]any{email.FieldSubject: commondatabasehandler.NotEq{Value: brokenValuer{}}},
			expectErr: "could not build query. EmailList: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// sqlmock without expectations: no query may reach the database
			h, _ := newMockHandler(t)

			res, err := h.EmailList(context.Background(), "", 10, tt.filters)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Expected a nil result. got: %v", res)
			}
		})
	}
}

func Test_EmailUpdate_fieldErrors(t *testing.T) {
	tests := []struct {
		name string

		fields map[email.Field]any

		expectErr string
	}{
		{
			name:      "unsupported_field_value",
			fields:    map[email.Field]any{email.FieldSubject: []any{make(chan int)}},
			expectErr: "could not prepare fields. EmailUpdate: ",
		},
		{
			name: "field_value_fails_while_building_query",
			fields: map[email.Field]any{email.FieldSubject: brokenSqlizer(func() (string, []any, error) {
				return "", nil, errors.New("simulated sqlizer failure")
			})},
			expectErr: "could not build query. EmailUpdate: simulated sqlizer failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// sqlmock without expectations: no statement may reach the database
			h, _ := newMockHandler(t)

			err := h.EmailUpdate(context.Background(), uuid.FromStringOrNil("5e8f0f0a-8888-4888-8888-000000000001"), tt.fields)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_emailUpdateToCache_cacheSetFails(t *testing.T) {
	h, mock := newMockHandler(t)
	id := uuid.FromStringOrNil("5e8f0f0a-9999-4999-8999-000000000001")

	columns, values := fullRow(t, id)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(columns).AddRow(values...))

	cacheErr := errors.New("cache set failed")
	mockCache := h.cache.(*cachehandler.MockCacheHandler)
	mockCache.EXPECT().EmailSet(context.Background(), gomock.Any()).Return(cacheErr)

	if err := h.emailUpdateToCache(context.Background(), id); !errors.Is(err, cacheErr) {
		t.Errorf("Wrong match. expect: %v, got: %v", cacheErr, err)
	}
}

func Test_emailUpdateToCache_dbGetFails(t *testing.T) {
	// the cache mock has no expectations: a failed read must not write the cache
	h, mock := newMockHandler(t)

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("simulated query failure"))

	err := h.emailUpdateToCache(context.Background(), uuid.FromStringOrNil("5e8f0f0a-aaaa-4aaa-8aaa-000000000001"))
	if err == nil || !strings.Contains(err.Error(), "could not query. emailGetFromDB: simulated query failure") {
		t.Errorf("Wrong match. got: %v", err)
	}
}
