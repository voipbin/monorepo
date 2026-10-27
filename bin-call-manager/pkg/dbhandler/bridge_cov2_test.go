package dbhandler

import (
	"context"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/bridge"
)

func Test_Bridge_cov2_singleGetFailures(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	cols := commondatabasehandler.GetDBFields(&bridge.Bridge{})
	id := "bridge-cov2-1"

	t.Run("bridgeGetFromDB", func(t *testing.T) {
		runCov2SingleGet(t, bridgeTable, cols, nil, func(h *handler) error {
			_, err := h.bridgeGetFromDB(ctx, id)
			return err
		})
	})

	t.Run("BridgeGet_db_error_after_cache_miss", func(t *testing.T) {
		prep := func(e cov2Env) {
			e.cache.EXPECT().BridgeGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
		}
		runCov2SingleGet(t, bridgeTable, cols, prep, func(h *handler) error {
			res, err := h.BridgeGet(ctx, id)
			if err != nil && res != nil {
				t.Errorf("expected no bridge together with an error, got: %v", res)
			}
			return err
		})
	})
}

func Test_Bridge_cov2_BridgeGet_cacheHit(t *testing.T) {
	resetTestDB(t)

	id := "bridge-cov2-2"
	want := &bridge.Bridge{}
	want.ID = id

	e := newCov2Env(t)
	e.cache.EXPECT().BridgeGet(gomock.Any(), id).Return(want, nil).Times(1)

	got, err := e.h.BridgeGet(context.Background(), id)
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

func Test_Bridge_cov2_BridgeGet_dbHitNormalizesNull(t *testing.T) {
	resetTestDB(t)

	id := "bridge-cov2-3"
	cols := commondatabasehandler.GetDBFields(&bridge.Bridge{})

	e := newCov2Env(t)
	e.cache.EXPECT().BridgeGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
	e.mock.ExpectQuery(cov2SelectRe(bridgeTable)).WillReturnRows(cov2NullRows(cols))
	e.cache.EXPECT().BridgeSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	got, err := e.h.BridgeGet(context.Background(), id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got.ChannelIDs, []string{}) {
		t.Errorf("Wrong ChannelIDs. got: %#v", got.ChannelIDs)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_Bridge_cov2_updateToCache(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	cols := commondatabasehandler.GetDBFields(&bridge.Bridge{})

	expectSetErr := func(e cov2Env) {
		e.cache.EXPECT().BridgeSet(gomock.Any(), gomock.Any()).Return(errCov2Cache).Times(1)
	}

	runCov2UpdateToCache(t, bridgeTable, cols, expectSetErr, func(h *handler) error {
		return h.bridgeUpdateToCache(ctx, "bridge-cov2-4")
	})
}

func Test_Bridge_cov2_writeFailures(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()

	tests := []struct {
		name   string
		stmtRe string
		run    func(h *handler) error
	}{
		{
			name:   "BridgeCreate",
			stmtRe: cov2InsertRe(bridgeTable),
			run: func(h *handler) error {
				b := &bridge.Bridge{}
				b.ID = "bridge-cov2-5"
				return h.BridgeCreate(ctx, b)
			},
		},
		{
			name:   "BridgeEnd",
			stmtRe: cov2UpdateRe(bridgeTable),
			run: func(h *handler) error {
				return h.BridgeEnd(ctx, "bridge-cov2-6")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCov2ExecFail(t, tt.stmtRe, tt.run)
		})
	}
}
