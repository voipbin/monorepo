package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/confbridge"
)

func Test_Confbridge_cov2_singleGetFailures(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	id := uuid.FromStringOrNil("c2b00001-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&confbridge.Confbridge{})

	t.Run("confbridgeGetFromDB", func(t *testing.T) {
		runCov2SingleGet(t, confbridgeTable, cols, nil, func(h *handler) error {
			_, err := h.confbridgeGetFromDB(ctx, id)
			return err
		})
	})

	t.Run("ConfbridgeGet_db_error_after_cache_miss", func(t *testing.T) {
		prep := func(e cov2Env) {
			e.cache.EXPECT().ConfbridgeGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
		}
		runCov2SingleGet(t, confbridgeTable, cols, prep, func(h *handler) error {
			res, err := h.ConfbridgeGet(ctx, id)
			if err != nil && res != nil {
				t.Errorf("expected no confbridge together with an error, got: %v", res)
			}
			return err
		})
	})

	t.Run("ConfbridgeGetByBridgeID", func(t *testing.T) {
		runCov2SingleGet(t, confbridgeTable, cols, nil, func(h *handler) error {
			_, err := h.ConfbridgeGetByBridgeID(ctx, "bridge-cov2")
			return err
		})
	})
}

func Test_Confbridge_cov2_ConfbridgeGet_cacheHit(t *testing.T) {
	resetTestDB(t)

	id := uuid.FromStringOrNil("c2b00002-1b2d-11f1-8a5e-0b3f6c1d2e01")
	want := &confbridge.Confbridge{}
	want.ID = id

	e := newCov2Env(t)
	e.cache.EXPECT().ConfbridgeGet(gomock.Any(), id).Return(want, nil).Times(1)

	got, err := e.h.ConfbridgeGet(context.Background(), id)
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

func Test_Confbridge_cov2_ConfbridgeGet_dbHitNormalizesNull(t *testing.T) {
	resetTestDB(t)

	id := uuid.FromStringOrNil("c2b00003-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&confbridge.Confbridge{})

	e := newCov2Env(t)
	e.cache.EXPECT().ConfbridgeGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
	e.mock.ExpectQuery(cov2SelectRe(confbridgeTable)).WillReturnRows(cov2NullRows(cols))
	e.cache.EXPECT().ConfbridgeSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	got, err := e.h.ConfbridgeGet(context.Background(), id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(got.ChannelCallIDs, map[string]uuid.UUID{}) {
		t.Errorf("Wrong ChannelCallIDs. got: %#v", got.ChannelCallIDs)
	}
	if !reflect.DeepEqual(got.RecordingIDs, []uuid.UUID{}) {
		t.Errorf("Wrong RecordingIDs. got: %#v", got.RecordingIDs)
	}
	if !reflect.DeepEqual(got.ExternalMediaIDs, []uuid.UUID{}) {
		t.Errorf("Wrong ExternalMediaIDs. got: %#v", got.ExternalMediaIDs)
	}
	if !reflect.DeepEqual(got.Flags, []confbridge.Flag{}) {
		t.Errorf("Wrong Flags. got: %#v", got.Flags)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_Confbridge_cov2_ConfbridgeList_failures(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	cols := commondatabasehandler.GetDBFields(&confbridge.Confbridge{})

	runCov2List(t, confbridgeTable, cols, true, func(h *handler) error {
		res, err := h.ConfbridgeList(ctx, 10, "", map[confbridge.Field]any{})
		if err != nil && res != nil {
			t.Errorf("expected no result together with an error, got: %v", res)
		}
		return err
	})
}

func Test_Confbridge_cov2_updateToCache(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	id := uuid.FromStringOrNil("c2b00004-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&confbridge.Confbridge{})

	expectSetErr := func(e cov2Env) {
		e.cache.EXPECT().ConfbridgeSet(gomock.Any(), gomock.Any()).Return(errCov2Cache).Times(1)
	}

	runCov2UpdateToCache(t, confbridgeTable, cols, expectSetErr, func(h *handler) error {
		return h.confbridgeUpdateToCache(ctx, id)
	})
}

func Test_Confbridge_cov2_writeFailures(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	id := uuid.FromStringOrNil("c2b00005-1b2d-11f1-8a5e-0b3f6c1d2e01")
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name   string
		stmtRe string
		run    func(h *handler) error
	}{
		{
			name:   "ConfbridgeCreate",
			stmtRe: cov2InsertRe(confbridgeTable),
			run: func(h *handler) error {
				cb := &confbridge.Confbridge{}
				cb.ID = id
				return h.ConfbridgeCreate(ctx, cb)
			},
		},
		{
			name:   "ConfbridgeUpdate",
			stmtRe: cov2UpdateRe(confbridgeTable),
			run: func(h *handler) error {
				return h.ConfbridgeUpdate(ctx, id, map[confbridge.Field]any{confbridge.FieldBridgeID: "bridge-cov2", confbridge.FieldTMUpdate: &ts})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCov2ExecFail(t, tt.stmtRe, tt.run)
		})
	}
}

func Test_Confbridge_cov2_ConfbridgeUpdate_emptyFields(t *testing.T) {
	resetTestDB(t)

	e := newCov2Env(t)

	if err := e.h.ConfbridgeUpdate(context.Background(), uuid.Nil, map[confbridge.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("nothing may be executed for empty fields. err: %v", err)
	}
}
