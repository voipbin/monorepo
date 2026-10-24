package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
)

// newClosedTestDB returns a database handle that is already closed, on a private
// in-memory database, so it does not touch the shared dbTest.
func newClosedTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:closed_%s?mode=memory&cache=private", uuid.Must(uuid.NewV4())))
	if err != nil {
		t.Fatalf("could not open the test database. err: %v", err)
	}
	if errClose := db.Close(); errClose != nil {
		t.Fatalf("could not close the test database. err: %v", errClose)
	}

	return db
}

func Test_transcribe_closedDB(t *testing.T) {
	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000001")
	curTime := time.Date(2023, 1, 3, 21, 35, 2, 0, time.UTC)

	tests := []struct {
		name string

		call func(h *handler, ctx context.Context) error
	}{
		{
			name: "create",
			call: func(h *handler, ctx context.Context) error {
				return h.TranscribeCreate(ctx, &transcribe.Transcribe{Identity: commonidentity.Identity{ID: id}})
			},
		},
		{
			name: "delete",
			call: func(h *handler, ctx context.Context) error { return h.TranscribeDelete(ctx, id) },
		},
		{
			name: "update",
			call: func(h *handler, ctx context.Context) error {
				return h.TranscribeUpdate(ctx, id, map[transcribe.Field]any{transcribe.FieldStatus: transcribe.StatusDone})
			},
		},
		{
			name: "list",
			call: func(h *handler, ctx context.Context) error {
				_, err := h.TranscribeList(ctx, 10, "2024-01-01 00:00:00.000000", nil)
				return err
			},
		},
		{
			name: "get by reference id and language",
			call: func(h *handler, ctx context.Context) error {
				_, err := h.TranscribeGetByReferenceIDAndLanguage(ctx, id, "en-US")
				return err
			},
		},
		{
			name: "get on a cache miss",
			call: func(h *handler, ctx context.Context) error {
				_, err := h.TranscribeGet(ctx, id)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{utilHandler: mockUtil, db: newClosedTestDB(t), cache: mockCache}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()
			mockCache.EXPECT().TranscribeGet(gomock.Any(), id).Return(nil, errors.New("cache miss")).AnyTimes()

			err := tt.call(h, ctx)
			if err == nil {
				t.Fatalf("Wrong match. expect: err, got: ok")
			}
			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrDuplicateID) {
				t.Errorf("Wrong match. a closed database must not be classified as not found or duplicate. err: %v", err)
			}
		})
	}
}

func Test_transcript_closedDB(t *testing.T) {
	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000011")
	curTime := time.Date(2023, 1, 3, 21, 35, 2, 0, time.UTC)

	tests := []struct {
		name string

		call func(h *handler, ctx context.Context) error
	}{
		{
			name: "create",
			call: func(h *handler, ctx context.Context) error {
				return h.TranscriptCreate(ctx, &transcript.Transcript{Identity: commonidentity.Identity{ID: id}})
			},
		},
		{
			name: "delete",
			call: func(h *handler, ctx context.Context) error { return h.TranscriptDelete(ctx, id) },
		},
		{
			name: "update",
			call: func(h *handler, ctx context.Context) error {
				return h.TranscriptUpdate(ctx, id, map[transcript.Field]any{transcript.FieldMessage: "changed"})
			},
		},
		{
			name: "list",
			call: func(h *handler, ctx context.Context) error {
				_, err := h.TranscriptList(ctx, 10, "2024-01-01 00:00:00.000000", nil)
				return err
			},
		},
		{
			name: "get on a cache miss",
			call: func(h *handler, ctx context.Context) error {
				_, err := h.TranscriptGet(ctx, id)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{utilHandler: mockUtil, db: newClosedTestDB(t), cache: mockCache}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()
			mockCache.EXPECT().TranscriptGet(gomock.Any(), id).Return(nil, errors.New("cache miss")).AnyTimes()

			err := tt.call(h, ctx)
			if err == nil {
				t.Fatalf("Wrong match. expect: err, got: ok")
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. a closed database must not be classified as not found. err: %v", err)
			}
		})
	}
}

func Test_cacheFailureIsNotFatalForWrites(t *testing.T) {
	curTime := time.Date(2023, 1, 3, 21, 35, 2, 0, time.UTC)
	cacheErr := errors.New("cache unavailable")

	t.Run("transcribe create still succeeds and stores the row", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockCache := cachehandler.NewMockCacheHandler(mc)
		h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
		ctx := context.Background()

		id := uuid.Must(uuid.NewV4())
		mockUtil.EXPECT().TimeNow().Return(&curTime)
		mockCache.EXPECT().TranscribeSet(ctx, gomock.Any()).Return(cacheErr)

		if err := h.TranscribeCreate(ctx, &transcribe.Transcribe{Identity: commonidentity.Identity{ID: id}}); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		// the row is readable straight from the database
		res, err := h.transcribeGetFromDB(ctx, id)
		if err != nil || res.ID != id {
			t.Errorf("Wrong match. expect the stored row, got: %v (err: %v)", res, err)
		}
	})

	t.Run("transcript create still succeeds and stores the row", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockCache := cachehandler.NewMockCacheHandler(mc)
		h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
		ctx := context.Background()

		id := uuid.Must(uuid.NewV4())
		mockUtil.EXPECT().TimeNow().Return(&curTime)
		mockCache.EXPECT().TranscriptSet(ctx, gomock.Any()).Return(cacheErr)

		if err := h.TranscriptCreate(ctx, &transcript.Transcript{Identity: commonidentity.Identity{ID: id}}); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res, err := h.transcriptGetFromDB(ctx, id)
		if err != nil || res.ID != id {
			t.Errorf("Wrong match. expect the stored row, got: %v (err: %v)", res, err)
		}
	})

	t.Run("get falls back to the database when the cache fails and ignores a failing cache write", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockCache := cachehandler.NewMockCacheHandler(mc)
		h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
		ctx := context.Background()

		id := uuid.Must(uuid.NewV4())
		mockUtil.EXPECT().TimeNow().Return(&curTime)
		mockCache.EXPECT().TranscribeSet(ctx, gomock.Any()).Return(nil)
		if err := h.TranscribeCreate(ctx, &transcribe.Transcribe{Identity: commonidentity.Identity{ID: id}}); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		mockCache.EXPECT().TranscribeGet(ctx, id).Return(nil, cacheErr)
		mockCache.EXPECT().TranscribeSet(ctx, gomock.Any()).Return(cacheErr)

		res, err := h.TranscribeGet(ctx, id)
		if err != nil || res.ID != id {
			t.Errorf("Wrong match. expect the stored row, got: %v (err: %v)", res, err)
		}
	})
}

func Test_Get_notFound(t *testing.T) {
	missing := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-0000000000ff")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	mockCache.EXPECT().TranscribeGet(ctx, missing).Return(nil, errors.New("cache miss"))
	if _, err := h.TranscribeGet(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. TranscribeGet expect: ErrNotFound, got: %v", err)
	}

	mockCache.EXPECT().TranscriptGet(ctx, missing).Return(nil, errors.New("cache miss"))
	if _, err := h.TranscriptGet(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. TranscriptGet expect: ErrNotFound, got: %v", err)
	}

	if _, err := h.TranscribeGetByReferenceIDAndLanguage(ctx, missing, "xx-XX"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. GetByReferenceIDAndLanguage expect: ErrNotFound, got: %v", err)
	}
}

func Test_StreamingCreate_cacheError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{cache: mockCache}
	ctx := context.Background()

	cacheErr := errors.New("cache unavailable")
	mockCache.EXPECT().StreamingSet(ctx, gomock.Any()).Return(cacheErr)

	if err := h.StreamingCreate(ctx, nil); !errors.Is(err, cacheErr) {
		t.Errorf("Wrong match. expect: the cache error, got: %v", err)
	}
}

func Test_StreamingGet_cacheError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{cache: mockCache}
	ctx := context.Background()

	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000031")
	cacheErr := errors.New("cache unavailable")
	mockCache.EXPECT().StreamingGet(ctx, id).Return(nil, cacheErr)

	res, err := h.StreamingGet(ctx, id)
	if !errors.Is(err, cacheErr) || res != nil {
		t.Errorf("Wrong match. expect: nil and the cache error, got: %v and %v", res, err)
	}
}

func Test_isDuplicateKeyErr_cases(t *testing.T) {
	tests := []struct {
		name string

		err error

		expectRes bool
	}{
		{"nil", nil, false},
		{"sqlite unique violation text", errors.New("UNIQUE constraint failed: transcribe_transcribes.id"), true},
		{"wrapped sqlite unique violation", fmt.Errorf("exec: %w", errors.New("UNIQUE constraint failed: t.id")), true},
		{"mysql duplicate entry", &gomysql.MySQLError{Number: 1062, Message: "Duplicate entry"}, true},
		{"wrapped mysql duplicate entry", fmt.Errorf("exec: %w", &gomysql.MySQLError{Number: 1062}), true},
		{"other mysql error number", &gomysql.MySQLError{Number: 1048, Message: "Column cannot be null"}, false},
		{"unrelated error that mentions 1062", errors.New("error 1062 in some other layer"), false},
		{"ordinary error", errors.New("connection reset"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := isDuplicateKeyErr(tt.err); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
