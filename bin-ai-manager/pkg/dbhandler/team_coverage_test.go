package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func newTeamCoverageHandler(t *testing.T) (*handler, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	return &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}, mockUtil, mockCache
}

func Test_TeamCreate_Get_List_Update_Delete(t *testing.T) {
	resetTestDB(t)

	h, mockUtil, mockCache := newTeamCoverageHandler(t)
	ctx := context.Background()

	teamID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	otherID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000002")
	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	startMemberID := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001")

	createTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	updateTime := time.Date(2024, 1, 2, 3, 5, 5, 0, time.UTC)
	deleteTime := time.Date(2024, 1, 2, 3, 6, 5, 0, time.UTC)

	// create: the row is written, then read back and handed to the cache.
	mockUtil.EXPECT().TimeNow().Return(&createTime)
	mockCache.EXPECT().TeamSet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, got *team.Team) error {
		if got.ID != teamID || got.Name != "first" || got.CustomerID != customerID {
			t.Errorf("cache received a different team. got: %+v", got)
		}
		return nil
	})
	// stale lifecycle values on the input are reset by create.
	stale := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	err := h.TeamCreate(ctx, &team.Team{
		Identity:      identity.Identity{ID: teamID, CustomerID: customerID},
		TMUpdate:      &stale,
		TMDelete:      &stale,
		Name:          "first",
		Detail:        "detail",
		StartMemberID: startMemberID,
		Parameter:     map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatalf("TeamCreate failed. err: %v", err)
	}

	// a second team of another customer, which no later operation may touch.
	bystanderID := uuid.FromStringOrNil("a1000000-0000-0000-0000-0000000000b2")
	mockUtil.EXPECT().TimeNow().Return(&createTime)
	mockCache.EXPECT().TeamSet(ctx, gomock.Any()).Return(nil)
	if errBystander := h.TeamCreate(ctx, &team.Team{
		Identity: identity.Identity{ID: bystanderID, CustomerID: uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000002")},
		Name:     "bystander",
	}); errBystander != nil {
		t.Fatalf("bystander create failed. err: %v", errBystander)
	}

	// get with a cache hit never touches the database.
	cached := &team.Team{Identity: identity.Identity{ID: teamID}, Name: "from-cache"}
	mockCache.EXPECT().TeamGet(ctx, teamID).Return(cached, nil)
	res, err := h.TeamGet(ctx, teamID)
	if err != nil || res != cached {
		t.Fatalf("expected the cached team. res: %+v, err: %v", res, err)
	}

	// get with a cache miss reads the row and refills the cache.
	mockCache.EXPECT().TeamGet(ctx, teamID).Return(nil, errors.New("miss"))
	mockCache.EXPECT().TeamSet(ctx, gomock.Any()).Return(nil)
	res, err = h.TeamGet(ctx, teamID)
	if err != nil {
		t.Fatalf("TeamGet failed. err: %v", err)
	}
	if res.Name != "first" || res.Detail != "detail" || res.StartMemberID != startMemberID || res.Parameter["k"] != "v" {
		t.Errorf("row mismatch. got: %+v", res)
	}
	if res.TMCreate == nil || !res.TMCreate.Equal(createTime) || res.TMUpdate != nil || res.TMDelete != nil {
		t.Errorf("timestamps mismatch. got: %+v", res)
	}

	// a miss on an unknown id is reported as not found.
	mockCache.EXPECT().TeamGet(ctx, otherID).Return(nil, errors.New("miss"))
	if _, errGet := h.TeamGet(ctx, otherID); !errors.Is(errGet, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", errGet)
	}

	// update changes only the given field and stamps tm_update.
	mockUtil.EXPECT().TimeNow().Return(&updateTime)
	mockCache.EXPECT().TeamSet(ctx, gomock.Any()).Return(nil)
	if errUpdate := h.TeamUpdate(ctx, teamID, map[team.Field]any{team.FieldName: "renamed"}); errUpdate != nil {
		t.Fatalf("TeamUpdate failed. err: %v", errUpdate)
	}
	mockCache.EXPECT().TeamGet(ctx, teamID).Return(nil, errors.New("miss"))
	mockCache.EXPECT().TeamSet(ctx, gomock.Any()).Return(nil)
	res, err = h.TeamGet(ctx, teamID)
	if err != nil {
		t.Fatalf("TeamGet after update failed. err: %v", err)
	}
	if res.Name != "renamed" || res.Detail != "detail" {
		t.Errorf("update must change only the given field. got: %+v", res)
	}
	if res.TMUpdate == nil || !res.TMUpdate.Equal(updateTime) {
		t.Errorf("tm_update mismatch. got: %v", res.TMUpdate)
	}

	// list by customer returns the row.
	mockUtil.EXPECT().TimeGetCurTime().Return("2099-01-01T00:00:00.000000Z")
	list, err := h.TeamList(ctx, 10, "", map[team.Field]any{team.FieldCustomerID: customerID})
	if err != nil {
		t.Fatalf("TeamList failed. err: %v", err)
	}
	if len(list) != 1 || list[0].ID != teamID {
		t.Errorf("list mismatch. got: %v", list)
	}

	// a second team of the same customer makes the page size observable.
	secondID := uuid.FromStringOrNil("a1000000-0000-0000-0000-0000000000c3")
	secondTime := createTime.Add(time.Minute)
	mockUtil.EXPECT().TimeNow().Return(&secondTime)
	mockCache.EXPECT().TeamSet(ctx, gomock.Any()).Return(nil)
	if errSecond := h.TeamCreate(ctx, &team.Team{
		Identity: identity.Identity{ID: secondID, CustomerID: customerID},
		Name:     "second",
	}); errSecond != nil {
		t.Fatalf("second create failed. err: %v", errSecond)
	}
	list, err = h.TeamList(ctx, 1, "2099-01-01 00:00:00", map[team.Field]any{team.FieldCustomerID: customerID})
	if err != nil {
		t.Fatalf("TeamList with a page size of one failed. err: %v", err)
	}
	if len(list) != 1 || list[0].ID != secondID {
		t.Errorf("expected the newest team only. got: %v", list)
	}

	// the token is an exclusive upper bound on tm_create. sqlite compares the
	// stored text, so the bounds use the layout the sqlite driver writes.
	for _, tc := range []struct {
		token  string
		expect int
	}{
		{createTime.Format("2006-01-02 15:04:05.999999999-07:00"), 0},
		{createTime.Add(time.Second).Format("2006-01-02 15:04:05.999999999-07:00"), 1},
	} {
		list, err = h.TeamList(ctx, 10, tc.token, map[team.Field]any{team.FieldCustomerID: customerID})
		if err != nil {
			t.Fatalf("TeamList with token %s failed. err: %v", tc.token, err)
		}
		if len(list) != tc.expect {
			t.Errorf("token %s: expected %d rows, got: %d", tc.token, tc.expect, len(list))
		}
	}

	// delete stamps tm_delete, and the cache is refilled from the row.
	mockUtil.EXPECT().TimeNow().Return(&deleteTime)
	mockCache.EXPECT().TeamSet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, got *team.Team) error {
		if got.TMDelete == nil || !got.TMDelete.Equal(deleteTime) || got.TMUpdate == nil || !got.TMUpdate.Equal(deleteTime) {
			t.Errorf("cache must receive the deleted row with both timestamps set. got: %+v", got)
		}
		return nil
	})
	if errDelete := h.TeamDelete(ctx, teamID); errDelete != nil {
		t.Fatalf("TeamDelete failed. err: %v", errDelete)
	}

	// neither the update nor the delete may have touched the other team.
	var name string
	var untouched bool
	if errQuery := dbTest.QueryRow(`SELECT name, tm_update IS NULL AND tm_delete IS NULL FROM ai_teams WHERE id = ?`, bystanderID.Bytes()).Scan(&name, &untouched); errQuery != nil {
		t.Fatalf("could not read the bystander row. err: %v", errQuery)
	}
	if name != "bystander" || !untouched {
		t.Errorf("another team was modified. name: %s, untouched: %v", name, untouched)
	}
}

func Test_TeamUpdate_ExecError(t *testing.T) {
	resetTestDB(t)

	h, mockUtil, _ := newTeamCoverageHandler(t)
	ctx := context.Background()

	// an unknown column makes the statement fail; the error must surface and
	// the cache must not be touched (no TeamSet expectation is registered).
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now)

	err := h.TeamUpdate(ctx, uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001"), map[team.Field]any{team.Field("no_such_column"): "x"})
	if err == nil {
		t.Errorf("expected an error for an unknown column")
	}
}

func Test_TeamCreate_ExecError(t *testing.T) {
	resetTestDB(t)

	h, mockUtil, _ := newTeamCoverageHandler(t)
	ctx := context.Background()

	teamID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	// the first insert succeeds, the second one violates the primary key.
	mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
	h.cache.(*cachehandler.MockCacheHandler).EXPECT().TeamSet(ctx, gomock.Any()).Return(nil)

	if err := h.TeamCreate(ctx, &team.Team{Identity: identity.Identity{ID: teamID}}); err != nil {
		t.Fatalf("first create failed. err: %v", err)
	}
	if err := h.TeamCreate(ctx, &team.Team{Identity: identity.Identity{ID: teamID}}); err == nil {
		t.Errorf("expected a duplicate id to be rejected")
	}
}
