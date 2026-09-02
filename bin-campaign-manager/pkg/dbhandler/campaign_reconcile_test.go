package dbhandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/cachehandler"
)

// Test_CampaignListDeletedSince verifies the date-range filter, the DESC
// order, and the NULL-comparison exclusion of live campaigns, against the
// real SQLite test-schema harness (see main_test.go).
func Test_CampaignListDeletedSince(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		util:  mockUtil,
		db:    dbTest,
		cache: mockCache,
	}

	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	liveCampaign := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")},
		Status:   campaign.StatusStop,
	}
	recentlyDeleted := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002")},
		Status:   campaign.StatusStop,
	}
	olderDeleted := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000003")},
		Status:   campaign.StatusStop,
	}
	outOfWindowDeleted := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000004")},
		Status:   campaign.StatusStop,
	}

	mockCache.EXPECT().CampaignSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()

	for _, c := range []*campaign.Campaign{liveCampaign, recentlyDeleted, olderDeleted, outOfWindowDeleted} {
		if err := h.CampaignCreate(ctx, c); err != nil {
			t.Fatalf("Could not create campaign. err: %v", err)
		}
	}

	since := now.Add(-24 * time.Hour)
	tmDeletes := map[uuid.UUID]time.Time{
		recentlyDeleted.ID:    now.Add(-1 * time.Hour),  // within window, newest
		olderDeleted.ID:       now.Add(-12 * time.Hour), // within window, older
		outOfWindowDeleted.ID: now.Add(-48 * time.Hour), // before the cutoff
	}
	for id, ts := range tmDeletes {
		tsCopy := ts
		if _, err := dbTest.ExecContext(ctx, "UPDATE campaign_campaigns SET tm_delete = ? WHERE id = ?", tsCopy, id.Bytes()); err != nil {
			t.Fatalf("Could not set tm_delete. err: %v", err)
		}
	}

	res, err := h.CampaignListDeletedSince(ctx, since, 10)
	if err != nil {
		t.Fatalf("Could not list deleted campaigns. err: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("Wrong match. expect: 2 campaigns (live and out-of-window excluded), got: %d", len(res))
	}

	// DESC order: most-recently-deleted first.
	if res[0].ID != recentlyDeleted.ID {
		t.Errorf("Wrong match. expect first (most recent): %s, got: %s", recentlyDeleted.ID, res[0].ID)
	}
	if res[1].ID != olderDeleted.ID {
		t.Errorf("Wrong match. expect second: %s, got: %s", olderDeleted.ID, res[1].ID)
	}

	for _, c := range res {
		if c.ID == liveCampaign.ID {
			t.Errorf("Wrong match. live campaign (tm_delete IS NULL) must be excluded via NULL-comparison semantics")
		}
		if c.ID == outOfWindowDeleted.ID {
			t.Errorf("Wrong match. campaign deleted before the cutoff must be excluded")
		}
	}

	// limit is respected
	limited, err := h.CampaignListDeletedSince(ctx, since, 1)
	if err != nil {
		t.Fatalf("Could not list deleted campaigns with limit. err: %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("Wrong match. expect: 1 campaign with limit=1, got: %d", len(limited))
	}
	if limited[0].ID != recentlyDeleted.ID {
		t.Errorf("Wrong match. expect the single most-recently-deleted row, got: %s", limited[0].ID)
	}
}
