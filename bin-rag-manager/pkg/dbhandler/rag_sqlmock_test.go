package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"monorepo/bin-rag-manager/models/rag"
)

func ragRows() *sqlmock.Rows {
	return sqlmock.NewRows(strings.Split(ragCols, ", "))
}

func Test_handler_RagCreate(t *testing.T) {
	const insertSQL = "INSERT INTO rag_rags (id,customer_id,name,description,tm_create,tm_update) VALUES ($1,$2,$3,$4,$5,$6)"

	tests := []struct {
		name    string
		execErr error
		wantErr bool
	}{
		{name: "success"},
		{name: "exec error is wrapped", execErr: errors.New("connection reset"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			expectNow(util, 2)

			exp := mock.ExpectExec(insertSQL).WithArgs(testID1, testID2, "name1", "desc1", testNow, testNow)
			if tt.execErr != nil {
				exp.WillReturnError(tt.execErr)
			} else {
				exp.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			r := &rag.Rag{ID: testID1, CustomerID: testID2, Name: "name1", Description: "desc1"}
			err := h.RagCreate(context.Background(), r)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Wrong match. wantErr: %v, got: %v", tt.wantErr, err)
			}
			if tt.execErr != nil && !errors.Is(err, tt.execErr) {
				t.Errorf("Wrong match. expect the driver error to be wrapped, got: %v", err)
			}
			if tt.execErr == nil && (r.TMCreate == nil || !r.TMCreate.Equal(testNow) || r.TMUpdate == nil || !r.TMUpdate.Equal(testNow)) {
				t.Errorf("Wrong match. expect timestamps to be populated on the caller struct, got: %v %v", r.TMCreate, r.TMUpdate)
			}
			assertMet(t, mock)
		})
	}
}

func Test_handler_RagGet(t *testing.T) {
	const getSQL = "SELECT " + ragCols + " FROM rag_rags WHERE id = $1 AND tm_delete IS NULL"

	create := testNow
	update := testNow.Add(time.Minute)

	t.Run("success", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(getSQL).WithArgs(testID1).
			WillReturnRows(ragRows().AddRow(testID1.String(), testID2.String(), "name1", "desc1", create, update, nil))

		got, err := h.RagGet(context.Background(), testID1)
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		want := &rag.Rag{ID: testID1, CustomerID: testID2, Name: "name1", Description: "desc1", TMCreate: &create, TMUpdate: &update}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", want, got)
		}
		assertMet(t, mock)
	})

	t.Run("no rows becomes ErrNotFound", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(getSQL).WithArgs(testID1).WillReturnError(sql.ErrNoRows)

		got, err := h.RagGet(context.Background(), testID1)
		if !errors.Is(err, ErrNotFound) || got != nil {
			t.Errorf("Wrong match. expect ErrNotFound and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("other scan error is not ErrNotFound", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("connection reset")
		mock.ExpectQuery(getSQL).WithArgs(testID1).WillReturnError(dbErr)

		_, err := h.RagGet(context.Background(), testID1)
		if err == nil || errors.Is(err, ErrNotFound) || !errors.Is(err, dbErr) {
			t.Errorf("Wrong match. expect wrapped driver error without ErrNotFound, got: %v", err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_RagList(t *testing.T) {
	const base = "SELECT " + ragCols + " FROM rag_rags WHERE tm_create < $1"
	const tail = " ORDER BY tm_create DESC LIMIT "

	tests := []struct {
		name     string
		size     uint64
		token    string
		filters  map[rag.Field]any
		wantSQL  string
		wantArgs []driver.Value
		curTime  bool
	}{
		{
			name:     "default token and size, no deleted filter hides deleted rows",
			wantSQL:  base + " AND tm_delete IS NULL" + tail + "100",
			wantArgs: []driver.Value{testCurTime},
			curTime:  true,
		},
		{
			name:     "explicit token and size",
			size:     7,
			token:    "2025-02-02 00:00:00.000000",
			wantSQL:  base + " AND tm_delete IS NULL" + tail + "7",
			wantArgs: []driver.Value{"2025-02-02 00:00:00.000000"},
		},
		{
			name:     "deleted false",
			token:    "t",
			filters:  map[rag.Field]any{"deleted": false},
			wantSQL:  base + " AND tm_delete IS NULL" + tail + "100",
			wantArgs: []driver.Value{"t"},
		},
		{
			name:     "deleted true",
			token:    "t",
			filters:  map[rag.Field]any{"deleted": true},
			wantSQL:  base + " AND tm_delete IS NOT NULL" + tail + "100",
			wantArgs: []driver.Value{"t"},
		},
		{
			name:     "non bool deleted adds no tm_delete condition",
			token:    "t",
			filters:  map[rag.Field]any{"deleted": "yes"},
			wantSQL:  base + tail + "100",
			wantArgs: []driver.Value{"t"},
		},
		{
			name:     "column filter",
			token:    "t",
			filters:  map[rag.Field]any{rag.FieldCustomerID: testID2},
			wantSQL:  base + " AND customer_id = $2 AND tm_delete IS NULL" + tail + "100",
			wantArgs: []driver.Value{"t", testID2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			if tt.curTime {
				util.EXPECT().TimeGetCurTime().Return(testCurTime)
			}

			create := testNow
			mock.ExpectQuery(tt.wantSQL).WithArgs(tt.wantArgs...).
				WillReturnRows(ragRows().
					AddRow(testID1.String(), testID2.String(), "n1", "d1", create, create, nil).
					AddRow(testID3.String(), testID2.String(), "n2", "d2", create, create, nil))

			got, err := h.RagList(context.Background(), tt.size, tt.token, tt.filters)
			if err != nil {
				t.Fatalf("Wrong match. expect ok, got: %v", err)
			}
			if len(got) != 2 || got[0].ID != testID1 || got[1].ID != testID3 || got[1].Name != "n2" {
				t.Errorf("Wrong match. unexpected rows: %+v", got)
			}
			assertMet(t, mock)
		})
	}

	t.Run("query error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		dbErr := errors.New("boom")
		mock.ExpectQuery(base + " AND tm_delete IS NULL" + tail + "100").WithArgs("t").WillReturnError(dbErr)

		got, err := h.RagList(context.Background(), 0, "t", nil)
		if !errors.Is(err, dbErr) || got != nil {
			t.Errorf("Wrong match. expect wrapped query error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("scan error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		mock.ExpectQuery(base + " AND tm_delete IS NULL" + tail + "100").WithArgs("t").
			WillReturnRows(ragRows().AddRow("not-a-uuid", testID2.String(), "n", "d", testNow, testNow, nil))

		got, err := h.RagList(context.Background(), 0, "t", nil)
		if err == nil || got != nil {
			t.Errorf("Wrong match. expect scan error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})

	t.Run("row iteration error", func(t *testing.T) {
		h, mock, _ := newTestHandler(t)
		rowErr := errors.New("stream broken")
		mock.ExpectQuery(base + " AND tm_delete IS NULL" + tail + "100").WithArgs("t").
			WillReturnRows(ragRows().AddRow(testID1.String(), testID2.String(), "n", "d", testNow, testNow, nil).RowError(0, rowErr))

		got, err := h.RagList(context.Background(), 0, "t", nil)
		if !errors.Is(err, rowErr) || got != nil {
			t.Errorf("Wrong match. expect row error and nil, got: %v, %v", got, err)
		}
		assertMet(t, mock)
	})
}

func Test_handler_RagUpdate(t *testing.T) {
	const updateSQL = "UPDATE rag_rags SET description = $1, name = $2, tm_update = $3 WHERE id = $4 AND tm_delete IS NULL"

	tests := []struct {
		name    string
		execErr error
	}{
		{name: "success"},
		{name: "exec error", execErr: errors.New("deadlock")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			expectNow(util, 1)

			exp := mock.ExpectExec(updateSQL).WithArgs("d2", "n2", testNow, testID1)
			if tt.execErr != nil {
				exp.WillReturnError(tt.execErr)
			} else {
				exp.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := h.RagUpdate(context.Background(), testID1, map[rag.Field]any{rag.FieldName: "n2", rag.FieldDescription: "d2"})
			if tt.execErr == nil && err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
			if tt.execErr != nil && !errors.Is(err, tt.execErr) {
				t.Errorf("Wrong match. expect wrapped exec error, got: %v", err)
			}
			assertMet(t, mock)
		})
	}
}

func Test_handler_RagDelete(t *testing.T) {
	const deleteSQL = "UPDATE rag_rags SET tm_delete = $1, tm_update = $2 WHERE id = $3 AND tm_delete IS NULL"

	tests := []struct {
		name    string
		execErr error
	}{
		{name: "success"},
		{name: "exec error", execErr: errors.New("deadlock")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, util := newTestHandler(t)
			expectNow(util, 1)

			exp := mock.ExpectExec(deleteSQL).WithArgs(testNow, testNow, testID1)
			if tt.execErr != nil {
				exp.WillReturnError(tt.execErr)
			} else {
				exp.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := h.RagDelete(context.Background(), testID1)
			if tt.execErr == nil && err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
			if tt.execErr != nil && !errors.Is(err, tt.execErr) {
				t.Errorf("Wrong match. expect wrapped exec error, got: %v", err)
			}
			assertMet(t, mock)
		})
	}
}
