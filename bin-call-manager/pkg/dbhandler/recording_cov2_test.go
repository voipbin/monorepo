package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/recording"
)

func Test_Recording_cov2_singleGetFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2c00001-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&recording.Recording{})

	t.Run("recordingGetFromDB", func(t *testing.T) {
		runCov2SingleGet(t, recordingTable, cols, nil, func(h *handler) error {
			_, err := h.recordingGetFromDB(ctx, id)
			return err
		})
	})

	t.Run("RecordingGet_db_error_after_cache_miss", func(t *testing.T) {
		prep := func(e cov2Env) {
			e.cache.EXPECT().RecordingGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
		}
		runCov2SingleGet(t, recordingTable, cols, prep, func(h *handler) error {
			res, err := h.RecordingGet(ctx, id)
			if err != nil && res != nil {
				t.Errorf("expected no recording together with an error, got: %v", res)
			}
			return err
		})
	})

	t.Run("RecordingGetByRecordingName", func(t *testing.T) {
		runCov2SingleGet(t, recordingTable, cols, nil, func(h *handler) error {
			_, err := h.RecordingGetByRecordingName(ctx, "recording-cov2")
			return err
		})
	})
}

func Test_Recording_cov2_RecordingGet_cacheHit(t *testing.T) {
	id := uuid.FromStringOrNil("c2c00002-1b2d-11f1-8a5e-0b3f6c1d2e01")
	want := &recording.Recording{}
	want.ID = id

	e := newCov2Env(t)
	e.cache.EXPECT().RecordingGet(gomock.Any(), id).Return(want, nil).Times(1)

	got, err := e.h.RecordingGet(context.Background(), id)
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

func Test_Recording_cov2_RecordingGet_dbHitNormalizesNull(t *testing.T) {
	id := uuid.FromStringOrNil("c2c00003-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&recording.Recording{})

	e := newCov2Env(t)
	e.cache.EXPECT().RecordingGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
	e.mock.ExpectQuery(cov2SelectRe(recordingTable)).WillReturnRows(cov2NullRows(cols))
	e.cache.EXPECT().RecordingSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	got, err := e.h.RecordingGet(context.Background(), id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got.Filenames, []string{}) {
		t.Errorf("Wrong Filenames. got: %#v", got.Filenames)
	}
	if !reflect.DeepEqual(got.ChannelIDs, []string{}) {
		t.Errorf("Wrong ChannelIDs. got: %#v", got.ChannelIDs)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_Recording_cov2_RecordingList_failures(t *testing.T) {
	ctx := context.Background()
	cols := commondatabasehandler.GetDBFields(&recording.Recording{})

	runCov2List(t, recordingTable, cols, true, func(h *handler) error {
		res, err := h.RecordingList(ctx, 10, "", map[recording.Field]any{})
		if err != nil && res != nil {
			t.Errorf("expected no result together with an error, got: %v", res)
		}
		return err
	})
}

func Test_Recording_cov2_updateToCache(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2c00004-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&recording.Recording{})

	expectSetErr := func(e cov2Env) {
		e.cache.EXPECT().RecordingSet(gomock.Any(), gomock.Any()).Return(errCov2Cache).Times(1)
	}

	runCov2UpdateToCache(t, recordingTable, cols, expectSetErr, func(h *handler) error {
		return h.recordingUpdateToCache(ctx, id)
	})
}

func Test_Recording_cov2_writeFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2c00005-1b2d-11f1-8a5e-0b3f6c1d2e01")
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name   string
		stmtRe string
		run    func(h *handler) error
	}{
		{
			name:   "RecordingCreate",
			stmtRe: cov2InsertRe(recordingTable),
			run: func(h *handler) error {
				r := &recording.Recording{}
				r.ID = id
				return h.RecordingCreate(ctx, r)
			},
		},
		{
			name:   "RecordingUpdate",
			stmtRe: cov2UpdateRe(recordingTable),
			run: func(h *handler) error {
				return h.RecordingUpdate(ctx, id, map[recording.Field]any{recording.FieldTMUpdate: &ts})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCov2ExecFail(t, tt.stmtRe, tt.run)
		})
	}
}

func Test_Recording_cov2_RecordingUpdate_emptyFields(t *testing.T) {
	e := newCov2Env(t)

	if err := e.h.RecordingUpdate(context.Background(), uuid.Nil, map[recording.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("nothing may be executed for empty fields. err: %v", err)
	}
}
