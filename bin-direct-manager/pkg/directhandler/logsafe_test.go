package directhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/cachehandler"
	"monorepo/bin-direct-manager/pkg/dbhandler"
)

const (
	secretLogOldHash = "direct.fedcba987654-SENTINEL-MUST-NOT-LEAK-old1"
)

func newLogTestHandler(mc *gomock.Controller) (*directHandler, *dbhandler.MockDBHandler, *cachehandler.MockCacheHandler, *notifyhandler.MockNotifyHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &directHandler{
		utilHandler:   utilhandler.NewMockUtilHandler(mc),
		db:            mockDB,
		cache:         mockCache,
		notifyhandler: mockNotify,
	}
	return h, mockDB, mockCache, mockNotify
}

func Test_GetByHash_noHashInLogs(t *testing.T) {
	d := &direct.Direct{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
		},
		ResourceType: "extension",
		Hash:         secretLogHash,
	}

	tests := []struct {
		name      string
		setup     func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockCache *cachehandler.MockCacheHandler)
		expectMsg string
		expectErr bool
	}{
		{
			name: "cache hit",
			setup: func(ctx context.Context, _ *dbhandler.MockDBHandler, c *cachehandler.MockCacheHandler) {
				c.EXPECT().DirectGetByHash(ctx, secretLogHash).Return(d, nil)
			},
			expectMsg: "Retrieved direct from cache.",
		},
		{
			name: "cache miss, db hit",
			setup: func(ctx context.Context, db *dbhandler.MockDBHandler, c *cachehandler.MockCacheHandler) {
				c.EXPECT().DirectGetByHash(ctx, secretLogHash).Return(nil, dbhandler.ErrNotFound)
				db.EXPECT().DirectGetByHash(ctx, secretLogHash).Return(d, nil)
				c.EXPECT().DirectSetByHash(ctx, secretLogHash, d).Return(nil)
			},
			expectMsg: "Retrieved direct from DB and cached.",
		},
		{
			name: "cache miss, db not found",
			setup: func(ctx context.Context, db *dbhandler.MockDBHandler, c *cachehandler.MockCacheHandler) {
				c.EXPECT().DirectGetByHash(ctx, secretLogHash).Return(nil, dbhandler.ErrNotFound)
				db.EXPECT().DirectGetByHash(ctx, secretLogHash).Return(nil, dbhandler.ErrNotFound)
			},
			expectMsg: "Could not get direct by hash.",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := newSecretLogHook(t)
			h, mockDB, mockCache, _ := newLogTestHandler(mc)
			ctx := context.Background()
			tt.setup(ctx, mockDB, mockCache)

			_, err := h.GetByHash(ctx, secretLogHash)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}

			assertNoSecretInLogs(t, hook, tt.expectMsg, secretLogHash, secretLogHashFragment)
		})
	}
}

func Test_Regenerate_noHashInLogs(t *testing.T) {
	id := uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5")
	current := &direct.Direct{
		Identity: commonidentity.Identity{ID: id},
		Hash:     secretLogOldHash,
	}
	regenerated := &direct.Direct{
		Identity: commonidentity.Identity{ID: id},
		Hash:     secretLogHash,
	}

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		hook := newSecretLogHook(t)
		h, mockDB, mockCache, mockNotify := newLogTestHandler(mc)
		ctx := context.Background()

		var newHash string
		mockDB.EXPECT().DirectGet(ctx, id).Return(current, nil)
		mockDB.EXPECT().DirectUpdate(ctx, id, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[direct.Field]any) error {
			newHash, _ = fields[direct.FieldHash].(string)
			return nil
		})
		mockCache.EXPECT().DirectDeleteByHash(ctx, secretLogOldHash).Return(nil)
		mockDB.EXPECT().DirectGet(ctx, id).Return(regenerated, nil)
		mockNotify.EXPECT().PublishEvent(ctx, direct.EventTypeDirectRegenerated, regenerated)

		if _, err := h.Regenerate(ctx, id); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if newHash == "" {
			t.Fatalf("the generated hash was not captured")
		}

		assertNoSecretInLogs(t, hook, "Regenerated the direct hash.", secretLogHash, secretLogHashFragment, secretLogOldHash, newHash)
	})

	t.Run("update failure on every attempt", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		hook := newSecretLogHook(t)
		h, mockDB, _, _ := newLogTestHandler(mc)
		ctx := context.Background()

		var hashes []string
		mockDB.EXPECT().DirectGet(ctx, id).Return(current, nil)
		mockDB.EXPECT().DirectUpdate(ctx, id, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[direct.Field]any) error {
			hash, _ := fields[direct.FieldHash].(string)
			hashes = append(hashes, hash)
			return errors.New("update failure")
		}).Times(3)

		if _, err := h.Regenerate(ctx, id); err == nil {
			t.Fatalf("Wrong match. expect: error, got: nil")
		}
		if len(hashes) != 3 {
			t.Fatalf("expected 3 captured hashes, got %d", len(hashes))
		}

		secrets := append([]string{secretLogOldHash, secretLogHashFragment}, hashes...)
		assertNoSecretInLogs(t, hook, "Could not update the direct.", secrets...)
		assertEntryHasFieldKeys(t, hook, []string{"hash"})
	})
}

// assertEntryHasFieldKeys requires a collected entry that carries the given
// field_keys, so the test fails if the key list is dropped from the log.
func assertEntryHasFieldKeys(t *testing.T, hook *logrustest.Hook, want []string) {
	t.Helper()
	for _, entry := range hook.AllEntries() {
		if got, ok := entry.Data["field_keys"]; ok && reflect.DeepEqual(got, want) {
			return
		}
	}
	t.Errorf("no log entry carried field_keys %v", want)
}

func Test_dbList_noHashInLogs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newSecretLogHook(t)
	h, mockDB, _, _ := newLogTestHandler(mc)
	ctx := context.Background()

	filters := map[direct.Field]any{
		direct.FieldHash: secretLogHash,
	}
	mockDB.EXPECT().DirectGets(ctx, uint64(10), "", filters).Return(nil, errors.New("list failure"))

	if _, err := h.dbList(ctx, 10, "", filters); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	// the mask keeps the leading 12 chars, so only the fragment is checked
	// together with the whole sentinel.
	assertNoSecretInLogs(t, hook, "Could not get directs info.", secretLogHash, secretLogHashFragment)

	// the caller's filter map must not be modified by the masking.
	if filters[direct.FieldHash] != secretLogHash {
		t.Errorf("dbList modified the caller's filters: %v", filters)
	}
}

func Test_dbUpdate_noHashInLogs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newSecretLogHook(t)
	h, mockDB, _, _ := newLogTestHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5")

	fields := map[direct.Field]any{direct.FieldHash: secretLogHash}
	mockDB.EXPECT().DirectUpdate(ctx, id, fields).Return(errors.New("update failure"))

	if err := h.dbUpdate(ctx, id, fields); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	assertNoSecretInLogs(t, hook, "Could not update the direct.", secretLogHash, secretLogHashFragment)
	assertEntryHasFieldKeys(t, hook, []string{"hash"})
}

func Test_dbGetByHash_noHashInLogs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newSecretLogHook(t)
	h, mockDB, _, _ := newLogTestHandler(mc)
	ctx := context.Background()

	mockDB.EXPECT().DirectGetByHash(ctx, secretLogHash).Return(nil, errors.New("get failure"))

	if _, err := h.dbGetByHash(ctx, secretLogHash); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	assertNoSecretInLogs(t, hook, "Could not get direct by hash.", secretLogHash, secretLogHashFragment)
}

func Test_dbCreate_noHashInLogs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newSecretLogHook(t)
	h, mockDB, _, _ := newLogTestHandler(mc)
	ctx := context.Background()

	d := &direct.Direct{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5")},
		Hash:     secretLogHash,
	}
	mockDB.EXPECT().DirectCreate(ctx, d).Return(errors.New("create failure"))

	if err := h.dbCreate(ctx, d); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	assertNoSecretInLogs(t, hook, "Could not create a new direct.", secretLogHash, secretLogHashFragment)
}

func Test_Gets_noHashInLogs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newSecretLogHook(t)
	h, mockDB, _, _ := newLogTestHandler(mc)
	ctx := context.Background()

	filters := map[direct.Field]any{
		direct.FieldHash: secretLogHash,
	}
	mockDB.EXPECT().DirectGets(ctx, uint64(10), "", filters).Return(nil, errors.New("list failure"))

	if _, err := h.Gets(ctx, 10, "", filters); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	assertNoSecretInLogs(t, hook, "Could not get directs info.", secretLogHash, secretLogHashFragment)
}
