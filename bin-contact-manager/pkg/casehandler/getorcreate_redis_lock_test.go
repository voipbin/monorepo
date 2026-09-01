package casehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/go-redsync/redsync/v4"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_GetOrCreate_redisLockHeld_returnsError verifies VOIP-1438's
// fail-closed path end-to-end through the public GetOrCreate entry point:
// when the Redis distributed lock is already held by another replica
// (ErrTaken), GetOrCreate returns an error WITHOUT ever touching the
// database -- mockDB.BeginTx is asserted via .Times(0) rather than
// EXPECT().Return(...), so any DB call at all fails the test.
func Test_GetOrCreate_redisLockHeld_returnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockLocker := NewMocklocker(mc)
	mockMutex := NewMockredsyncMutex(mc)

	mockLocker.EXPECT().NewMutex(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockMutex)
	mockMutex.EXPECT().LockContext(gomock.Any()).Return(&redsync.ErrTaken{Nodes: []int{0}})

	mockDB.EXPECT().BeginTx(gomock.Any()).Times(0)

	h := &caseHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
		peerLocks:     make(map[string]chan struct{}),
		redisLocker:   mockLocker,
	}

	customerID := uuid.FromStringOrNil("f1b2c3d4-7020-7020-7020-000000000001")
	res, err := h.GetOrCreate(context.Background(), customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"}, "call", nil, "")
	if err == nil {
		t.Fatal("expected a non-nil error when the Redis peer lock is held by another replica")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
	// acquirePeerLock wraps tryAcquireRedisLock's raw ErrTaken with
	// fmt.Errorf("%w"), so errors.As must still unwrap to it.
	var errTaken *redsync.ErrTaken
	if !errors.As(err, &errTaken) {
		t.Errorf("expected err to wrap *redsync.ErrTaken, got: %v", err)
	}
}

// Test_GetOrCreate_redisLockRedisError_failsOpen verifies that a Redis
// connectivity failure does not block GetOrCreate: the call proceeds on
// the in-process lock alone and completes the DB transaction normally.
func Test_GetOrCreate_redisLockRedisError_failsOpen(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockLocker := NewMocklocker(mc)
	mockMutex := NewMockredsyncMutex(mc)

	mockLocker.EXPECT().NewMutex(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockMutex)
	mockMutex.EXPECT().LockContext(gomock.Any()).Return(&redsync.RedisError{Node: 0, Err: errors.New("connection refused")})
	// tryAcquireRedisLock's fail-open path never calls UnlockContext (no
	// lock was actually acquired) -- absence of an EXPECT() here means
	// gomock fails the test if it's called unexpectedly.

	mockCache := cachehandler.NewMockCacheHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)

	h := &caseHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            db,
		notifyHandler: mockNotify,
		peerLocks:     make(map[string]chan struct{}),
		redisLocker:   mockLocker,
	}

	customerID := uuid.FromStringOrNil("f1b2c3d4-7020-7020-7020-000000000002")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now)
	newID := uuid.FromStringOrNil("f1b2c3d4-7020-7020-7020-000000000003")
	mockUtil.EXPECT().UUIDCreate().Return(newID)

	res, err := h.GetOrCreate(context.Background(), customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230002"}, "conversation_message", nil, "")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v, want nil (fail-open on Redis error)", err)
	}
	if res == nil || res.ID != newID {
		t.Errorf("expected a freshly-inserted case %s, got: %v", newID, res)
	}
}

// Test_GetOrCreate_redisLockNil_failsOpen verifies nil-safety explicitly
// at the GetOrCreate entry point (distinct from the existing 19
// getorcreate*_test.go regression tests, which exercise the same nil path
// incidentally): a caseHandler with redisLocker left at its zero value
// (nil) behaves exactly as it did before VOIP-1438 -- in-process lock
// only, DB transaction completes normally.
func Test_GetOrCreate_redisLockNil_failsOpen(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)

	h := &caseHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            db,
		notifyHandler: mockNotify,
		peerLocks:     make(map[string]chan struct{}),
		// redisLocker intentionally left nil.
	}

	customerID := uuid.FromStringOrNil("f1b2c3d4-7020-7020-7020-000000000004")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now)
	newID := uuid.FromStringOrNil("f1b2c3d4-7020-7020-7020-000000000005")
	mockUtil.EXPECT().UUIDCreate().Return(newID)

	res, err := h.GetOrCreate(context.Background(), customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230003"}, "conversation_message", nil, "")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v, want nil (nil redisLocker fail-open)", err)
	}
	if res == nil || res.ID != newID {
		t.Errorf("expected a freshly-inserted case %s, got: %v", newID, res)
	}
}
