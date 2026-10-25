package casehandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

var errLifecycleBoom = errors.New("injected database error")

// newMockedCaseHandler returns a caseHandler backed by gomock handlers, plus
// a sqlmock-backed transaction source for the Tx-scoped dbhandler calls.
func newMockedCaseHandler(t *testing.T) (*caseHandler, *dbhandler.MockDBHandler, *utilhandler.MockUtilHandler, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()

	mc := gomock.NewController(t)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	sdb, smock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = sdb.Close() })

	return &caseHandler{utilHandler: mockUtil, db: mockDB}, mockDB, mockUtil, smock, sdb
}

func beginMockTx(t *testing.T, smock sqlmock.Sqlmock, sdb *sql.DB) *sql.Tx {
	t.Helper()

	smock.ExpectBegin()
	tx, err := sdb.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	return tx
}

func closedSourceCase(customerID, ownerID uuid.UUID, contactID *uuid.UUID) *kase.Case {
	return &kase.Case{
		ID:            uuid.Must(uuid.NewV4()),
		CustomerID:    customerID,
		Peer:          commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555070001"},
		ReferenceType: "call",
		ContactID:     contactID,
		Owner:         commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: ownerID},
		Status:        kase.StatusClosed,
	}
}

func Test_CaseListAll_Delegates(t *testing.T) {
	h, mockDB, _, _, _ := newMockedCaseHandler(t)
	ctx := context.Background()

	expect := []*kase.Case{{ID: uuid.Must(uuid.NewV4())}}
	mockDB.EXPECT().CaseListAll(ctx).Return(expect, nil)

	res, err := h.CaseListAll(ctx)
	if err != nil || len(res) != 1 || res[0].ID != expect[0].ID {
		t.Errorf("CaseListAll() = %v, %v, want the dbhandler result", res, err)
	}

	mockDB.EXPECT().CaseListAll(ctx).Return(nil, errLifecycleBoom)
	if _, err := h.CaseListAll(ctx); !errors.Is(err, errLifecycleBoom) {
		t.Errorf("CaseListAll() error = %v, want the dbhandler error", err)
	}
}

func Test_Close_ErrorPaths(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	caseID := uuid.Must(uuid.NewV4())
	agentID := uuid.Must(uuid.NewV4())
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setup      func(mockDB *dbhandler.MockDBHandler)
		expectIs   error
		expectText string
	}{
		{
			name: "update failure is wrapped and nothing is re-read",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().CaseUpdateStatusClosed(ctx, customerID, caseID, kase.ClosedReasonAgentClosed, string(commonidentity.OwnerTypeAgent), &agentID, &now).Return(false, errLifecycleBoom)
			},
			expectIs:   errLifecycleBoom,
			expectText: "could not close case. Close",
		},
		{
			name: "re-read failure after a successful close is returned",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().CaseUpdateStatusClosed(ctx, customerID, caseID, kase.ClosedReasonAgentClosed, string(commonidentity.OwnerTypeAgent), &agentID, &now).Return(true, nil)
				mockDB.EXPECT().CaseGetByID(ctx, caseID).Return(nil, errLifecycleBoom)
			},
			expectIs: errLifecycleBoom,
		},
		{
			name: "re-read failure after a no-op close is returned",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().CaseUpdateStatusClosed(ctx, customerID, caseID, kase.ClosedReasonAgentClosed, string(commonidentity.OwnerTypeAgent), &agentID, &now).Return(false, nil)
				mockDB.EXPECT().CaseGetByID(ctx, caseID).Return(nil, dbhandler.ErrNotFound)
			},
			expectIs: dbhandler.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, mockUtil, _, _ := newMockedCaseHandler(t)
			mockUtil.EXPECT().TimeNow().Return(&now)
			tt.setup(mockDB)

			res, err := h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, agentID)
			if res != nil {
				t.Errorf("Close() = %v, want nil on error", res)
			}
			if !errors.Is(err, tt.expectIs) {
				t.Errorf("Close() error = %v, want %v", err, tt.expectIs)
			}
			if tt.expectText != "" && (err == nil || !strings.Contains(err.Error(), tt.expectText)) {
				t.Errorf("Close() error = %v, want text %q", err, tt.expectText)
			}
		})
	}
}

func Test_Continue_PreconditionErrors(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	agentID := uuid.Must(uuid.NewV4())

	t.Run("source lookup failure is returned", func(t *testing.T) {
		h, mockDB, _, _, _ := newMockedCaseHandler(t)
		id := uuid.Must(uuid.NewV4())
		mockDB.EXPECT().CaseGetByID(ctx, id).Return(nil, errLifecycleBoom)

		res, err := h.Continue(ctx, customerID, id, commonidentity.OwnerTypeAgent, agentID, false)
		if res != nil || !errors.Is(err, errLifecycleBoom) {
			t.Errorf("Continue() = %v, %v, want nil and the lookup error", res, err)
		}
	})

	t.Run("a case of another customer is reported as not found", func(t *testing.T) {
		h, mockDB, _, _, _ := newMockedCaseHandler(t)
		source := closedSourceCase(uuid.Must(uuid.NewV4()), agentID, nil)
		mockDB.EXPECT().CaseGetByID(ctx, source.ID).Return(source, nil)

		res, err := h.Continue(ctx, customerID, source.ID, commonidentity.OwnerTypeAgent, agentID, true)
		if res != nil || !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Continue() = %v, %v, want nil and ErrNotFound", res, err)
		}
	})
}

func Test_Continue_TransactionFailures(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	agentID := uuid.Must(uuid.NewV4())
	contactID := uuid.Must(uuid.NewV4())
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		contactID  *uuid.UUID
		setup      func(mockDB *dbhandler.MockDBHandler, smock sqlmock.Sqlmock, tx *sql.Tx, source *kase.Case)
		expectIs   error
		expectText string
	}{
		{
			name: "begin failure",
			setup: func(mockDB *dbhandler.MockDBHandler, _ sqlmock.Sqlmock, _ *sql.Tx, _ *kase.Case) {
				mockDB.EXPECT().BeginTx(ctx).Return(nil, errLifecycleBoom)
			},
			expectIs:   errLifecycleBoom,
			expectText: "could not begin transaction. Continue",
		},
		{
			name: "insert failure rolls back",
			setup: func(mockDB *dbhandler.MockDBHandler, smock sqlmock.Sqlmock, tx *sql.Tx, _ *kase.Case) {
				mockDB.EXPECT().BeginTx(ctx).Return(tx, nil)
				mockDB.EXPECT().CaseInsertTx(ctx, tx, gomock.Any()).Return(errLifecycleBoom)
				smock.ExpectRollback()
			},
			expectIs:   errLifecycleBoom,
			expectText: "could not insert case. insertWithRetry",
		},
		{
			name: "insert deadlock rolls back and is reported as ErrDeadlock",
			setup: func(mockDB *dbhandler.MockDBHandler, smock sqlmock.Sqlmock, tx *sql.Tx, _ *kase.Case) {
				mockDB.EXPECT().BeginTx(ctx).Return(tx, nil)
				mockDB.EXPECT().CaseInsertTx(ctx, tx, gomock.Any()).Return(dbhandler.ErrDeadlock)
				smock.ExpectRollback()
			},
			expectIs: dbhandler.ErrDeadlock,
		},
		{
			name:      "contact carry-over failure rolls back",
			contactID: &contactID,
			setup: func(mockDB *dbhandler.MockDBHandler, smock sqlmock.Sqlmock, tx *sql.Tx, _ *kase.Case) {
				mockDB.EXPECT().BeginTx(ctx).Return(tx, nil)
				mockDB.EXPECT().CaseInsertTx(ctx, tx, gomock.Any()).Return(nil)
				mockDB.EXPECT().CaseUpdateContactIDTx(ctx, tx, customerID, gomock.Any(), contactID).Return(errLifecycleBoom)
				smock.ExpectRollback()
			},
			expectIs:   errLifecycleBoom,
			expectText: "could not carry over contact_id. Continue",
		},
		{
			// After a failed Commit database/sql marks the transaction done,
			// so the deferred Rollback never reaches the driver.
			name: "commit failure",
			setup: func(mockDB *dbhandler.MockDBHandler, smock sqlmock.Sqlmock, tx *sql.Tx, _ *kase.Case) {
				mockDB.EXPECT().BeginTx(ctx).Return(tx, nil)
				mockDB.EXPECT().CaseInsertTx(ctx, tx, gomock.Any()).Return(nil)
				smock.ExpectCommit().WillReturnError(errLifecycleBoom)
			},
			expectIs:   errLifecycleBoom,
			expectText: "could not commit transaction. Continue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, mockUtil, smock, sdb := newMockedCaseHandler(t)

			source := closedSourceCase(customerID, agentID, tt.contactID)
			mockDB.EXPECT().CaseGetByID(ctx, source.ID).Return(source, nil)
			mockUtil.EXPECT().TimeNow().Return(&now)
			mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())).AnyTimes()

			var tx *sql.Tx
			if tt.name != "begin failure" {
				tx = beginMockTx(t, smock, sdb)
			}
			tt.setup(mockDB, smock, tx, source)

			res, err := h.Continue(ctx, customerID, source.ID, commonidentity.OwnerTypeAgent, agentID, false)
			if res != nil {
				t.Errorf("Continue() = %v, want nil on error", res)
			}
			if !errors.Is(err, tt.expectIs) {
				t.Errorf("Continue() error = %v, want %v", err, tt.expectIs)
			}
			if tt.expectText != "" && (err == nil || !strings.Contains(err.Error(), tt.expectText)) {
				t.Errorf("Continue() error = %v, want text %q", err, tt.expectText)
			}
			if err := smock.ExpectationsWereMet(); err != nil {
				t.Errorf("transaction expectations: %v", err)
			}
		})
	}
}

func Test_insertWithRetry_ConflictErrorPaths(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555070002"}
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setup      func(mockDB *dbhandler.MockDBHandler, tx *sql.Tx)
		expectIs   error
		expectText string
	}{
		{
			name: "re-select deadlock is reported as ErrDeadlock",
			setup: func(mockDB *dbhandler.MockDBHandler, tx *sql.Tx) {
				mockDB.EXPECT().CaseInsertTx(ctx, tx, gomock.Any()).Return(dbhandler.ErrDuplicate)
				mockDB.EXPECT().CaseGetOpenByPeer(ctx, tx, customerID, peer.Type, peer.Target, "call").Return(nil, dbhandler.ErrDeadlock)
			},
			expectIs: dbhandler.ErrDeadlock,
		},
		{
			name: "re-select failure is wrapped",
			setup: func(mockDB *dbhandler.MockDBHandler, tx *sql.Tx) {
				mockDB.EXPECT().CaseInsertTx(ctx, tx, gomock.Any()).Return(dbhandler.ErrDuplicate)
				mockDB.EXPECT().CaseGetOpenByPeer(ctx, tx, customerID, peer.Type, peer.Target, "call").Return(nil, errLifecycleBoom)
			},
			expectIs:   errLifecycleBoom,
			expectText: "could not re-select after insert conflict. insertWithRetry",
		},
		{
			name: "a winner that closed again every time exhausts the retries",
			setup: func(mockDB *dbhandler.MockDBHandler, tx *sql.Tx) {
				mockDB.EXPECT().CaseInsertTx(ctx, tx, gomock.Any()).Return(dbhandler.ErrDuplicate).Times(maxInsertRetries)
				mockDB.EXPECT().CaseGetOpenByPeer(ctx, tx, customerID, peer.Type, peer.Target, "call").Return(nil, nil).Times(maxInsertRetries)
			},
			expectIs: ErrGetOrCreateExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, mockUtil, smock, sdb := newMockedCaseHandler(t)
			mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())).AnyTimes()
			tx := beginMockTx(t, smock, sdb)
			tt.setup(mockDB, tx)

			res, created, err := h.insertWithRetry(ctx, tx, customerID, commonaddress.Address{}, peer, "call", nil, &now, "")
			if res != nil || created {
				t.Errorf("insertWithRetry() = %v, %v, want nil, false", res, created)
			}
			if !errors.Is(err, tt.expectIs) {
				t.Errorf("insertWithRetry() error = %v, want %v", err, tt.expectIs)
			}
			if tt.expectText != "" && (err == nil || !strings.Contains(err.Error(), tt.expectText)) {
				t.Errorf("insertWithRetry() error = %v, want text %q", err, tt.expectText)
			}
		})
	}
}
