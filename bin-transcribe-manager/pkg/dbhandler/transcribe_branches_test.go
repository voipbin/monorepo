package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
)

// newPrivateTestDB returns an empty database that only this test uses, with the test schema
// applied. It has one connection, because every extra connection of a private in-memory
// database would be a new, empty database.
func newPrivateTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:private_%s?mode=memory&cache=private", uuid.Must(uuid.NewV4())))
	if err != nil {
		t.Fatalf("could not open the test database. err: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	files, err := filepath.Glob("../../scripts/database_scripts_test/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("could not find the schema scripts. files: %v, err: %v", files, err)
	}
	for _, f := range files {
		contents, errRead := os.ReadFile(f)
		if errRead != nil {
			t.Fatalf("could not read the schema script. file: %s, err: %v", f, errRead)
		}
		if _, errExec := db.Exec(string(contents)); errExec != nil {
			t.Fatalf("could not apply the schema script. file: %s, err: %v", f, errExec)
		}
	}

	return db
}

func newBranchesHandler(t *testing.T, db *sql.DB) (*handler, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	return &handler{utilHandler: mockUtil, db: db, cache: mockCache}, mockUtil, mockCache
}

func Test_TranscribeList_emptyTokenUsesCurrentTime(t *testing.T) {
	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000101")
	customerID := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000102")

	db := newPrivateTestDB(t)
	if _, err := db.Exec(`insert into transcribe_transcribes(id, customer_id, tm_create) values (?, ?, '2023-01-03 21:35:02.000000')`, id.Bytes(), customerID.Bytes()); err != nil {
		t.Fatalf("could not insert the row. err: %v", err)
	}

	tests := []struct {
		name string

		currentTime string

		expectCount int
	}{
		{
			name:        "the row was created before the current time",
			currentTime: "2023-01-04 00:00:00.000000",
			expectCount: 1,
		},
		{
			name:        "the row was created after the current time",
			currentTime: "2023-01-03 00:00:00.000000",
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockUtil, _ := newBranchesHandler(t, db)
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.currentTime)

			res, err := h.TranscribeList(context.Background(), 10, "", map[transcribe.Field]any{transcribe.FieldCustomerID: customerID})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(res) != tt.expectCount {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectCount, len(res))
			}
		})
	}
}

func Test_TranscribeList_unsupportedFilter(t *testing.T) {
	h, _, _ := newBranchesHandler(t, newPrivateTestDB(t))

	res, err := h.TranscribeList(context.Background(), 10, "2024-01-01 00:00:00.000000", map[transcribe.Field]any{transcribe.FieldCustomerID: struct{}{}})
	if err == nil || !strings.Contains(err.Error(), "could not apply filters") {
		t.Errorf("Wrong match. expect: filter error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

func Test_TranscribeUpdate_branches(t *testing.T) {
	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000103")
	curTime := time.Date(2023, 1, 3, 21, 35, 2, 0, time.UTC)

	t.Run("empty fields change nothing and do not touch the database or the cache", func(t *testing.T) {
		// no expectation is set on the util handler and the cache: any call fails the test
		h, _, _ := newBranchesHandler(t, newClosedTestDB(t))

		if err := h.TranscribeUpdate(context.Background(), id, map[transcribe.Field]any{}); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("a value that cannot be converted for the database is rejected before any query", func(t *testing.T) {
		// the closed database proves that the failure comes before the query is executed
		h, mockUtil, _ := newBranchesHandler(t, newClosedTestDB(t))
		mockUtil.EXPECT().TimeNow().Return(&curTime)

		err := h.TranscribeUpdate(context.Background(), id, map[transcribe.Field]any{transcribe.FieldStreamingIDs: []chan int{make(chan int)}})
		if err == nil || !strings.Contains(err.Error(), "prepare fields failed") {
			t.Errorf("Wrong match. expect: prepare fields error, got: %v", err)
		}
	})
}

func Test_TranscribeGet_rowBranches(t *testing.T) {
	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000104")
	customerID := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000105")
	referenceID := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000106")

	t.Run("a row without streaming ids is returned with an empty list", func(t *testing.T) {
		db := newPrivateTestDB(t)
		if _, err := db.Exec(`insert into transcribe_transcribes(id, customer_id, tm_create) values (?, ?, '2023-01-03 21:35:02.000000')`, id.Bytes(), customerID.Bytes()); err != nil {
			t.Fatalf("could not insert the row. err: %v", err)
		}
		h, _, _ := newBranchesHandler(t, db)

		res, err := h.transcribeGetFromDB(context.Background(), id)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.StreamingIDs == nil || len(res.StreamingIDs) != 0 {
			t.Errorf("Wrong match. expect: an empty, non-nil list, got: %#v", res.StreamingIDs)
		}
	})

	t.Run("a stored value that cannot be scanned is reported by every reader", func(t *testing.T) {
		db := newPrivateTestDB(t)
		// the customer id is 3 bytes long, so the row cannot be scanned into a uuid
		if _, err := db.Exec(`insert into transcribe_transcribes(id, customer_id, reference_id, language, tm_create) values (?, ?, ?, 'en-US', '2023-01-03 21:35:02.000000')`, id.Bytes(), []byte{1, 2, 3}, referenceID.Bytes()); err != nil {
			t.Fatalf("could not insert the row. err: %v", err)
		}
		h, _, mockCache := newBranchesHandler(t, db)
		mockCache.EXPECT().TranscribeGet(gomock.Any(), id).Return(nil, errors.New("cache miss"))
		ctx := context.Background()

		if _, err := h.TranscribeGet(ctx, id); err == nil || !strings.Contains(err.Error(), "could not scan the row") {
			t.Errorf("Wrong match. expect: scan error from get, got: %v", err)
		}
		if _, err := h.TranscribeGetByReferenceIDAndLanguage(ctx, referenceID, "en-US"); err == nil || !strings.Contains(err.Error(), "could not scan the row") {
			t.Errorf("Wrong match. expect: scan error from get by reference, got: %v", err)
		}
		if _, err := h.TranscribeList(ctx, 10, "2024-01-01 00:00:00.000000", nil); err == nil || !strings.Contains(err.Error(), "could not scan the row") {
			t.Errorf("Wrong match. expect: scan error from list, got: %v", err)
		}
	})
}

func Test_transcribeUpdateToCache_notFound(t *testing.T) {
	h, _, _ := newBranchesHandler(t, newPrivateTestDB(t))

	// the row does not exist, so nothing is written to the cache (no expectation on the cache)
	err := h.transcribeUpdateToCache(context.Background(), uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000107"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
}
