package dbhandler

import (
	"context"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/models/schedule"
	"monorepo/bin-scheduler-manager/pkg/cachehandler"
)

// newExecutionTestHandler returns a handler with the real utilhandler and a
// permissive mock cache (claim paths refresh the schedule cache after commit).
func newExecutionTestHandler(t *testing.T) (*handler, *gomock.Controller) {
	mc := gomock.NewController(t)

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().ScheduleSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().ScheduleGet(gomock.Any(), gomock.Any()).Return(nil, ErrNotFound).AnyTimes()
	mockCache.EXPECT().ScheduleDelete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          dbTest,
		cache:       mockCache,
	}

	return h, mc
}

// executionTestScheduleCreate creates a schedule for execution tests.
func executionTestScheduleCreate(t *testing.T, h *handler, id uuid.UUID, name string, enabled bool, nextRun *time.Time, timeoutMS int, retryMax int) *schedule.Schedule {
	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
		},
		Name:      name,
		Type:      schedule.TypeRPC,
		Cron:      "0 1 * * *",
		TimeoutMS: timeoutMS,
		RetryMax:  retryMax,
		Enabled:   enabled,
		TMNextRun: nextRun,
	}

	if err := h.ScheduleCreate(context.Background(), s); err != nil {
		t.Fatalf("could not create test schedule. err: %v", err)
	}

	return s
}

func Test_ScheduleClaimAndCreateExecution_Cron(t *testing.T) {
	h, mc := newExecutionTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	slot := time.Date(2026, 7, 10, 1, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 10, 1, 0, 5, 0, time.UTC)
	next := time.Date(2026, 7, 11, 1, 0, 0, 0, time.UTC)

	s := executionTestScheduleCreate(t, h,
		uuid.FromStringOrNil("11a0b101-6f12-11f0-b001-0242ac110002"),
		"claim-cron", true, &slot, 1000, 0)

	res, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeCron, slot)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: execution, got: nil")
	}

	if res.ScheduleID != s.ID {
		t.Errorf("Wrong match. expect: %v, got: %v", s.ID, res.ScheduleID)
	}
	if res.TriggerType != execution.TriggerTypeCron {
		t.Errorf("Wrong match. expect: %v, got: %v", execution.TriggerTypeCron, res.TriggerType)
	}
	if res.Status != execution.StatusRunning {
		t.Errorf("Wrong match. expect: %v, got: %v", execution.StatusRunning, res.Status)
	}
	if res.TMScheduled == nil || !res.TMScheduled.Equal(slot) {
		t.Errorf("Wrong match. expect: %v, got: %v", slot, res.TMScheduled)
	}

	// tm_deadline = now + timeout_ms*(retry_max+1) + 5000*retry_max + 60000 ms
	expectDeadline := now.Add(time.Duration(1000*(0+1)+5000*0+60000) * time.Millisecond)
	if res.TMDeadline == nil || !res.TMDeadline.Equal(expectDeadline) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectDeadline, res.TMDeadline)
	}

	// schedule row: tm_next_run advanced, tm_last_run set
	resSchedule, err := h.scheduleGetFromDB(ctx, s.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resSchedule.TMNextRun == nil || !resSchedule.TMNextRun.Equal(next) {
		t.Errorf("Wrong match. expect: %v, got: %v", next, resSchedule.TMNextRun)
	}
	if resSchedule.TMLastRun == nil || !resSchedule.TMLastRun.Equal(now) {
		t.Errorf("Wrong match. expect: %v, got: %v", now, resSchedule.TMLastRun)
	}

	// execution row exists in the DB
	resExecution, err := h.ExecutionGet(ctx, res.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resExecution.Status != execution.StatusRunning {
		t.Errorf("Wrong match. expect: %v, got: %v", execution.StatusRunning, resExecution.Status)
	}

	// lost CAS: tm_next_run is no longer the stale slot
	resLost, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeCron, slot)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resLost != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", resLost)
	}
}

func Test_ScheduleClaimAndCreateExecution_CronDisabled(t *testing.T) {
	h, mc := newExecutionTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	slot := time.Date(2026, 7, 10, 2, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 10, 2, 0, 5, 0, time.UTC)
	next := time.Date(2026, 7, 11, 2, 0, 0, 0, time.UTC)

	s := executionTestScheduleCreate(t, h,
		uuid.FromStringOrNil("11a0b202-6f12-11f0-b002-0242ac110002"),
		"claim-cron-disabled", false, &slot, 1000, 0)

	// a just-disabled schedule must not fire one last time
	res, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeCron, slot)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_ScheduleClaimAndCreateExecution_Manual(t *testing.T) {
	h, mc := newExecutionTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	slot := time.Date(2026, 7, 10, 3, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 10, 3, 30, 0, 0, time.UTC)
	next := time.Date(2026, 7, 11, 3, 0, 0, 0, time.UTC)

	// manual fire works on a disabled schedule too (design §6)
	s := executionTestScheduleCreate(t, h,
		uuid.FromStringOrNil("11a0b303-6f12-11f0-b003-0242ac110002"),
		"claim-manual", false, &slot, 1000, 0)

	res, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeManual, now)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: execution, got: nil")
	}
	if res.TriggerType != execution.TriggerTypeManual {
		t.Errorf("Wrong match. expect: %v, got: %v", execution.TriggerTypeManual, res.TriggerType)
	}

	// manual run must never touch tm_next_run/tm_last_run
	resSchedule, err := h.scheduleGetFromDB(ctx, s.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resSchedule.TMNextRun == nil || !resSchedule.TMNextRun.Equal(slot) {
		t.Errorf("Manual run must not touch tm_next_run. expect: %v, got: %v", slot, resSchedule.TMNextRun)
	}
	if resSchedule.TMLastRun != nil {
		t.Errorf("Manual run must not touch tm_last_run. got: %v", resSchedule.TMLastRun)
	}

	// duplicate (schedule_id, trigger_type, tm_scheduled) → unique-key backstop → nil, nil
	resDup, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeManual, now)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resDup != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", resDup)
	}
}

func Test_ExecutionComplete(t *testing.T) {
	h, mc := newExecutionTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	slot := time.Date(2026, 7, 10, 4, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 10, 4, 0, 5, 0, time.UTC)
	next := time.Date(2026, 7, 11, 4, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 7, 10, 4, 0, 9, 0, time.UTC)

	s := executionTestScheduleCreate(t, h,
		uuid.FromStringOrNil("11a0b404-6f12-11f0-b004-0242ac110002"),
		"complete", true, &slot, 1000, 0)

	res, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeCron, slot)
	if err != nil || res == nil {
		t.Fatalf("Wrong match. expect: execution, got: %v, err: %v", res, err)
	}

	// running → success
	ok, err := h.ExecutionComplete(ctx, res.ID, execution.StatusSuccess, 200, "", `{"renewed":3}`, 1, 4000, endTime)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !ok {
		t.Errorf("Wrong match. expect: true, got: false")
	}

	resExecution, err := h.ExecutionGet(ctx, res.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resExecution.Status != execution.StatusSuccess {
		t.Errorf("Wrong match. expect: %v, got: %v", execution.StatusSuccess, resExecution.Status)
	}
	if resExecution.StatusCode != 200 {
		t.Errorf("Wrong match. expect: 200, got: %d", resExecution.StatusCode)
	}
	if resExecution.Result != `{"renewed":3}` {
		t.Errorf("Wrong match. expect: %v, got: %v", `{"renewed":3}`, resExecution.Result)
	}
	if resExecution.AttemptCount != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", resExecution.AttemptCount)
	}
	if resExecution.DurationMS != 4000 {
		t.Errorf("Wrong match. expect: 4000, got: %d", resExecution.DurationMS)
	}
	if resExecution.TMEnd == nil || !resExecution.TMEnd.Equal(endTime) {
		t.Errorf("Wrong match. expect: %v, got: %v", endTime, resExecution.TMEnd)
	}

	// second completion must be a no-op (status is no longer running)
	ok, err = h.ExecutionComplete(ctx, res.ID, execution.StatusFailed, 500, "late failure", "", 1, 5000, endTime)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if ok {
		t.Errorf("Wrong match. expect: false, got: true")
	}

	resExecution, err = h.ExecutionGet(ctx, res.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resExecution.Status != execution.StatusSuccess {
		t.Errorf("Completed execution must keep its terminal state. got: %v", resExecution.Status)
	}
}

func Test_ExecutionGet_NotFound(t *testing.T) {
	h, mc := newExecutionTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	if _, err := h.ExecutionGet(ctx, uuid.FromStringOrNil("11a0b505-6f12-11f0-b005-0242ac110002")); err != ErrNotFound {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}
}

func Test_ExecutionList(t *testing.T) {
	h, mc := newExecutionTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	slot := time.Date(2026, 7, 10, 5, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 10, 5, 0, 5, 0, time.UTC)
	next := time.Date(2026, 7, 11, 5, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 7, 10, 5, 0, 9, 0, time.UTC)

	s := executionTestScheduleCreate(t, h,
		uuid.FromStringOrNil("11a0b606-6f12-11f0-b006-0242ac110002"),
		"list", true, &slot, 1000, 0)

	res, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeCron, slot)
	if err != nil || res == nil {
		t.Fatalf("Wrong match. expect: execution, got: %v, err: %v", res, err)
	}
	if _, err := h.ExecutionComplete(ctx, res.ID, execution.StatusFailed, 500, "boom", "", 1, 1000, endTime); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// filter by schedule_id
	filters := map[execution.Field]any{
		execution.FieldScheduleID: s.ID,
	}
	resList, err := h.ExecutionList(ctx, 10, utilhandler.TimeGetCurTime(), filters)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(resList) != 1 {
		t.Fatalf("Wrong match. expect: 1, got: %d", len(resList))
	}
	if resList[0].ID != res.ID {
		t.Errorf("Wrong match. expect: %v, got: %v", res.ID, resList[0].ID)
	}

	// filter by schedule_id + status: no running rows remain
	filtersRunning := map[execution.Field]any{
		execution.FieldScheduleID: s.ID,
		execution.FieldStatus:     execution.StatusRunning,
	}
	resRunning, err := h.ExecutionList(ctx, 10, utilhandler.TimeGetCurTime(), filtersRunning)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resRunning == nil {
		t.Errorf("Expected non-nil empty slice, got nil")
	}
	if len(resRunning) != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", len(resRunning))
	}
}

func Test_ExecutionPrune(t *testing.T) {
	h, mc := newExecutionTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	// prune-test rows live in their own ancient epoch (2020) so the cutoff
	// (2021) cannot touch any other test's rows (all 2026).
	slot := time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC)
	next := time.Date(2020, 1, 2, 1, 0, 0, 0, time.UTC)
	cutoff := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	s := executionTestScheduleCreate(t, h,
		uuid.FromStringOrNil("11a0b707-6f12-11f0-b007-0242ac110002"),
		"prune", false, nil, 1000, 0)

	// three manual executions with distinct tm_scheduled, all tm_create = 2020
	for i := 0; i < 3; i++ {
		tmScheduled := slot.Add(time.Duration(i) * time.Minute)
		res, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, tmScheduled, execution.TriggerTypeManual, tmScheduled)
		if err != nil || res == nil {
			t.Fatalf("Wrong match. expect: execution, got: %v, err: %v", res, err)
		}
	}

	// batch 2: only two rows deleted
	deleted, err := h.ExecutionPrune(ctx, cutoff, 2)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if deleted != 2 {
		t.Errorf("Wrong match. expect: 2, got: %d", deleted)
	}

	// next batch: the remaining row
	deleted, err = h.ExecutionPrune(ctx, cutoff, 2)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", deleted)
	}

	// nothing left under the cutoff
	deleted, err = h.ExecutionPrune(ctx, cutoff, 2)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", deleted)
	}
}

func Test_ExecutionCountAll(t *testing.T) {
	h, mc := newExecutionTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	before, err := h.ExecutionCountAll(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	slot := time.Date(2026, 7, 10, 6, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 10, 6, 0, 5, 0, time.UTC)
	next := time.Date(2026, 7, 11, 6, 0, 0, 0, time.UTC)

	s := executionTestScheduleCreate(t, h,
		uuid.FromStringOrNil("11a0b808-6f12-11f0-b008-0242ac110002"),
		"count", true, &slot, 1000, 0)

	res, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeCron, slot)
	if err != nil || res == nil {
		t.Fatalf("Wrong match. expect: execution, got: %v, err: %v", res, err)
	}
	// finalize so this row cannot wedge later reap assertions
	if _, err := h.ExecutionComplete(ctx, res.ID, execution.StatusSuccess, 200, "", "", 1, 100, now); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	after, err := h.ExecutionCountAll(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if after != before+1 {
		t.Errorf("Wrong match. expect: %d, got: %d", before+1, after)
	}
}

func Test_ExecutionReapAbandoned(t *testing.T) {
	h, mc := newExecutionTestHandler(t)
	defer mc.Finish()
	ctx := context.Background()

	// reap-test rows live in their own epoch (2026-06) so reap timestamps here
	// cannot touch other tests' running rows (their deadlines are 2026-07+).
	slot := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 1, 0, 0, 5, 0, time.UTC)
	next := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)

	// timeout 1000ms, retry 0 → deadline = now + 1s + 60s
	s := executionTestScheduleCreate(t, h,
		uuid.FromStringOrNil("11a0b909-6f12-11f0-b009-0242ac110002"),
		"reap", true, &slot, 1000, 0)

	res, err := h.ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeCron, slot)
	if err != nil || res == nil {
		t.Fatalf("Wrong match. expect: execution, got: %v, err: %v", res, err)
	}

	// before the deadline: nothing reaped
	reaped, err := h.ExecutionReapAbandoned(ctx, now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if reaped != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", reaped)
	}

	// after the deadline: exactly this row reaped
	reaped, err = h.ExecutionReapAbandoned(ctx, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if reaped != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", reaped)
	}

	resExecution, err := h.ExecutionGet(ctx, res.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resExecution.Status != execution.StatusAbandoned {
		t.Errorf("Wrong match. expect: %v, got: %v", execution.StatusAbandoned, resExecution.Status)
	}
	if resExecution.Error != "replica died mid-dispatch" {
		t.Errorf("Wrong match. expect: replica died mid-dispatch, got: %v", resExecution.Error)
	}

	// idempotent: a second reap finds nothing
	reaped, err = h.ExecutionReapAbandoned(ctx, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if reaped != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", reaped)
	}
}

// Test_DoubleFire is the design §10 acceptance test: N concurrent claimants
// (spread over two dbhandler "replica" instances sharing one DB, simulating a
// Redis split-brain where every replica's TryLock succeeded) race for the SAME
// slot — exactly one may win and exactly one execution row may exist.
//
// Caveat: sqlite serializes writers (SetMaxOpenConns(1)), so this proves the
// SQL-level invariant, not MySQL REPEATABLE READ interleavings — the CAS is a
// single conditional UPDATE whose row-lock semantics under MySQL make the same
// guarantee (argued, not executed, in CI; design §10).
func Test_DoubleFire(t *testing.T) {
	h1, mc1 := newExecutionTestHandler(t)
	defer mc1.Finish()
	h2, mc2 := newExecutionTestHandler(t)
	defer mc2.Finish()
	ctx := context.Background()

	slot := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 20, 1, 0, 5, 0, time.UTC)
	next := time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC)

	s := executionTestScheduleCreate(t, h1,
		uuid.FromStringOrNil("11a0ba0a-6f12-11f0-b00a-0242ac110002"),
		"double-fire", true, &slot, 1000, 0)

	const n = 20
	handlers := []*handler{h1, h2}

	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := 0
	losers := 0
	var errs []error

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			res, err := handlers[idx%2].ScheduleClaimAndCreateExecution(ctx, s, next, now, execution.TriggerTypeCron, slot)

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				errs = append(errs, err)
			case res != nil:
				winners++
			default:
				losers++
			}
		}(i)
	}
	wg.Wait()

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}
	if winners != 1 {
		t.Errorf("Wrong match. expect: exactly 1 winner, got: %d", winners)
	}
	if losers != n-1 {
		t.Errorf("Wrong match. expect: %d losers, got: %d", n-1, losers)
	}

	// exactly one execution row exists for the schedule
	filters := map[execution.Field]any{
		execution.FieldScheduleID: s.ID,
	}
	resList, err := h1.ExecutionList(ctx, 100, utilhandler.TimeGetCurTime(), filters)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(resList) != 1 {
		t.Errorf("Wrong match. expect: exactly 1 execution row, got: %d", len(resList))
	}

	// finalize so this row cannot wedge later reap assertions
	if _, err := h1.ExecutionComplete(ctx, resList[0].ID, execution.StatusSuccess, 200, "", "", 1, 100, now); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_KillNineStateMachine is the design §10 kill -9 acceptance test at the
// state-machine level: replica A claims and inserts a running execution, then
// vanishes without a completion write; replica B's overlap guard skips while
// the row is within budget; after the budget elapses B's reap marks it
// abandoned and the next due slot claims and fires normally.
func Test_KillNineStateMachine(t *testing.T) {
	hA, mcA := newExecutionTestHandler(t)
	defer mcA.Finish()
	hB, mcB := newExecutionTestHandler(t)
	defer mcB.Finish()
	ctx := context.Background()

	// kill-9 rows live in their own epoch (2026-05) so reap timestamps here
	// cannot touch other tests' running rows.
	slot1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	now1 := time.Date(2026, 5, 1, 0, 0, 5, 0, time.UTC)
	slot2 := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)

	// timeout 1000ms, retry 0 → deadline = now1 + 61s
	s := executionTestScheduleCreate(t, hA,
		uuid.FromStringOrNil("11a0bb0b-6f12-11f0-b00b-0242ac110002"),
		"kill-nine", true, &slot1, 1000, 0)

	// (a) replica A claims and inserts a running execution, then "dies" —
	// no completion write ever happens.
	resA, err := hA.ScheduleClaimAndCreateExecution(ctx, s, slot2, now1, execution.TriggerTypeCron, slot1)
	if err != nil || resA == nil {
		t.Fatalf("Wrong match. expect: execution, got: %v, err: %v", resA, err)
	}

	// (b) replica B's overlap guard sees the running row and skips
	hasRunning, err := hB.ExecutionHasRunning(ctx, s.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !hasRunning {
		t.Errorf("Wrong match. expect: true, got: false")
	}

	// within budget: reap must not touch the row
	reaped, err := hB.ExecutionReapAbandoned(ctx, now1.Add(30*time.Second))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if reaped != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", reaped)
	}

	// (c) after the budget elapses: B's reap marks it abandoned
	reaped, err = hB.ExecutionReapAbandoned(ctx, now1.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if reaped != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", reaped)
	}

	hasRunning, err = hB.ExecutionHasRunning(ctx, s.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if hasRunning {
		t.Errorf("Wrong match. expect: false, got: true")
	}

	resAbandoned, err := hB.ExecutionGet(ctx, resA.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resAbandoned.Status != execution.StatusAbandoned {
		t.Errorf("Wrong match. expect: %v, got: %v", execution.StatusAbandoned, resAbandoned.Status)
	}

	// the next due slot claims and fires normally
	now2 := time.Date(2026, 5, 2, 0, 0, 5, 0, time.UTC)
	slot3 := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	resB, err := hB.ScheduleClaimAndCreateExecution(ctx, s, slot3, now2, execution.TriggerTypeCron, slot2)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resB == nil {
		t.Fatalf("Wrong match. expect: execution, got: nil")
	}

	// finalize so this row cannot wedge later reap assertions
	if _, err := hB.ExecutionComplete(ctx, resB.ID, execution.StatusSuccess, 200, "", "", 1, 100, now2); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}
