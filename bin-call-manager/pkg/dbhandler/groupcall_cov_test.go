package dbhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

func Test_GroupcallDelete_cov(t *testing.T) {
	resetTestDB(t)

	g := &groupcall.Groupcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a3f0c4e2-1b2d-11f1-8a5e-0b3f6c1d2e01")}}

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	mockUtil.EXPECT().TimeNow().Return(testhelper.TimePtr("2023-01-18T03:22:18.995000Z"))
	mockCache.EXPECT().GroupcallSet(ctx, gomock.Cond(func(x *groupcall.Groupcall) bool { return x != nil && x.ID == g.ID })).Return(nil)
	if err := h.GroupcallCreate(ctx, g); err != nil {
		t.Fatalf("could not create the groupcall. err: %v", err)
	}

	deleted := testhelper.TimePtr("2023-01-18T03:22:19.995000Z")
	mockUtil.EXPECT().TimeNow().Return(deleted).MinTimes(1)
	mockCache.EXPECT().GroupcallSet(ctx, gomock.Cond(func(x *groupcall.Groupcall) bool { return x != nil && x.ID == g.ID })).Return(nil)
	if err := h.GroupcallDelete(ctx, g.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.groupcallGetFromDB(ctx, g.ID)
	if err != nil {
		t.Fatalf("could not get the groupcall. err: %v", err)
	}
	if res.TMDelete == nil || !res.TMDelete.Equal(*deleted) || res.TMUpdate == nil || !res.TMUpdate.Equal(*deleted) {
		t.Errorf("Wrong match. expect tm_update/tm_delete: %v, got: %v / %v", deleted, res.TMUpdate, res.TMDelete)
	}
}
