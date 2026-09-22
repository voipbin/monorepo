package dbhandler

// UUID-prefix isolation for this file (shared package dbTest DB): all fixtures
// use the f0xxxxxx-1539-... block, distinct from the 4bxx/a2xx blocks used by
// agent_test.go / agent_additional_test.go. See voipbin-dbhandler-entity skill.

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
)

// createAgentForReserve inserts an available agent row for reserve tests.
func createAgentForReserve(t *testing.T, h *handler, mockUtil *utilhandler.MockUtilHandler, mockCache *cachehandler.MockCacheHandler, id uuid.UUID, status agent.Status) {
	ctx := context.Background()
	mockUtil.EXPECT().TimeNow().Return(testTime("2020-04-18T03:22:17.995000Z")).AnyTimes()
	mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any()).AnyTimes()

	a := &agent.Agent{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("f0000000-1539-0000-0000-0000000000ff"),
		},
		Username: "reserve-" + id.String(),
		Status:   status,
	}
	if err := h.AgentCreate(ctx, a); err != nil {
		t.Fatalf("could not create agent for reserve test. err: %v", err)
	}
}

func Test_AgentReserve(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	agentID := uuid.FromStringOrNil("f0000001-1539-0000-0000-000000000001")
	refID := uuid.FromStringOrNil("f0000001-1539-0000-0000-0000000000a1")
	createAgentForReserve(t, h, mockUtil, mockCache, agentID, agent.StatusAvailable)

	// first reserve wins (CAS)
	ok, err := h.AgentReserve(ctx, agentID, "queuecall", refID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !ok {
		t.Fatalf("expected reserve to win, got false")
	}

	// verify persisted reserve fields
	res, err := h.agentGetFromDB(ctx, agentID)
	if err != nil {
		t.Fatalf("could not get agent. err: %v", err)
	}
	if res.ReserveReferenceType != "queuecall" {
		t.Errorf("wrong reserve_reference_type. got: %s", res.ReserveReferenceType)
	}
	if res.ReserveReferenceID != refID {
		t.Errorf("wrong reserve_reference_id. got: %s", res.ReserveReferenceID)
	}
	if res.TMReserve == nil {
		t.Errorf("expected tm_reserve to be set")
	}

	// second reserve by a different token loses (already reserved)
	refID2 := uuid.FromStringOrNil("f0000001-1539-0000-0000-0000000000a2")
	ok2, err := h.AgentReserve(ctx, agentID, "queuecall", refID2)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if ok2 {
		t.Fatalf("expected second reserve to lose, got true")
	}

	// clean up the reservation so it does not leak into the shared-DB
	// sweep test's global count.
	if err := h.AgentReserveRelease(ctx, agentID, refID); err != nil {
		t.Fatalf("could not release reservation on cleanup. err: %v", err)
	}
}

func Test_AgentReserve_NotAvailable(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	agentID := uuid.FromStringOrNil("f0000002-1539-0000-0000-000000000001")
	refID := uuid.FromStringOrNil("f0000002-1539-0000-0000-0000000000a1")
	createAgentForReserve(t, h, mockUtil, mockCache, agentID, agent.StatusBusy)

	// a busy agent can not be reserved
	ok, err := h.AgentReserve(ctx, agentID, "queuecall", refID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if ok {
		t.Fatalf("expected reserve on busy agent to lose, got true")
	}
}

func Test_AgentReserveRelease(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	agentID := uuid.FromStringOrNil("f0000003-1539-0000-0000-000000000001")
	refID := uuid.FromStringOrNil("f0000003-1539-0000-0000-0000000000a1")
	createAgentForReserve(t, h, mockUtil, mockCache, agentID, agent.StatusAvailable)

	ok, err := h.AgentReserve(ctx, agentID, "queuecall", refID)
	if err != nil || !ok {
		t.Fatalf("could not reserve. ok: %v, err: %v", ok, err)
	}

	// releasing with the wrong owner token must not clear
	wrongRef := uuid.FromStringOrNil("f0000003-1539-0000-0000-0000000000ff")
	if err := h.AgentReserveRelease(ctx, agentID, wrongRef); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	res, err := h.agentGetFromDB(ctx, agentID)
	if err != nil {
		t.Fatalf("could not get agent. err: %v", err)
	}
	if res.ReserveReferenceID != refID {
		t.Errorf("reserve should still be held by original owner. got: %s", res.ReserveReferenceID)
	}

	// releasing with the correct owner token clears the reserve
	if err := h.AgentReserveRelease(ctx, agentID, refID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	res, err = h.agentGetFromDB(ctx, agentID)
	if err != nil {
		t.Fatalf("could not get agent. err: %v", err)
	}
	if res.ReserveReferenceID != uuid.Nil {
		t.Errorf("expected reserve_reference_id cleared, got: %s", res.ReserveReferenceID)
	}
	if res.ReserveReferenceType != "" {
		t.Errorf("expected reserve_reference_type cleared, got: %s", res.ReserveReferenceType)
	}
	if res.TMReserve != nil {
		t.Errorf("expected tm_reserve cleared, got: %v", res.TMReserve)
	}
}

func Test_AgentReserveSweep(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	// zombie agent: reserved long ago
	zombieID := uuid.FromStringOrNil("f0000004-1539-0000-0000-000000000001")
	zombieRef := uuid.FromStringOrNil("f0000004-1539-0000-0000-0000000000a1")
	createAgentForReserve(t, h, mockUtil, mockCache, zombieID, agent.StatusAvailable)

	// fresh agent: reserved just now (not a zombie)
	freshID := uuid.FromStringOrNil("f0000004-1539-0000-0000-000000000002")
	freshRef := uuid.FromStringOrNil("f0000004-1539-0000-0000-0000000000a2")
	createAgentForReserve(t, h, mockUtil, mockCache, freshID, agent.StatusAvailable)

	if ok, err := h.AgentReserve(ctx, zombieID, "queuecall", zombieRef); err != nil || !ok {
		t.Fatalf("could not reserve zombie. ok: %v, err: %v", ok, err)
	}
	if ok, err := h.AgentReserve(ctx, freshID, "queuecall", freshRef); err != nil || !ok {
		t.Fatalf("could not reserve fresh. ok: %v, err: %v", ok, err)
	}

	// tm_reserve was stamped at 2020-04-18 (testTime); sweeping with a
	// "before" of 2020-04-19 must reclaim both. Use a boundary that only
	// catches the stamped time to prove the predicate: sweep before
	// 2020-04-18T03:22:18 (both stamped at 17.995) reclaims both; instead
	// use a before that reclaims neither to prove the < predicate.
	before := time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC) // strictly before both stamps
	count, err := h.AgentReserveSweep(ctx, before)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 swept (both reserved after the before boundary), got: %d", count)
	}

	// now sweep with a boundary after both stamps -> reclaims both
	after := time.Date(2020, 4, 18, 3, 22, 18, 0, time.UTC)
	count, err = h.AgentReserveSweep(ctx, after)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 swept, got: %d", count)
	}

	// verify reserve cleared on both
	for _, id := range []uuid.UUID{zombieID, freshID} {
		res, err := h.agentGetFromDB(ctx, id)
		if err != nil {
			t.Fatalf("could not get agent. err: %v", err)
		}
		if res.ReserveReferenceID != uuid.Nil {
			t.Errorf("expected reserve cleared for %s, got: %s", id, res.ReserveReferenceID)
		}
	}
}
