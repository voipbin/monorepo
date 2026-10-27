package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/channel"
)

func Test_Channel_cov2_singleGetFailures(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	cols := commondatabasehandler.GetDBFields(&channel.Channel{})
	id := "channel-cov2-1"

	t.Run("channelGetFromDB", func(t *testing.T) {
		runCov2SingleGet(t, channelTable, cols, nil, func(h *handler) error {
			_, err := h.channelGetFromDB(ctx, id)
			return err
		})
	})

	t.Run("ChannelGet_db_error_after_cache_miss", func(t *testing.T) {
		prep := func(e cov2Env) {
			e.cache.EXPECT().ChannelGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
		}
		runCov2SingleGet(t, channelTable, cols, prep, func(h *handler) error {
			res, err := h.ChannelGet(ctx, id)
			if err != nil && res != nil {
				t.Errorf("expected no channel together with an error, got: %v", res)
			}
			return err
		})
	})
}

func Test_Channel_cov2_ChannelGet_cacheHit(t *testing.T) {
	resetTestDB(t)

	id := "channel-cov2-2"
	want := &channel.Channel{}
	want.ID = id

	e := newCov2Env(t)
	e.cache.EXPECT().ChannelGet(gomock.Any(), id).Return(want, nil).Times(1)

	got, err := e.h.ChannelGet(context.Background(), id)
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

func Test_Channel_cov2_listFailures(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	cols := commondatabasehandler.GetDBFields(&channel.Channel{})
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("ChannelList", func(t *testing.T) {
		runCov2List(t, channelTable, cols, true, func(h *handler) error {
			res, err := h.ChannelList(ctx, 10, "", map[channel.Field]any{})
			if err != nil && res != nil {
				t.Errorf("expected no result together with an error, got: %v", res)
			}
			return err
		})
	})

	t.Run("ChannelGetsForRecovery", func(t *testing.T) {
		runCov2List(t, channelTable, cols, false, func(h *handler) error {
			res, err := h.ChannelGetsForRecovery(ctx, "asterisk-cov2", channel.TypeCall, &start, &end, 10)
			if err != nil && res != nil {
				t.Errorf("expected no result together with an error, got: %v", res)
			}
			return err
		})
	})
}

func Test_Channel_cov2_updateToCache(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	cols := commondatabasehandler.GetDBFields(&channel.Channel{})

	expectSetErr := func(e cov2Env) {
		e.cache.EXPECT().ChannelSet(gomock.Any(), gomock.Any()).Return(errCov2Cache).Times(1)
	}

	runCov2UpdateToCache(t, channelTable, cols, expectSetErr, func(h *handler) error {
		return h.channelUpdateToCache(ctx, "channel-cov2-3")
	})
}

func Test_Channel_cov2_writeFailures(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name   string
		stmtRe string
		run    func(h *handler) error
	}{
		{
			name:   "ChannelCreate",
			stmtRe: cov2InsertRe(channelTable),
			run: func(h *handler) error {
				c := &channel.Channel{}
				c.ID = "channel-cov2-4"
				return h.ChannelCreate(ctx, c)
			},
		},
		{
			name:   "channelUpdate",
			stmtRe: cov2UpdateRe(channelTable),
			run: func(h *handler) error {
				return h.channelUpdate(ctx, "channel-cov2-5", map[channel.Field]any{channel.FieldBridgeID: "bridge-cov2", channel.FieldTMUpdate: &ts})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCov2ExecFail(t, tt.stmtRe, tt.run)
		})
	}
}

func Test_Channel_cov2_channelUpdate_emptyFields(t *testing.T) {
	resetTestDB(t)

	e := newCov2Env(t)

	if err := e.h.channelUpdate(context.Background(), "channel-cov2-6", map[channel.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("nothing may be executed for empty fields. err: %v", err)
	}
}
