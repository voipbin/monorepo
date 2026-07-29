package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// insertInsightAI seeds an ai_ais row of the given type/active state via raw SQL.
func insertInsightAI(t *testing.T, id, customerID uuid.UUID, aiType ai.Type, active bool, tmCreate time.Time) {
	t.Helper()

	activeVal := 0
	if active {
		activeVal = 1
	}
	if _, err := dbTest.Exec(
		`INSERT INTO ai_ais (id, customer_id, type, is_insight_active, tm_create) VALUES (?, ?, ?, ?, ?)`,
		id.Bytes(), customerID.Bytes(), string(aiType), activeVal,
		tmCreate.UTC().Format("2006-01-02 15:04:05.000000"),
	); err != nil {
		t.Fatalf("insertInsightAI: %v", err)
	}
}

// fetchAIIsInsightActive reads is_insight_active straight from the DB, bypassing the cache.
func fetchAIIsInsightActive(t *testing.T, id uuid.UUID) bool {
	t.Helper()

	var active bool
	if err := dbTest.QueryRow(`SELECT is_insight_active FROM ai_ais WHERE id = ?`, id.Bytes()).Scan(&active); err != nil {
		t.Fatalf("fetchAIIsInsightActive: %v", err)
	}
	return active
}

// activeAIMatcher asserts the AI written to the cache carries the expected
// is_insight_active state for the expected id.
type activeAIMatcher struct {
	expectedID     uuid.UUID
	expectedActive bool
}

func (m activeAIMatcher) Matches(x any) bool {
	a, ok := x.(*ai.AI)
	if !ok {
		return false
	}
	return a.ID == m.expectedID && a.IsInsightActive == m.expectedActive
}

func (m activeAIMatcher) String() string {
	return "*ai.AI matching the expected id and is_insight_active state"
}

func newActivateTestHandler(t *testing.T, mc *gomock.Controller) (handler, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	return handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}, mockUtil, mockCache
}

// --- paths that return before the FOR UPDATE statement (runnable on SQLite) ---

func Test_AIActivateInsight_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _ := newActivateTestHandler(t, mc)

	// Negative control: no cache write may happen for a missing target, so no
	// mockCache expectation is set — mc.Finish() fails on an unexpected AISet.
	_, _, err := h.AIActivateInsight(t.Context(), uuid.FromStringOrNil("b1b1b1b1-0000-0000-0000-0000000000ff"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func Test_AIActivateInsight_SoftDeletedTarget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _ := newActivateTestHandler(t, mc)

	customerID := uuid.FromStringOrNil("b1b1b1b1-0000-0000-0000-000000000001")
	aiID := uuid.FromStringOrNil("b1b1b1b1-0000-0000-0000-000000000002")
	insertInsightAI(t, aiID, customerID, ai.TypeInsight, false, time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))

	if _, err := dbTest.Exec(`UPDATE ai_ais SET tm_delete = ? WHERE id = ?`,
		"2026-07-29 01:00:00.000000", aiID.Bytes()); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	_, _, err := h.AIActivateInsight(t.Context(), aiID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a soft-deleted target, got %v", err)
	}
	if got := fetchAIIsInsightActive(t, aiID); got {
		t.Errorf("soft-deleted ai: is_insight_active = true, want false")
	}
}

func Test_AIActivateInsight_NormalTypeRejected(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _ := newActivateTestHandler(t, mc)

	customerID := uuid.FromStringOrNil("b1b1b1b1-0000-0000-0000-000000000003")
	aiID := uuid.FromStringOrNil("b1b1b1b1-0000-0000-0000-000000000004")
	insertInsightAI(t, aiID, customerID, ai.TypeNormal, false, time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))

	_, _, err := h.AIActivateInsight(t.Context(), aiID)
	if !errors.Is(err, ErrNotInsightAI) {
		t.Fatalf("expected ErrNotInsightAI for a type=normal target, got %v", err)
	}
	if got := fetchAIIsInsightActive(t, aiID); got {
		t.Errorf("type=normal ai: is_insight_active = true, want false")
	}
}

// --- delete / update side effects on is_insight_active (runnable on SQLite) ---

func Test_AIDelete_ClearsIsInsightActive(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockCache := newActivateTestHandler(t, mc)
	mockCache.EXPECT().AISet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	customerID := uuid.FromStringOrNil("b2b2b2b2-0000-0000-0000-000000000001")
	aiID := uuid.FromStringOrNil("b2b2b2b2-0000-0000-0000-000000000002")
	insertInsightAI(t, aiID, customerID, ai.TypeInsight, true, time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))

	ts := time.Date(2026, 7, 29, 2, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&ts)

	if err := h.AIDelete(t.Context(), aiID); err != nil {
		t.Fatalf("AIDelete: %v", err)
	}

	if got := fetchAIIsInsightActive(t, aiID); got {
		t.Errorf("after AIDelete: is_insight_active = true, want false")
	}
}

func Test_AIUpdate_ClearsIsInsightActive(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockCache := newActivateTestHandler(t, mc)
	mockCache.EXPECT().AISet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	customerID := uuid.FromStringOrNil("b2b2b2b2-0000-0000-0000-000000000003")
	aiID := uuid.FromStringOrNil("b2b2b2b2-0000-0000-0000-000000000004")
	insertInsightAI(t, aiID, customerID, ai.TypeInsight, true, time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))

	ts := time.Date(2026, 7, 29, 3, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&ts)

	// This is the field map aihandler.buildUpdateFields produces for a
	// type-change away from insight.
	if err := h.AIUpdate(t.Context(), aiID, map[ai.Field]any{
		ai.FieldType:            ai.TypeNormal,
		ai.FieldIsInsightActive: false,
	}); err != nil {
		t.Fatalf("AIUpdate: %v", err)
	}

	if got := fetchAIIsInsightActive(t, aiID); got {
		t.Errorf("after type change to normal: is_insight_active = true, want false")
	}
}

func Test_AIGet_RoundTripsIsInsightActive(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockCache := newActivateTestHandler(t, mc)

	customerID := uuid.FromStringOrNil("b2b2b2b2-0000-0000-0000-000000000005")
	aiID := uuid.FromStringOrNil("b2b2b2b2-0000-0000-0000-000000000006")
	insertInsightAI(t, aiID, customerID, ai.TypeInsight, true, time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))

	mockCache.EXPECT().AIGet(gomock.Any(), aiID).Return(nil, errors.New("cache miss"))
	mockCache.EXPECT().AISet(gomock.Any(), activeAIMatcher{expectedID: aiID, expectedActive: true}).Return(nil)

	res, err := h.AIGet(t.Context(), aiID)
	if err != nil {
		t.Fatalf("AIGet: %v", err)
	}
	if !res.IsInsightActive {
		t.Errorf("AIGet: IsInsightActive = false, want true")
	}

	// The identity round-trip guards against a column-order regression in the
	// generated select list.
	if res.CustomerID != customerID {
		t.Errorf("AIGet: customer_id = %v, want %v", res.CustomerID, customerID)
	}
}

// --- paths that reach the FOR UPDATE statement ---
//
// KNOWN, ACCEPTED COVERAGE GAP -- NOT AN OVERSIGHT.
//
// AIActivateInsight locks the customer's currently-active insight row with
// `SELECT … FOR UPDATE`, which the in-memory SQLite test driver (main_test.go)
// does not parse. That means EVERY test below is skipped, not just the
// concurrency one: the swap, the idempotent re-activation, and the
// cross-customer isolation case all execute the same locking statement and
// would fail on a driver error rather than on a real behavioral difference.
//
// This is a deliberate deviation from the design's test plan and cannot be
// worked around at this layer -- rewriting the query to avoid FOR UPDATE would
// drop the row lock the single-active-insight invariant depends on, and
// stubbing it out would make the tests assert something other than the shipped
// SQL. Closing the gap requires MySQL/MariaDB-backed test infrastructure; it is
// tracked as a follow-up. Until then these tests are kept as executable
// documentation of the intended behavior and will run unmodified once that
// infra exists. Same reasoning, and same precedent, as the skipped tests in
// aipromptproposal_test.go.

func Test_AIActivateInsight_SwapsActiveAndRefreshesBothCaches(t *testing.T) {
	t.Skip(skipReasonNoForUpdate)

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockCache := newActivateTestHandler(t, mc)

	customerID := uuid.FromStringOrNil("b3b3b3b3-0000-0000-0000-000000000001")
	firstID := uuid.FromStringOrNil("b3b3b3b3-0000-0000-0000-000000000002")
	secondID := uuid.FromStringOrNil("b3b3b3b3-0000-0000-0000-000000000003")

	insertInsightAI(t, firstID, customerID, ai.TypeInsight, true, time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))
	insertInsightAI(t, secondID, customerID, ai.TypeInsight, false, time.Date(2026, 7, 29, 0, 0, 1, 0, time.UTC))

	ts := time.Date(2026, 7, 29, 4, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&ts)

	// BOTH rows must be written back to the cache: the deactivated one and the
	// newly activated one. Skipping either leaves the Case panel resolving a
	// stale AI indefinitely (AIGet is cache-first).
	mockCache.EXPECT().AISet(gomock.Any(), activeAIMatcher{expectedID: firstID, expectedActive: false}).Return(nil).Times(1)
	mockCache.EXPECT().AISet(gomock.Any(), activeAIMatcher{expectedID: secondID, expectedActive: true}).Return(nil).Times(1)

	res, previous, err := h.AIActivateInsight(t.Context(), secondID)
	if err != nil {
		t.Fatalf("AIActivateInsight: %v", err)
	}
	if !res.IsInsightActive {
		t.Errorf("returned ai: IsInsightActive = false, want true")
	}

	// The deactivated row must come back to the caller so aihandler can publish
	// its ai_updated webhook.
	if previous == nil {
		t.Fatalf("previous ai = nil, want the deactivated ai")
	}
	if previous.ID != firstID {
		t.Errorf("previous ai: id = %v, want %v", previous.ID, firstID)
	}
	if previous.IsInsightActive {
		t.Errorf("previous ai: IsInsightActive = true, want false")
	}

	if got := fetchAIIsInsightActive(t, firstID); got {
		t.Errorf("previously active ai: is_insight_active = true, want false")
	}
	if got := fetchAIIsInsightActive(t, secondID); !got {
		t.Errorf("newly activated ai: is_insight_active = false, want true")
	}
}

func Test_AIActivateInsight_Idempotent(t *testing.T) {
	t.Skip(skipReasonNoForUpdate)

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockCache := newActivateTestHandler(t, mc)

	customerID := uuid.FromStringOrNil("b4b4b4b4-0000-0000-0000-000000000001")
	aiID := uuid.FromStringOrNil("b4b4b4b4-0000-0000-0000-000000000002")
	insertInsightAI(t, aiID, customerID, ai.TypeInsight, true, time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))

	ts := time.Date(2026, 7, 29, 5, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&ts)
	mockCache.EXPECT().AISet(gomock.Any(), activeAIMatcher{expectedID: aiID, expectedActive: true}).Return(nil).Times(1)

	res, previous, err := h.AIActivateInsight(t.Context(), aiID)
	if err != nil {
		t.Fatalf("re-activating the already-active ai must succeed, got: %v", err)
	}
	if !res.IsInsightActive {
		t.Errorf("returned ai: IsInsightActive = false, want true")
	}
	// Nothing was displaced, so there is no previous ai to report -- a non-nil
	// value here would make aihandler publish a spurious deactivation event.
	if previous != nil {
		t.Errorf("previous ai = %v, want nil for an idempotent re-activation", previous.ID)
	}
	// And no row was written: tm_update must still be NULL from the seed insert.
	var tmUpdate any
	if err := dbTest.QueryRow(`SELECT tm_update FROM ai_ais WHERE id = ?`, aiID.Bytes()).Scan(&tmUpdate); err != nil {
		t.Fatalf("select tm_update: %v", err)
	}
	if tmUpdate != nil {
		t.Errorf("idempotent re-activation bumped tm_update to %v, want no write at all", tmUpdate)
	}
	if got := fetchAIIsInsightActive(t, aiID); !got {
		t.Errorf("after idempotent re-activation: is_insight_active = false, want true")
	}
}

func Test_AIActivateInsight_DoesNotTouchOtherCustomers(t *testing.T) {
	t.Skip(skipReasonNoForUpdate)

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockCache := newActivateTestHandler(t, mc)

	// Cross-customer guard. Authz lives in bin-api-manager, but the FOR UPDATE
	// scope here is derived from the TARGET row's own customer_id — so even if
	// that check regressed, activating customer B's AI must never clear
	// customer A's active AI.
	customerA := uuid.FromStringOrNil("b5b5b5b5-0000-0000-0000-00000000000a")
	customerB := uuid.FromStringOrNil("b5b5b5b5-0000-0000-0000-00000000000b")
	aiA := uuid.FromStringOrNil("b5b5b5b5-0000-0000-0000-000000000001")
	aiB := uuid.FromStringOrNil("b5b5b5b5-0000-0000-0000-000000000002")

	insertInsightAI(t, aiA, customerA, ai.TypeInsight, true, time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))
	insertInsightAI(t, aiB, customerB, ai.TypeInsight, false, time.Date(2026, 7, 29, 0, 0, 1, 0, time.UTC))

	ts := time.Date(2026, 7, 29, 6, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&ts)
	mockCache.EXPECT().AISet(gomock.Any(), activeAIMatcher{expectedID: aiB, expectedActive: true}).Return(nil).Times(1)

	if _, _, err := h.AIActivateInsight(t.Context(), aiB); err != nil {
		t.Fatalf("AIActivateInsight: %v", err)
	}

	if got := fetchAIIsInsightActive(t, aiA); !got {
		t.Errorf("customer A's active ai was cleared by an activation in customer B")
	}
	if got := fetchAIIsInsightActive(t, aiB); !got {
		t.Errorf("customer B's ai: is_insight_active = false, want true")
	}
}

func Test_AIActivateInsight_ConcurrentActivationsSerialize(t *testing.T) {
	t.Skip(skipReasonNoForUpdate)

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockCache := newActivateTestHandler(t, mc)
	mockUtil.EXPECT().TimeNow().Return(&[]time.Time{time.Date(2026, 7, 29, 7, 0, 0, 0, time.UTC)}[0]).AnyTimes()
	mockCache.EXPECT().AISet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	customerID := uuid.FromStringOrNil("b6b6b6b6-0000-0000-0000-000000000001")
	firstID := uuid.FromStringOrNil("b6b6b6b6-0000-0000-0000-000000000002")
	secondID := uuid.FromStringOrNil("b6b6b6b6-0000-0000-0000-000000000003")

	insertInsightAI(t, firstID, customerID, ai.TypeInsight, false, time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))
	insertInsightAI(t, secondID, customerID, ai.TypeInsight, false, time.Date(2026, 7, 29, 0, 0, 1, 0, time.UTC))

	ctx := context.Background()
	errs := make(chan error, 2)
	go func() { _, _, e := h.AIActivateInsight(ctx, firstID); errs <- e }()
	go func() { _, _, e := h.AIActivateInsight(ctx, secondID); errs <- e }()

	var failures int
	for i := 0; i < 2; i++ {
		if e := <-errs; e != nil {
			// A loser either blocks on the row lock and then succeeds, or loses
			// the unique index and surfaces as a duplicate-key error. Both are
			// acceptable; anything else is not.
			if !IsErrDuplicate(e) {
				t.Errorf("unexpected error from concurrent activation: %v", e)
			}
			failures++
		}
	}

	// Whatever happened, the invariant must hold: exactly one active row.
	activeCount := 0
	for _, id := range []uuid.UUID{firstID, secondID} {
		if fetchAIIsInsightActive(t, id) {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Errorf("active insight ai count = %d, want exactly 1", activeCount)
	}
}
