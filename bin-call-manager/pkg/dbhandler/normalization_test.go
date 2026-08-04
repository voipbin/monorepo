package dbhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

// These tests pin the write-side empty-collection normalization recorded in the
// R-A table in normalization.go (VOIP-1078 plan §5 R-A, Phase 2.5/3).
//
// They assert the RAW stored column value, not the value read back through the
// dbhandler, because the read path normalizes nil to an empty collection and
// would therefore mask a regression from "[]" to "null". The pre-existing
// Test_ConfbridgeSetFlags / groupcall setter tests only cover non-nil input and
// so are NOT a safety net for this behavior — that gap is why these exist.
//
// SQLite executes these statements fine: none of the three sites uses a MySQL
// JSON function, they are plain column assignments.

// selectRawColumn reads a single column back as raw text, bypassing the
// dbhandler read path (and thus its nil normalization).
func selectRawColumn(t *testing.T, query string, arg any) string {
	t.Helper()

	var got []byte
	if err := dbTest.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatalf("could not read raw column. err: %v", err)
	}
	return string(got)
}

func Test_normalization_ConfbridgeSetFlags_nilStoresEmptyArray(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	id := uuid.FromStringOrNil("b1000000-0000-11f0-9e2b-0242ac110002")
	ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

	mockUtil.EXPECT().TimeNow().Return(ts)
	mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
	if err := h.ConfbridgeCreate(ctx, &confbridge.Confbridge{
		Identity: commonidentity.Identity{ID: id},
	}); err != nil {
		t.Fatalf("could not create confbridge. err: %v", err)
	}

	// nil flags must be stored as "[]", matching the pre-migration behavior at
	// confbridge.go:451-453, not as the JSON literal "null".
	mockUtil.EXPECT().TimeNow().Return(ts)
	mockCache.EXPECT().ConfbridgeSet(ctx, gomock.Any())
	if err := h.ConfbridgeSetFlags(ctx, id, nil); err != nil {
		t.Fatalf("could not set flags. err: %v", err)
	}

	if got := selectRawColumn(t, "SELECT flags FROM call_confbridges WHERE id = ?", id.Bytes()); got != "[]" {
		t.Errorf("nil flags must normalize to the JSON literal []. expect: [], got: %s", got)
	}
}

func Test_normalization_GroupcallSetCallIDs_nilStoresEmptyArray(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	id := uuid.FromStringOrNil("b2000000-0000-11f0-9e2b-0242ac110002")
	ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

	mockUtil.EXPECT().TimeNow().Return(ts)
	mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
	if err := h.GroupcallCreate(ctx, &groupcall.Groupcall{
		Identity: commonidentity.Identity{ID: id},
	}); err != nil {
		t.Fatalf("could not create groupcall. err: %v", err)
	}

	// nil callIDs must be stored as "[]", matching groupcall.go:336-338.
	mockUtil.EXPECT().TimeNow().Return(ts)
	mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
	if err := h.GroupcallSetCallIDsAndCallCountAndDialIndex(ctx, id, nil, 0, 0); err != nil {
		t.Fatalf("could not set call ids. err: %v", err)
	}

	if got := selectRawColumn(t, "SELECT call_ids FROM call_groupcalls WHERE id = ?", id.Bytes()); got != "[]" {
		t.Errorf("nil callIDs must normalize to the JSON literal []. expect: [], got: %s", got)
	}
}

func Test_normalization_GroupcallSetGroupcallIDs_nilStoresEmptyArray(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	id := uuid.FromStringOrNil("b3000000-0000-11f0-9e2b-0242ac110002")
	ts := testhelper.TimePtr("2026-08-04T01:02:03.000000Z")

	mockUtil.EXPECT().TimeNow().Return(ts)
	mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
	if err := h.GroupcallCreate(ctx, &groupcall.Groupcall{
		Identity: commonidentity.Identity{ID: id},
	}); err != nil {
		t.Fatalf("could not create groupcall. err: %v", err)
	}

	// nil groupcallIDs must be stored as "[]", matching groupcall.go:366-368.
	mockUtil.EXPECT().TimeNow().Return(ts)
	mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
	if err := h.GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx, id, nil, 0, 0); err != nil {
		t.Fatalf("could not set groupcall ids. err: %v", err)
	}

	if got := selectRawColumn(t, "SELECT groupcall_ids FROM call_groupcalls WHERE id = ?", id.Bytes()); got != "[]" {
		t.Errorf("nil groupcallIDs must normalize to the JSON literal []. expect: [], got: %s", got)
	}
}
