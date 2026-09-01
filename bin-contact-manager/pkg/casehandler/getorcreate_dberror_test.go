package casehandler

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_GetOrCreate_EmptyPeer_returnsInvalidArgument covers GetOrCreate's
// very first guard clause -- peer.Type/peer.Target validation -- which no
// other existing test exercises (every other test in this package
// supplies a valid peer).
func Test_GetOrCreate_EmptyPeer_returnsInvalidArgument(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify, peerLocks: make(map[string]chan struct{})}

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000001")

	res, err := h.GetOrCreate(context.Background(), customerID, commonaddress.Address{}, commonaddress.Address{}, "call", nil, "")
	if err == nil {
		t.Fatal("expected a non-nil error for an empty peer")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
}

// newRealTxDBMock returns a MockDBHandler whose BeginTx call is backed by
// a genuine dbTest transaction (so getOrCreateAttempt's deferred
// Rollback()/tx.Commit() calls are structurally valid), letting every
// OTHER dbhandler.DBHandler method be mocked freely to simulate arbitrary
// error injection that real SQLite cannot easily reproduce (e.g. a
// non-deadlock driver error from CaseGetByIDForUpdate).
func newRealTxDBMock(t *testing.T, mc *gomock.Controller, ctx context.Context) *dbhandler.MockDBHandler {
	t.Helper()
	mockDB := dbhandler.NewMockDBHandler(mc)
	var tx *sql.Tx
	mockDB.EXPECT().BeginTx(ctx).DoAndReturn(func(_ context.Context) (*sql.Tx, error) {
		var err error
		tx, err = dbTest.Begin()
		return tx, err
	})
	t.Cleanup(func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	})
	return mockDB
}

func newTestHandler(mockUtil *utilhandler.MockUtilHandler, mockReq *requesthandler.MockRequestHandler, mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler) *caseHandler {
	return &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify, peerLocks: make(map[string]chan struct{})}
}

// Test_GetOrCreate_BeginTxFails_returnsError covers getOrCreateAttempt's
// BeginTx error branch.
func Test_GetOrCreate_BeginTxFails_returnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	mockDB.EXPECT().BeginTx(ctx).Return(nil, errors.New("connection refused"))
	mockUtil.EXPECT().TimeNow().Return(timePtr(t, "2026-06-28T12:00:00Z"))

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000002")
	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130002"}, "call", nil, "")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
}

// Test_GetOrCreate_CaseGetByIDForUpdate_GenericError_returnsError covers
// the hint-lookup branch's non-deadlock error path.
func Test_GetOrCreate_CaseGetByIDForUpdate_GenericError_returnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := newRealTxDBMock(t, mc, ctx)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000003")
	hintID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000004")

	mockUtil.EXPECT().TimeNow().Return(timePtr(t, "2026-06-28T12:00:00Z"))
	mockDB.EXPECT().CaseGetByIDForUpdate(ctx, gomock.Any(), customerID, hintID).Return(nil, errors.New("driver: bad connection"))

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130003"}, "call", &hintID, "")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
}

// Test_GetOrCreate_CaseGetByIDForUpdate_Deadlock_retriesAndSucceeds
// covers the hint-lookup branch's deadlock-retry path (distinct from
// getorcreate_deadlock_test.go, which only exercises the no-hint insert
// path's deadlock).
func Test_GetOrCreate_CaseGetByIDForUpdate_Deadlock_retriesAndSucceeds(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000005")
	hintID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000006")
	now1 := timePtr(t, "2026-06-28T12:00:00Z")
	now2 := timePtr(t, "2026-06-28T12:00:01Z")

	var txs []*sql.Tx
	mockDB.EXPECT().BeginTx(ctx).Times(2).DoAndReturn(func(_ context.Context) (*sql.Tx, error) {
		tx, err := dbTest.Begin()
		txs = append(txs, tx)
		return tx, err
	})
	t.Cleanup(func() {
		for _, tx := range txs {
			_ = tx.Rollback()
		}
	})

	mockUtil.EXPECT().TimeNow().Return(now1)
	mockDB.EXPECT().CaseGetByIDForUpdate(ctx, gomock.Any(), customerID, hintID).Return(nil, dbhandler.ErrDeadlock)

	mockUtil.EXPECT().TimeNow().Return(now2)
	hinted := &kase.Case{ID: hintID, CustomerID: customerID, Status: kase.StatusOpen}
	mockDB.EXPECT().CaseGetByIDForUpdate(ctx, gomock.Any(), customerID, hintID).Return(hinted, nil)
	mockDB.EXPECT().CaseUpdateTMUpdateTx(ctx, gomock.Any(), hintID, now2).Return(nil)

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130004"}, "call", &hintID, "")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != hintID {
		t.Errorf("expected the hinted case %s, got: %v", hintID, res)
	}
}

// Test_GetOrCreate_CaseGetOpenByPeer_GenericError_returnsError covers
// getOrCreateInTx's peer-lookup branch's non-deadlock error path.
func Test_GetOrCreate_CaseGetOpenByPeer_GenericError_returnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := newRealTxDBMock(t, mc, ctx)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000007")

	mockUtil.EXPECT().TimeNow().Return(timePtr(t, "2026-06-28T12:00:00Z"))
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130005", "call").Return(nil, errors.New("driver: bad connection"))

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130005"}, "call", nil, "")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
}

// Test_GetOrCreate_CaseUpdateStatusClosedTx_GenericError_returnsError
// covers the timed-out-case close branch's non-deadlock error path.
func Test_GetOrCreate_CaseUpdateStatusClosedTx_GenericError_returnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := newRealTxDBMock(t, mc, ctx)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000008")
	existingID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000009")
	now := timePtr(t, "2026-06-28T12:00:00Z")
	longAgo := now.Add(-999 * time.Hour) // guaranteed to exceed any configured timeout

	existing := &kase.Case{ID: existingID, CustomerID: customerID, Status: kase.StatusOpen, TMUpdate: &longAgo}

	mockUtil.EXPECT().TimeNow().Return(now)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130006", "call").Return(existing, nil)
	mockDB.EXPECT().CaseUpdateStatusClosedTx(ctx, gomock.Any(), customerID, existingID, kase.ClosedReasonTimeout, kase.ClosedByTypeSystem, nil, now).Return(false, errors.New("driver: bad connection"))

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130006"}, "call", nil, "")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
}

// Test_GetOrCreate_CaseUpdateStatusClosedTx_Deadlock_retriesAndSucceeds
// covers the timed-out-case close branch's deadlock-retry path.
func Test_GetOrCreate_CaseUpdateStatusClosedTx_Deadlock_retriesAndSucceeds(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-00000000000a")
	existingID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-00000000000b")
	newID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-00000000000c")
	now1 := timePtr(t, "2026-06-28T12:00:00Z")
	now2 := timePtr(t, "2026-06-28T12:00:01Z")
	longAgo1 := now1.Add(-999 * time.Hour)
	longAgo2 := now2.Add(-999 * time.Hour)

	var txs []*sql.Tx
	mockDB.EXPECT().BeginTx(ctx).Times(2).DoAndReturn(func(_ context.Context) (*sql.Tx, error) {
		tx, err := dbTest.Begin()
		txs = append(txs, tx)
		return tx, err
	})
	t.Cleanup(func() {
		for _, tx := range txs {
			_ = tx.Rollback()
		}
	})

	existing1 := &kase.Case{ID: existingID, CustomerID: customerID, Status: kase.StatusOpen, TMUpdate: &longAgo1}
	mockUtil.EXPECT().TimeNow().Return(now1)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130007", "call").Return(existing1, nil)
	mockDB.EXPECT().CaseUpdateStatusClosedTx(ctx, gomock.Any(), customerID, existingID, kase.ClosedReasonTimeout, kase.ClosedByTypeSystem, nil, now1).Return(false, dbhandler.ErrDeadlock)

	existing2 := &kase.Case{ID: existingID, CustomerID: customerID, Status: kase.StatusOpen, TMUpdate: &longAgo2}
	mockUtil.EXPECT().TimeNow().Return(now2)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130007", "call").Return(existing2, nil)
	mockDB.EXPECT().CaseUpdateStatusClosedTx(ctx, gomock.Any(), customerID, existingID, kase.ClosedReasonTimeout, kase.ClosedByTypeSystem, nil, now2).Return(true, nil)
	mockUtil.EXPECT().UUIDCreate().Return(newID)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().CaseUpdateTMUpdateTx(ctx, gomock.Any(), newID, now2).Return(nil)

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130007"}, "call", nil, "")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != newID {
		t.Errorf("expected the freshly-inserted case %s, got: %v", newID, res)
	}
}

// Test_GetOrCreate_CaseGetLastClosedByPeerTx_GenericError_returnsError
// covers the fresh-insert branch's previous_case_id lookup error path.
func Test_GetOrCreate_CaseGetLastClosedByPeerTx_GenericError_returnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := newRealTxDBMock(t, mc, ctx)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-00000000000d")

	mockUtil.EXPECT().TimeNow().Return(timePtr(t, "2026-06-28T12:00:00Z"))
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130008", "call").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130008", "call").Return(nil, errors.New("driver: bad connection"))

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130008"}, "call", nil, "")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
}

// Test_GetOrCreate_CaseGetLastClosedByPeerTx_Deadlock_retriesAndSucceeds
// covers the fresh-insert branch's previous_case_id lookup deadlock-retry
// path.
func Test_GetOrCreate_CaseGetLastClosedByPeerTx_Deadlock_retriesAndSucceeds(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-00000000000e")
	newID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-00000000000f")
	now1 := timePtr(t, "2026-06-28T12:00:00Z")
	now2 := timePtr(t, "2026-06-28T12:00:01Z")

	var txs []*sql.Tx
	mockDB.EXPECT().BeginTx(ctx).Times(2).DoAndReturn(func(_ context.Context) (*sql.Tx, error) {
		tx, err := dbTest.Begin()
		txs = append(txs, tx)
		return tx, err
	})
	t.Cleanup(func() {
		for _, tx := range txs {
			_ = tx.Rollback()
		}
	})

	mockUtil.EXPECT().TimeNow().Return(now1)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130009", "call").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130009", "call").Return(nil, dbhandler.ErrDeadlock)

	mockUtil.EXPECT().TimeNow().Return(now2)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130009", "call").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130009", "call").Return(nil, nil)
	mockUtil.EXPECT().UUIDCreate().Return(newID)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().CaseUpdateTMUpdateTx(ctx, gomock.Any(), newID, now2).Return(nil)

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130009"}, "call", nil, "")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != newID {
		t.Errorf("expected the freshly-inserted case %s, got: %v", newID, res)
	}
}

// Test_GetOrCreate_CaseInsertTx_GenericError_returnsError covers
// insertWithRetry's non-deadlock, non-duplicate CaseInsertTx error path.
func Test_GetOrCreate_CaseInsertTx_GenericError_returnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := newRealTxDBMock(t, mc, ctx)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000010")
	newID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000011")

	mockUtil.EXPECT().TimeNow().Return(timePtr(t, "2026-06-28T12:00:00Z"))
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130010", "call").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130010", "call").Return(nil, nil)
	mockUtil.EXPECT().UUIDCreate().Return(newID)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(errors.New("driver: bad connection"))

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130010"}, "call", nil, "")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
}

// Test_GetOrCreate_InsertRetry_ReselectDeadlock_returnsDeadlock covers
// insertWithRetry's ON-DUPLICATE re-select branch's deadlock path.
func Test_GetOrCreate_InsertRetry_ReselectDeadlock_returnsDeadlock(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000012")
	attempt1ID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000013")
	newID2 := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000014")
	now1 := timePtr(t, "2026-06-28T12:00:00Z")
	now2 := timePtr(t, "2026-06-28T12:00:01Z")

	var txs []*sql.Tx
	mockDB.EXPECT().BeginTx(ctx).Times(2).DoAndReturn(func(_ context.Context) (*sql.Tx, error) {
		tx, err := dbTest.Begin()
		txs = append(txs, tx)
		return tx, err
	})
	t.Cleanup(func() {
		for _, tx := range txs {
			_ = tx.Rollback()
		}
	})

	// Attempt 1: insert collides (ErrDuplicate), re-select hits a
	// deadlock -- GetOrCreate's outer loop restarts from a fresh BeginTx.
	mockUtil.EXPECT().TimeNow().Return(now1)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130011", "call").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130011", "call").Return(nil, nil)
	mockUtil.EXPECT().UUIDCreate().Return(attempt1ID)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(dbhandler.ErrDuplicate)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130011", "call").Return(nil, dbhandler.ErrDeadlock)

	// Attempt 2: succeeds cleanly.
	mockUtil.EXPECT().TimeNow().Return(now2)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130011", "call").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130011", "call").Return(nil, nil)
	mockUtil.EXPECT().UUIDCreate().Return(newID2)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().CaseUpdateTMUpdateTx(ctx, gomock.Any(), newID2, now2).Return(nil)

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130011"}, "call", nil, "")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != newID2 {
		t.Errorf("expected the second attempt's freshly-inserted case %s, got: %v", newID2, res)
	}
}

// Test_GetOrCreate_InsertRetry_ReselectGenericError_returnsError covers
// insertWithRetry's ON-DUPLICATE re-select branch's non-deadlock error
// path.
func Test_GetOrCreate_InsertRetry_ReselectGenericError_returnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := newRealTxDBMock(t, mc, ctx)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000015")
	attemptID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000016")

	mockUtil.EXPECT().TimeNow().Return(timePtr(t, "2026-06-28T12:00:00Z"))
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130012", "call").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130012", "call").Return(nil, nil)
	mockUtil.EXPECT().UUIDCreate().Return(attemptID)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(dbhandler.ErrDuplicate)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130012", "call").Return(nil, errors.New("driver: bad connection"))

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130012"}, "call", nil, "")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
}

// Test_GetOrCreate_CaseUpdateTMUpdateTx_GenericError_returnsError covers
// getOrCreateAttempt's tm_update bump non-deadlock error path.
func Test_GetOrCreate_CaseUpdateTMUpdateTx_GenericError_returnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	ctx := context.Background()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := newRealTxDBMock(t, mc, ctx)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := newTestHandler(mockUtil, mockReq, mockDB, mockNotify)

	customerID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000017")
	newID := uuid.FromStringOrNil("f1b2c3d4-7030-7030-7030-000000000018")
	now := timePtr(t, "2026-06-28T12:00:00Z")

	mockUtil.EXPECT().TimeNow().Return(now)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130013", "call").Return(nil, nil)
	mockDB.EXPECT().CaseGetLastClosedByPeerTx(ctx, gomock.Any(), customerID, commonaddress.TypeTel, "+15551130013", "call").Return(nil, nil)
	mockUtil.EXPECT().UUIDCreate().Return(newID)
	mockDB.EXPECT().CaseInsertTx(ctx, gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().CaseUpdateTMUpdateTx(ctx, gomock.Any(), newID, now).Return(errors.New("driver: bad connection"))

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130013"}, "call", nil, "")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if res != nil {
		t.Errorf("expected a nil result, got: %v", res)
	}
}

// timePtr is a small helper for constructing *time.Time literals inline
// (RFC3339, UTC) without repeating time.Parse's error-handling boilerplate
// across every test above.
func timePtr(t *testing.T, rfc3339 string) *time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		t.Fatalf("could not parse test time %q: %v", rfc3339, err)
	}
	return &tm
}
