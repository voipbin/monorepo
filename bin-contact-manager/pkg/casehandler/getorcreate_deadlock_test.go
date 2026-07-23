package casehandler

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_GetOrCreate_Deadlock_RetriesWithFreshBeginTx verifies VOIP-1232's
// core fix: when CaseInsertTx surfaces dbhandler.ErrDeadlock (MySQL errno
// 1213), GetOrCreate discards the dead transaction and restarts the WHOLE
// attempt from a fresh BeginTx (not just re-running the failed
// statement) -- succeeding on the second attempt.
func Test_GetOrCreate_Deadlock_RetriesWithFreshBeginTx(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify, peerLocks: make(map[string]chan struct{})}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7010-7010-7010-000000000001")
	attempt1ID := uuid.FromStringOrNil("f1b2c3d4-7010-7010-7010-000000000002")
	attempt2ID := uuid.FromStringOrNil("f1b2c3d4-7010-7010-7010-000000000003")
	now1 := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	now2 := now1.Add(50 * time.Millisecond)

	// dbTest has SetMaxOpenConns(1), so tx's must be opened and disposed
	// of (Rollback/Commit) sequentially, not both pre-opened upfront --
	// use DoAndReturn to lazily begin each real tx exactly when GetOrCreate
	// itself calls BeginTx, matching production's real one-at-a-time
	// lifecycle.
	var txs []*sql.Tx
	mockDB.EXPECT().BeginTx(ctx).Times(2).DoAndReturn(func(_ context.Context) (*sql.Tx, error) {
		tx, err := dbTest.Begin()
		txs = append(txs, tx)
		return tx, err
	})
	defer func() {
		for _, tx := range txs {
			_ = tx.Rollback()
		}
	}()

	// Attempt 1: own `now`, insert hits a deadlock -- GetOrCreate's own
	// defer rolls this tx back (committed stays false) before attempt 2.
	mockUtil.EXPECT().TimeNow().Return(&now1)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+155****0001", "conversation_message").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+155****0001", "conversation_message").Return(nil, nil)
	mockUtil.EXPECT().UUIDCreate().Return(attempt1ID)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(dbhandler.ErrDeadlock)

	// Attempt 2: a genuinely FRESH BeginTx and a freshly re-captured `now`
	// (not the stale now1) -- succeeds.
	mockUtil.EXPECT().TimeNow().Return(&now2)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+155****0001", "conversation_message").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+155****0001", "conversation_message").Return(nil, nil)
	mockUtil.EXPECT().UUIDCreate().Return(attempt2ID)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().CaseUpdateTMUpdateTx(ctx, gomock.Any(), attempt2ID, &now2).Return(nil)

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"}, "conversation_message", nil, "")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != attempt2ID {
		t.Errorf("expected the second (post-deadlock-retry) attempt's case %s to win, got: %v", attempt2ID, res)
	}
	if len(txs) != 2 || txs[0] == txs[1] {
		t.Errorf("expected two DISTINCT transactions (fresh BeginTx per attempt), got: %d tx(s)", len(txs))
	}
}

// Test_GetOrCreate_Deadlock_ExhaustionReturnsDistinctError verifies
// VOIP-1232's exhaustion path: when every maxDeadlockRetries attempt hits
// a deadlock, GetOrCreate surfaces ErrDeadlockExhausted (distinct from
// ErrGetOrCreateExhausted, which covers the narrower 1062/ErrDuplicate
// retry) so callers can tag it separately.
func Test_GetOrCreate_Deadlock_ExhaustionReturnsDistinctError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify, peerLocks: make(map[string]chan struct{})}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7011-7011-7011-000000000001")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	// dbTest's shared connection has SetMaxOpenConns(1); pre-opening
	// maxDeadlockRetries real tx's upfront (before any is rolled back)
	// would deadlock the connection pool itself. Lazily open one per
	// attempt via DoAndReturn instead, matching each attempt's real
	// begin -> (fail) -> GetOrCreate's own deferred rollback lifecycle.
	var txs []*sql.Tx
	mockDB.EXPECT().BeginTx(ctx).Times(maxDeadlockRetries).DoAndReturn(func(_ context.Context) (*sql.Tx, error) {
		tx, err := dbTest.Begin()
		txs = append(txs, tx)
		return tx, err
	})
	defer func() {
		for _, tx := range txs {
			_ = tx.Rollback()
		}
	}()

	mockUtil.EXPECT().TimeNow().Return(&now).Times(maxDeadlockRetries)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+155****0002", "call").Return(nil, nil).Times(maxDeadlockRetries)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+155****0002", "call").Return(nil, nil).Times(maxDeadlockRetries)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())).Times(maxDeadlockRetries)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(dbhandler.ErrDeadlock).Times(maxDeadlockRetries)

	_, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"}, "call", nil, "")
	if !errors.Is(err, ErrDeadlockExhausted) {
		t.Errorf("expected ErrDeadlockExhausted, got: %v", err)
	}
	if len(txs) != maxDeadlockRetries {
		t.Fatalf("expected exactly %d BeginTx calls, got: %d", maxDeadlockRetries, len(txs))
	}
	// VOIP-1232 round-1 PR review finding: a call-count assertion alone
	// would still pass even if a bug reused the SAME *sql.Tx handle
	// across every exhaustion attempt (instead of a genuinely fresh
	// transaction per retry). Assert pairwise pointer distinctness to
	// actually prove the fresh-BeginTx-per-attempt contract holds even
	// on the give-up path, matching the assertion already present in
	// Test_GetOrCreate_Deadlock_RetriesWithFreshBeginTx above.
	for i := 0; i < len(txs); i++ {
		for j := i + 1; j < len(txs); j++ {
			if txs[i] == txs[j] {
				t.Errorf("expected every deadlock-retry attempt to use a distinct *sql.Tx, but attempt %d and %d shared the same tx handle", i, j)
			}
		}
	}
}

// Test_GetOrCreate_PeerLock_SerializesSameTuple verifies VOIP-1232's
// in-process peer-lock: two goroutines calling GetOrCreate for the
// IDENTICAL (customer_id, peer_type, peer_target, reference_type) tuple
// never run their DB transactions concurrently -- the second call's
// BeginTx is only observed to start after the first call's has fully
// completed (committed).
func Test_GetOrCreate_PeerLock_SerializesSameTuple(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify, peerLocks: make(map[string]chan struct{})}
	ctx := context.Background()

	// Use a dedicated, higher-pool-size in-memory DB (not the shared
	// package-level dbTest, which is intentionally SetMaxOpenConns(1) --
	// that alone would serialize BeginTx regardless of the peer lock,
	// making this test unable to distinguish "serialized by my code" from
	// "serialized by the connection pool"). No schema is needed: every DB
	// operation below goes through the mocked DBHandler; only tx.Commit()/
	// tx.Rollback() run against the real driver.
	concurrentDB, err := sql.Open("sqlite3", "file:voip1232_concurrency_test1?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("could not open concurrency-test db. err: %v", err)
	}
	concurrentDB.SetMaxOpenConns(4)
	defer func() { _ = concurrentDB.Close() }()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7012-7012-7012-000000000001")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	var activeCount int
	var maxObservedConcurrent int
	var mu sync.Mutex

	// gomock's default expectation ordering doesn't itself enforce
	// concurrency limits, so we assert serialization via a counter
	// incremented/decremented around each BeginTx call, using DoAndReturn
	// to observe overlap directly.
	mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
	mockDB.EXPECT().BeginTx(ctx).Times(2).DoAndReturn(func(_ context.Context) (*sql.Tx, error) {
		mu.Lock()
		activeCount++
		if activeCount > maxObservedConcurrent {
			maxObservedConcurrent = activeCount
		}
		mu.Unlock()

		tx, err := concurrentDB.Begin()

		// Simulate non-trivial DB work duration so a broken (non-
		// serializing) implementation would have a real chance to overlap.
		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		activeCount--
		mu.Unlock()

		return tx, err
	})
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+155****0003", "call").Return(nil, nil).Times(2)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+155****0003", "call").Return(nil, nil).Times(2)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())).Times(2)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(nil).Times(2)
	mockDB.EXPECT().CaseUpdateTMUpdateTx(ctx, gomock.Any(), gomock.Any(), &now).Return(nil).Times(2)

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			_, _ = h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0003"}, "call", nil, "")
		}()
	}
	wg.Wait()

	if maxObservedConcurrent > 1 {
		t.Errorf("expected the peer lock to serialize same-tuple GetOrCreate calls (max concurrent BeginTx == 1), observed max concurrent = %d", maxObservedConcurrent)
	}
}

// Test_GetOrCreate_PeerLock_DifferentTuplesProceedConcurrently verifies
// the keyed-lock does NOT over-serialize: two DIFFERENT peer tuples must
// not block each other.
func Test_GetOrCreate_PeerLock_DifferentTuplesProceedConcurrently(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify, peerLocks: make(map[string]chan struct{})}
	ctx := context.Background()

	// See Test_GetOrCreate_PeerLock_SerializesSameTuple's comment: a
	// dedicated higher-pool-size DB, not the shared single-connection
	// dbTest, so the connection pool itself can't be the thing enforcing
	// (or masking a failure to enforce) serialization.
	concurrentDB, err := sql.Open("sqlite3", "file:voip1232_concurrency_test2?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("could not open concurrency-test db. err: %v", err)
	}
	concurrentDB.SetMaxOpenConns(4)
	defer func() { _ = concurrentDB.Close() }()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7013-7013-7013-000000000001")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	var activeCount int32
	var maxObservedConcurrent int32
	var mu sync.Mutex

	mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
	mockDB.EXPECT().BeginTx(ctx).Times(2).DoAndReturn(func(_ context.Context) (*sql.Tx, error) {
		mu.Lock()
		activeCount++
		if activeCount > maxObservedConcurrent {
			maxObservedConcurrent = activeCount
		}
		mu.Unlock()

		tx, err := concurrentDB.Begin()
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		activeCount--
		mu.Unlock()

		return tx, err
	})
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, gomock.Any(), "call").Return(nil, nil).Times(2)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, gomock.Any(), "call").Return(nil, nil).Times(2)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())).Times(2)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(nil).Times(2)
	mockDB.EXPECT().CaseUpdateTMUpdateTx(ctx, gomock.Any(), gomock.Any(), &now).Return(nil).Times(2)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0004"}, "call", nil, "")
	}()
	go func() {
		defer wg.Done()
		_, _ = h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0005"}, "call", nil, "")
	}()
	wg.Wait()

	if maxObservedConcurrent < 2 {
		t.Errorf("expected two DIFFERENT peer tuples to proceed concurrently (max concurrent BeginTx >= 2), observed max concurrent = %d", maxObservedConcurrent)
	}
}

// Test_GetOrCreate_PeerLock_TimeoutReturnsDistinctError verifies that a
// caller unable to acquire the peer lock within peerLockTimeout gets
// ErrPeerLockTimeout, not an indefinite hang. Exercises the real
// (non-mocked) channel semaphore directly via acquirePeerLock -- no DB
// mocks needed for this specific behavior.
func Test_GetOrCreate_PeerLock_TimeoutReturnsDistinctError(t *testing.T) {
	h := &caseHandler{peerLocks: make(map[string]chan struct{})}
	key := "timeout-test-tuple"

	// Hold the lock ourselves so a second acquire attempt must wait.
	release, err := h.acquirePeerLock(context.Background(), key)
	if err != nil {
		t.Fatalf("first acquirePeerLock() error = %v", err)
	}
	defer release()

	// Use a short-lived context to keep the test fast rather than waiting
	// the full peerLockTimeout.
	shortCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = h.acquirePeerLock(shortCtx, key)
	if !errors.Is(err, ErrPeerLockTimeout) {
		t.Errorf("expected ErrPeerLockTimeout, got: %v", err)
	}
}
