package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-transcribe-manager/models/transcript"
)

func Test_TranscriptList_emptyTokenUsesCurrentTime(t *testing.T) {
	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000201")
	transcribeID := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000202")

	db := newPrivateTestDB(t)
	if _, err := db.Exec(`insert into transcribe_transcripts(id, transcribe_id, tm_create) values (?, ?, '2023-01-03 21:35:02.000000')`, id.Bytes(), transcribeID.Bytes()); err != nil {
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

			res, err := h.TranscriptList(context.Background(), 10, "", map[transcript.Field]any{transcript.FieldTranscribeID: transcribeID})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(res) != tt.expectCount {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectCount, len(res))
			}
		})
	}
}

func Test_TranscriptList_unsupportedFilter(t *testing.T) {
	h, _, _ := newBranchesHandler(t, newPrivateTestDB(t))

	res, err := h.TranscriptList(context.Background(), 10, "2024-01-01 00:00:00.000000", map[transcript.Field]any{transcript.FieldTranscribeID: struct{}{}})
	if err == nil || !strings.Contains(err.Error(), "could not apply filters") {
		t.Errorf("Wrong match. expect: filter error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

func Test_TranscriptUpdate_branches(t *testing.T) {
	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000203")

	t.Run("empty fields change nothing and do not touch the database or the cache", func(t *testing.T) {
		h, _, _ := newBranchesHandler(t, newClosedTestDB(t))

		if err := h.TranscriptUpdate(context.Background(), id, map[transcript.Field]any{}); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("a value that cannot be converted for the database is rejected before any query", func(t *testing.T) {
		h, _, _ := newBranchesHandler(t, newClosedTestDB(t))

		err := h.TranscriptUpdate(context.Background(), id, map[transcript.Field]any{transcript.FieldMessage: []chan int{make(chan int)}})
		if err == nil || !strings.Contains(err.Error(), "prepare fields failed") {
			t.Errorf("Wrong match. expect: prepare fields error, got: %v", err)
		}
	})
}

func Test_TranscriptGet_cacheAndRowBranches(t *testing.T) {
	id := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000204")
	transcribeID := uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000205")

	t.Run("a cached transcript is returned without reading the database", func(t *testing.T) {
		// the closed database fails any query, so a result proves that it came from the cache
		h, _, mockCache := newBranchesHandler(t, newClosedTestDB(t))
		cached := &transcript.Transcript{Identity: commonidentity.Identity{ID: id}, Message: "from cache"}
		mockCache.EXPECT().TranscriptGet(gomock.Any(), id).Return(cached, nil)

		res, err := h.TranscriptGet(context.Background(), id)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res != cached {
			t.Errorf("Wrong match. expect: the cached transcript, got: %v", res)
		}
	})

	t.Run("a stored value that cannot be scanned is reported by every reader", func(t *testing.T) {
		db := newPrivateTestDB(t)
		// the customer id is 3 bytes long, so the row cannot be scanned into a uuid
		if _, err := db.Exec(`insert into transcribe_transcripts(id, customer_id, transcribe_id, tm_create) values (?, ?, ?, '2023-01-03 21:35:02.000000')`, id.Bytes(), []byte{1, 2, 3}, transcribeID.Bytes()); err != nil {
			t.Fatalf("could not insert the row. err: %v", err)
		}
		h, _, mockCache := newBranchesHandler(t, db)
		mockCache.EXPECT().TranscriptGet(gomock.Any(), id).Return(nil, errors.New("cache miss"))
		ctx := context.Background()

		if _, err := h.TranscriptGet(ctx, id); err == nil || !strings.Contains(err.Error(), "could not scan the row") {
			t.Errorf("Wrong match. expect: scan error from get, got: %v", err)
		}
		if _, err := h.TranscriptList(ctx, 10, "2024-01-01 00:00:00.000000", nil); err == nil || !strings.Contains(err.Error(), "could not scan the row") {
			t.Errorf("Wrong match. expect: scan error from list, got: %v", err)
		}
	})
}

func Test_transcriptUpdateToCache_notFound(t *testing.T) {
	h, _, _ := newBranchesHandler(t, newPrivateTestDB(t))

	// the row does not exist, so nothing is written to the cache (no expectation on the cache)
	err := h.transcriptUpdateToCache(context.Background(), uuid.FromStringOrNil("8d2e0b10-0000-11ee-9a11-000000000206"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
}
