package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdialtarget"
)

// availableColumns are the columns of outdialTargetSelectAvailable in order.
var availableColumns = []string{
	"id", "outdial_id", "name", "detail", "data", "status",
	"destination_0", "destination_1", "destination_2", "destination_3", "destination_4",
	"try_count_0", "try_count_1", "try_count_2", "try_count_3", "try_count_4",
	"tm_create", "tm_update", "tm_delete",
	"des_0", "des_1", "des_2", "des_3", "des_4",
}

func Test_OutdialTargetGetAvailable_args(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mock := newMockHandler(t, mc)
	ctx := context.Background()

	outdialID := uuid.FromStringOrNil("5c1e7a30-b9c7-11ef-8a01-0a1b2c3d4e5f")

	// distinct values, so that an argument reorder fails the expectation
	mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
		ExpectQuery().
		WithArgs(1, 2, 3, 4, 5, outdialID.Bytes(), uint64(7)).
		WillReturnRows(sqlmock.NewRows(availableColumns))

	res, err := h.OutdialTargetGetAvailable(ctx, outdialID, 1, 2, 3, 4, 5, 7)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res == nil || len(res) != 0 {
		t.Errorf("expected an empty non-nil list, got: %+v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_OutdialTargetGetAvailable_mapping(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mock := newMockHandler(t, mc)
	ctx := context.Background()

	outdialID := uuid.FromStringOrNil("5c1e7a30-b9c7-11ef-8a01-0a1b2c3d4e5f")
	id1 := uuid.FromStringOrNil("6d2f8b41-b9c7-11ef-9b12-1b2c3d4e5f60")
	id2 := uuid.FromStringOrNil("7e309c52-b9c7-11ef-8c23-2c3d4e5f6071")
	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tmUpdate := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	tmDelete := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows(availableColumns).
		// every column populated with a distinct value
		AddRow(
			id1.Bytes(), outdialID.Bytes(), "name 1", "detail 1", "data 1", "idle",
			`{"type":"tel","target":"+15550100000"}`,
			`{"type":"tel","target":"+15550100001"}`,
			`{"type":"tel","target":"+15550100002"}`,
			`{"type":"tel","target":"+15550100003"}`,
			`{"type":"tel","target":"+15550100004"}`,
			int64(10), int64(11), int64(12), int64(13), int64(14),
			tmCreate, tmUpdate, tmDelete,
			int64(1), int64(0), int64(1), int64(0), int64(1),
		).
		// nullable columns left NULL
		AddRow(
			id2.Bytes(), outdialID.Bytes(), nil, nil, nil, nil,
			nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil,
			nil, nil, nil,
			nil, nil, nil, nil, nil,
		)

	mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
		ExpectQuery().
		WithArgs(1, 2, 3, 4, 5, outdialID.Bytes(), uint64(10)).
		WillReturnRows(rows)

	res, err := h.OutdialTargetGetAvailable(ctx, outdialID, 1, 2, 3, 4, 5, 10)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 targets, got: %d", len(res))
	}

	got := res[0]
	if got.ID != id1 || got.OutdialID != outdialID {
		t.Errorf("wrong ids. id: %s, outdial_id: %s", got.ID, got.OutdialID)
	}
	if got.Name != "name 1" || got.Detail != "detail 1" || got.Data != "data 1" || got.Status != outdialtarget.StatusIdle {
		t.Errorf("wrong basic fields. got: %+v", got)
	}
	if got.TryCount0 != 10 || got.TryCount1 != 11 || got.TryCount2 != 12 || got.TryCount3 != 13 || got.TryCount4 != 14 {
		t.Errorf("wrong try counts. got: %d %d %d %d %d", got.TryCount0, got.TryCount1, got.TryCount2, got.TryCount3, got.TryCount4)
	}
	dests := []string{
		got.Destination0.Target, got.Destination1.Target, got.Destination2.Target,
		got.Destination3.Target, got.Destination4.Target,
	}
	for i, d := range dests {
		if expect := "+1555010000" + string(rune('0'+i)); d != expect {
			t.Errorf("wrong destination %d. expect: %s, got: %s", i, expect, d)
		}
	}
	if got.TMCreate == nil || !got.TMCreate.Equal(tmCreate) {
		t.Errorf("wrong tm_create. got: %v", got.TMCreate)
	}
	if got.TMUpdate == nil || !got.TMUpdate.Equal(tmUpdate) {
		t.Errorf("wrong tm_update. got: %v", got.TMUpdate)
	}
	if got.TMDelete == nil || !got.TMDelete.Equal(tmDelete) {
		t.Errorf("wrong tm_delete. got: %v", got.TMDelete)
	}

	empty := res[1]
	if empty.ID != id2 {
		t.Errorf("wrong id of the second target. got: %s", empty.ID)
	}
	if empty.Name != "" || empty.Status != "" || empty.TryCount0 != 0 || empty.TryCount4 != 0 {
		t.Errorf("expected zero values for NULL columns. got: %+v", empty)
	}
	if empty.Destination0 != nil || empty.Destination4 != nil {
		t.Errorf("expected nil destinations for NULL columns")
	}
	if empty.TMCreate != nil || empty.TMUpdate != nil || empty.TMDelete != nil {
		t.Errorf("expected nil times for NULL columns")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_OutdialTargetGetAvailable_error(t *testing.T) {
	outdialID := uuid.FromStringOrNil("5c1e7a30-b9c7-11ef-8a01-0a1b2c3d4e5f")
	errDB := errors.New("simulated db failure")
	good := `{"type":"tel","target":"+15550100000"}`

	// rowWithIDs returns a full row with explicit id and outdial id column
	// values, where the destination at position i is replaced.
	rowWithIDs := func(i int, dest string, id, outdialIDValue driver.Value) []driver.Value {
		dests := []driver.Value{good, good, good, good, good}
		dests[i] = dest
		row := []driver.Value{id, outdialIDValue, "n", "d", "x", "idle"}
		row = append(row, dests...)
		row = append(row, int64(0), int64(0), int64(0), int64(0), int64(0))
		row = append(row, nil, nil, nil)
		row = append(row, int64(1), int64(1), int64(1), int64(1), int64(1))
		return row
	}

	// rowWith returns a full row where the destination at position i is replaced.
	rowWith := func(i int, dest string) []driver.Value {
		return rowWithIDs(i, dest, outdialID.Bytes(), outdialID.Bytes())
	}

	tests := []struct {
		name string

		setup func(mock sqlmock.Sqlmock)
		// expect is a fragment the error message must contain.
		expect string
	}{
		{
			"prepare fails",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).WillReturnError(errDB)
			},
			"simulated db failure",
		},
		{
			"query fails",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().WillReturnError(errDB)
			},
			"simulated db failure",
		},
		{
			"scan fails on a wrong column count",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(outdialID.Bytes()))
			},
			"could not scan the row. outdialTargetGetFromRowForAvailable",
		},
		{
			"invalid json in destination 0",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().
					WillReturnRows(sqlmock.NewRows(availableColumns).AddRow(rowWith(0, "{broken")...))
			},
			"could not parse destination0",
		},
		{
			"invalid json in destination 1",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().
					WillReturnRows(sqlmock.NewRows(availableColumns).AddRow(rowWith(1, "{broken")...))
			},
			"could not parse destination1",
		},
		{
			"invalid json in destination 2",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().
					WillReturnRows(sqlmock.NewRows(availableColumns).AddRow(rowWith(2, "{broken")...))
			},
			"could not parse destination2",
		},
		{
			"invalid json in destination 3",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().
					WillReturnRows(sqlmock.NewRows(availableColumns).AddRow(rowWith(3, "{broken")...))
			},
			"could not parse destination3",
		},
		{
			"invalid json in destination 4",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().
					WillReturnRows(sqlmock.NewRows(availableColumns).AddRow(rowWith(4, "{broken")...))
			},
			"could not parse destination4",
		},
		{
			"invalid id length",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().
					WillReturnRows(sqlmock.NewRows(availableColumns).AddRow(rowWithIDs(0, good, []byte("abc"), outdialID.Bytes())...))
			},
			"could not parse id",
		},
		{
			"empty id",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().
					WillReturnRows(sqlmock.NewRows(availableColumns).AddRow(rowWithIDs(0, good, []byte{}, outdialID.Bytes())...))
			},
			"could not parse id",
		},
		{
			"invalid outdial id length",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(regexp.QuoteMeta(outdialTargetSelectAvailable)).
					ExpectQuery().
					WillReturnRows(sqlmock.NewRows(availableColumns).AddRow(rowWithIDs(0, good, outdialID.Bytes(), []byte("abc"))...))
			},
			"could not parse outdial id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mock := newMockHandler(t, mc)
			tt.setup(mock)

			res, err := h.OutdialTargetGetAvailable(context.Background(), outdialID, 1, 2, 3, 4, 5, 10)
			if err == nil {
				t.Fatalf("expected an error, got nil and result: %+v", res)
			}
			if res != nil {
				t.Errorf("expected a nil result on error, got: %+v", res)
			}
			if !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("wrong error message. expect fragment: %q, got: %v", tt.expect, err)
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("a failure must not be reported as ErrNotFound. err: %v", err)
			}
		})
	}
}
