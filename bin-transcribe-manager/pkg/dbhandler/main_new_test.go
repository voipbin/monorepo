package dbhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
)

// Test_NewHandler_usesGivenDependencies checks the handler built by NewHandler reads from the
// given cache first and falls back to the given database when the cache read fails.
func Test_NewHandler_usesGivenDependencies(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)
	ctx := context.Background()

	// unique ids, because the in-memory database is shared with the other tests of the package
	id := uuid.Must(uuid.NewV4())
	row := &transcribe.Transcribe{
		Identity:      commonidentity.Identity{ID: id, CustomerID: uuid.Must(uuid.NewV4())},
		ReferenceType: transcribe.ReferenceTypeCall,
		ReferenceID:   uuid.Must(uuid.NewV4()),
		HostID:        uuid.Must(uuid.NewV4()),
		Language:      "en-US",
		Direction:     transcribe.DirectionBoth,
	}
	mockCache.EXPECT().TranscribeSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	if err := h.TranscribeCreate(ctx, row); err != nil {
		t.Fatalf("TranscribeCreate() returned an error. err: %v", err)
	}

	t.Run("cache hit does not read the database", func(t *testing.T) {
		cached := &transcribe.Transcribe{Identity: commonidentity.Identity{ID: id}, Language: "ko-KR"}
		mockCache.EXPECT().TranscribeGet(gomock.Any(), id).Return(cached, nil)

		got, err := h.TranscribeGet(ctx, id)
		if err != nil {
			t.Fatalf("TranscribeGet() returned an error. err: %v", err)
		}
		if got != cached {
			t.Errorf("TranscribeGet() did not return the cached transcribe. got language: %s", got.Language)
		}
	})

	t.Run("cache error falls back to the database", func(t *testing.T) {
		mockCache.EXPECT().TranscribeGet(gomock.Any(), id).Return(nil, fmt.Errorf("cache unavailable"))

		got, err := h.TranscribeGet(ctx, id)
		if err != nil {
			t.Fatalf("TranscribeGet() returned an error. err: %v", err)
		}
		if got.ID != id || got.Language != "en-US" {
			t.Errorf("TranscribeGet() did not return the database row. got: %+v", got)
		}
	})
}
