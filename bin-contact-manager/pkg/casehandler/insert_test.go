package casehandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_insertWithRetry_InsertConflict_RetrySelectsWinner verifies design
// §4.2's round-2 correction: insertWithRetry re-selects the winning row
// WITH FOR UPDATE (via CaseGetOpenByPeer, not a bare unlocked SELECT) and
// uses it directly -- it does NOT retry the insert a second time when the
// winner is found still open on the first re-select. The only production
// consumer is Continue (design §5.3).
func Test_insertWithRetry_InsertConflict_RetrySelectsWinner(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, db: mockDB}
	ctx := context.Background()

	realTx, err := dbTest.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	defer func() { _ = realTx.Rollback() }()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7007-7007-7007-000000000001")
	loserAttemptID := uuid.FromStringOrNil("f1b2c3d4-7007-7007-7007-000000000002")
	winnerCaseID := uuid.FromStringOrNil("f1b2c3d4-7007-7007-7007-000000000003")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	winnerOpened := now.Add(-1 * time.Minute)

	winner := &kase.Case{
		ID: winnerCaseID, CustomerID: customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551160001"}, ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &winnerOpened, TMCreate: &winnerOpened, TMUpdate: &winnerOpened,
	}

	mockUtil.EXPECT().UUIDCreate().Return(loserAttemptID)
	// This attempt's own INSERT loses the race.
	mockDB.EXPECT().CaseInsertTx(ctx, realTx, gomock.Any()).Return(dbhandler.ErrDuplicate)
	// Retry-select (locked, FOR UPDATE per CaseGetOpenByPeer's contract)
	// finds the winner still open -- must be used directly, no second
	// insert attempt.
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, realTx, customerID, commonaddress.TypeTel, "+15551160001", "call").Return(winner, nil)

	res, isNew, err := h.insertWithRetry(
		ctx, realTx, customerID,
		commonaddress.Address{},
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551160001"},
		"call",
		nil,
		&now,
		"",
	)
	if err != nil {
		t.Fatalf("insertWithRetry() error = %v", err)
	}
	if res == nil || res.ID != winnerCaseID {
		t.Errorf("expected to use the winner case %s, got: %v", winnerCaseID, res)
	}
	if isNew {
		t.Errorf("expected isNew=false when re-selecting an existing winner, got: %v", isNew)
	}
}

// Test_insertWithRetry_RaceWinnerAlsoClosedBeforeReselect verifies design
// §4.2's rarer second race: the row this attempt lost to has ALSO
// transitioned out of 'open' by the time of the locked re-select (e.g.
// closed by a third actor). insertWithRetry must loop and retry the
// insert again, not give up or use the no-longer-open row.
func Test_insertWithRetry_RaceWinnerAlsoClosedBeforeReselect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, db: mockDB}
	ctx := context.Background()

	realTx, err := dbTest.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	defer func() { _ = realTx.Rollback() }()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7008-7008-7008-000000000001")
	firstAttemptID := uuid.FromStringOrNil("f1b2c3d4-7008-7008-7008-000000000002")
	secondAttemptID := uuid.FromStringOrNil("f1b2c3d4-7008-7008-7008-000000000003")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	// Attempt 1: insert conflicts.
	mockUtil.EXPECT().UUIDCreate().Return(firstAttemptID)
	mockDB.EXPECT().CaseInsertTx(ctx, realTx, gomock.Any()).Return(dbhandler.ErrDuplicate)
	// Locked re-select: the row we raced against has ALSO since closed
	// (nil, nil) -- must NOT be treated as "use nil", must loop and retry.
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, realTx, customerID, commonaddress.TypeTel, "+15551170001", "call").Return(nil, nil)

	// Attempt 2: succeeds.
	mockUtil.EXPECT().UUIDCreate().Return(secondAttemptID)
	mockDB.EXPECT().CaseInsertTx(ctx, realTx, gomock.Any()).Return(nil)

	res, isNew, err := h.insertWithRetry(
		ctx, realTx, customerID,
		commonaddress.Address{},
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551170001"},
		"call",
		nil,
		&now,
		"",
	)
	if err != nil {
		t.Fatalf("insertWithRetry() error = %v", err)
	}
	if res == nil || res.ID != secondAttemptID {
		t.Errorf("expected the second attempt's case %s to win, got: %v", secondAttemptID, res)
	}
	if !isNew {
		t.Errorf("expected isNew=true for a fresh successful insert, got: %v", isNew)
	}
}

// Test_insertWithRetry_LoopExhaustion_ReturnsErrGetOrCreateExhausted
// verifies design §4.2's "Loop exhaustion" path: when every bounded
// retry attempt collides (a thundering-herd scenario), insertWithRetry
// surfaces ErrGetOrCreateExhausted rather than silently dropping the
// request or looping forever.
func Test_insertWithRetry_LoopExhaustion_ReturnsErrGetOrCreateExhausted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, db: mockDB}
	ctx := context.Background()

	realTx, err := dbTest.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	defer func() { _ = realTx.Rollback() }()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7009-7009-7009-000000000001")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	// Every attempt (maxInsertRetries=3) conflicts, and every re-select
	// finds nothing open (the row that won each time has already closed
	// by the time we look).
	mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())).Times(maxInsertRetries)
	mockDB.EXPECT().CaseInsertTx(ctx, realTx, gomock.Any()).Return(dbhandler.ErrDuplicate).Times(maxInsertRetries)
	mockDB.EXPECT().CaseGetOpenByPeer(ctx, realTx, customerID, commonaddress.TypeTel, "+15551180001", "call").Return(nil, nil).Times(maxInsertRetries)

	res, isNew, err := h.insertWithRetry(
		ctx, realTx, customerID,
		commonaddress.Address{},
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551180001"},
		"call",
		nil,
		&now,
		"",
	)
	if err != ErrGetOrCreateExhausted {
		t.Errorf("expected ErrGetOrCreateExhausted, got: %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result on exhaustion, got: %v", res)
	}
	if isNew {
		t.Errorf("expected isNew=false on exhaustion, got: %v", isNew)
	}
}
