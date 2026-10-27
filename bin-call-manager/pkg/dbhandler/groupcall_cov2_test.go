package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-call-manager/models/groupcall"
)

func Test_Groupcall_cov2_singleGetFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2d00001-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&groupcall.Groupcall{})

	t.Run("groupcallGetFromDB", func(t *testing.T) {
		runCov2SingleGet(t, groupcallTable, cols, nil, func(h *handler) error {
			_, err := h.groupcallGetFromDB(ctx, id)
			return err
		})
	})

	t.Run("GroupcallGet_db_error_after_cache_miss", func(t *testing.T) {
		prep := func(e cov2Env) {
			e.cache.EXPECT().GroupcallGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
		}
		runCov2SingleGet(t, groupcallTable, cols, prep, func(h *handler) error {
			res, err := h.GroupcallGet(ctx, id)
			if err != nil && res != nil {
				t.Errorf("expected no groupcall together with an error, got: %v", res)
			}
			return err
		})
	})
}

func Test_Groupcall_cov2_GroupcallGet_cacheHit(t *testing.T) {
	id := uuid.FromStringOrNil("c2d00002-1b2d-11f1-8a5e-0b3f6c1d2e01")
	want := &groupcall.Groupcall{}
	want.ID = id

	e := newCov2Env(t)
	e.cache.EXPECT().GroupcallGet(gomock.Any(), id).Return(want, nil).Times(1)

	got, err := e.h.GroupcallGet(context.Background(), id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", want, got)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the DB must not be touched on a cache hit. err: %v", err)
	}
}

func Test_Groupcall_cov2_GroupcallGet_dbHitNormalizesNull(t *testing.T) {
	id := uuid.FromStringOrNil("c2d00003-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&groupcall.Groupcall{})

	e := newCov2Env(t)
	e.cache.EXPECT().GroupcallGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
	e.mock.ExpectQuery(cov2SelectRe(groupcallTable)).WillReturnRows(cov2NullRows(cols))
	e.cache.EXPECT().GroupcallSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	got, err := e.h.GroupcallGet(context.Background(), id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got.Source, &commonaddress.Address{}) {
		t.Errorf("Wrong Source. got: %#v", got.Source)
	}
	if !reflect.DeepEqual(got.Destinations, []commonaddress.Address{}) {
		t.Errorf("Wrong Destinations. got: %#v", got.Destinations)
	}
	if !reflect.DeepEqual(got.CallIDs, []uuid.UUID{}) {
		t.Errorf("Wrong CallIDs. got: %#v", got.CallIDs)
	}
	if !reflect.DeepEqual(got.GroupcallIDs, []uuid.UUID{}) {
		t.Errorf("Wrong GroupcallIDs. got: %#v", got.GroupcallIDs)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_Groupcall_cov2_GroupcallList_failures(t *testing.T) {
	ctx := context.Background()
	cols := commondatabasehandler.GetDBFields(&groupcall.Groupcall{})

	runCov2List(t, groupcallTable, cols, true, func(h *handler) error {
		res, err := h.GroupcallList(ctx, 10, "", map[groupcall.Field]any{})
		if err != nil && res != nil {
			t.Errorf("expected no result together with an error, got: %v", res)
		}
		return err
	})
}

func Test_Groupcall_cov2_updateToCache(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2d00004-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&groupcall.Groupcall{})

	expectSetErr := func(e cov2Env) {
		e.cache.EXPECT().GroupcallSet(gomock.Any(), gomock.Any()).Return(errCov2Cache).Times(1)
	}

	runCov2UpdateToCache(t, groupcallTable, cols, expectSetErr, func(h *handler) error {
		return h.groupcallUpdateToCache(ctx, id)
	})
}

func Test_Groupcall_cov2_writeFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2d00005-1b2d-11f1-8a5e-0b3f6c1d2e01")
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name   string
		stmtRe string
		run    func(h *handler) error
	}{
		{
			name:   "GroupcallCreate",
			stmtRe: cov2InsertRe(groupcallTable),
			run: func(h *handler) error {
				g := &groupcall.Groupcall{}
				g.ID = id
				return h.GroupcallCreate(ctx, g)
			},
		},
		{
			name:   "GroupcallUpdate",
			stmtRe: cov2UpdateRe(groupcallTable),
			run: func(h *handler) error {
				return h.GroupcallUpdate(ctx, id, map[groupcall.Field]any{groupcall.FieldTMUpdate: &ts})
			},
		},
		{
			name:   "groupcallDecreaseCount",
			stmtRe: cov2UpdateRe(groupcallTable),
			run: func(h *handler) error {
				return h.groupcallDecreaseCount(ctx, "call_count", id, "GroupcallDecreaseCallCount")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCov2ExecFail(t, tt.stmtRe, tt.run)
		})
	}
}

func Test_Groupcall_cov2_GroupcallUpdate_emptyFields(t *testing.T) {
	e := newCov2Env(t)

	if err := e.h.GroupcallUpdate(context.Background(), uuid.Nil, map[groupcall.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("nothing may be executed for empty fields. err: %v", err)
	}
}
