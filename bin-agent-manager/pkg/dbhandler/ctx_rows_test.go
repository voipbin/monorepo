package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// newSQLMockHandler returns a handler backed by a dedicated sqlmock database
// (the shared sqlite dbTest cannot inject row iteration errors).
func newSQLMockHandler(t *testing.T) (*handler, sqlmock.Sqlmock, *cachehandler.MockCacheHandler) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().DoAndReturn(func() *time.Time { return &now }).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

	return &handler{db: db, cache: mockCache, utilHandler: mockUtil}, mock, mockCache
}

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error and is not reported as ErrNotFound (single-row
// getters) or swallowed as an empty or partial success (lists, sweep). The
// first row fails when it is read, so Next() returns false and the cause is
// only reported by rows.Err(). The wrapping uses %v, so the cause is checked
// by its message (agentAddressListByAgentIDs returns it unwrapped).
func Test_RowsErr_NotMisreported(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
	agentColumns := commondatabasehandler.GetDBFields(&agent.Agent{})

	// brokenRows makes the first row fail when it is read.
	brokenRows := func(columns ...string) *sqlmock.Rows {
		values := make([]driver.Value, len(columns))
		return sqlmock.NewRows(columns).AddRow(values...).RowError(0, errRowsBroken)
	}

	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)
		fn     func(h *handler) error
		isWrap bool
	}{
		{
			name: "agentGetFromDB",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(brokenRows(agentColumns...))
			},
			fn: func(h *handler) error { _, err := h.agentGetFromDB(context.Background(), id); return err },
		},
		{
			name: "agentAddressLookupOwnerAgentID",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(brokenRows("agent_id"))
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressLookupOwnerAgentID(context.Background(), h.db, id, commonaddress.TypeTel, "+123")
				return err
			},
		},
		{
			name: "AgentGetByCustomerIDAndAddress",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(brokenRows("agent_id"))
			},
			fn: func(h *handler) error {
				_, err := h.AgentGetByCustomerIDAndAddress(context.Background(), id, &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+123"})
				return err
			},
		},
		{
			name: "AgentList",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(brokenRows(agentColumns...))
			},
			fn: func(h *handler) error { _, err := h.AgentList(context.Background(), 10, "", nil); return err },
		},
		{
			name: "AgentList address hydrate",
			expect: func(mock sqlmock.Sqlmock) {
				values := make([]driver.Value, len(agentColumns))
				values[0] = id.Bytes()
				mock.ExpectQuery("FROM agent_agents").WillReturnRows(sqlmock.NewRows(agentColumns).AddRow(values...))
				mock.ExpectQuery("FROM agent_addresses").WillReturnRows(brokenRows("agent_id"))
			},
			fn: func(h *handler) error { _, err := h.AgentList(context.Background(), 10, "", nil); return err },
		},
		{
			name: "agentAddressListByAgentID",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(brokenRows("type"))
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressListByAgentID(context.Background(), h.db, id)
				return err
			},
		},
		{
			name: "agentAddressListByAgentIDs",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(brokenRows("agent_id"))
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressListByAgentIDs(context.Background(), h.db, []uuid.UUID{id})
				return err
			},
			isWrap: true,
		},
		{
			name: "AgentReserveSweep",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(brokenRows("id"))
			},
			fn: func(h *handler) error {
				n, err := h.AgentReserveSweep(context.Background(), time.Now())
				if err == nil && n != 0 {
					t.Errorf("expected nothing reclaimed on an iteration failure, got: %d", n)
				}
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newSQLMockHandler(t)
			tt.expect(mock)

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
			}
			if !strings.Contains(err.Error(), errRowsBroken.Error()) {
				t.Errorf("expected the iteration failure in the error, got: %v", err)
			}
			if tt.isWrap && !errors.Is(err, errRowsBroken) {
				t.Errorf("expected the cause to be reachable with errors.Is, got: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// Test_ContextCancel_AgentUpdateAborted verifies that the request context
// reaches the database driver: a cancelled context aborts the update. The
// expectation is registered so that the pre-change code (Exec ignores the
// context) consumes it and returns success; with ExecContext database/sql
// checks the context before calling the driver, so the expectation stays
// unconsumed and ExpectationsWereMet must not be checked.
func Test_ContextCancel_AgentUpdateAborted(t *testing.T) {
	h, mock, _ := newSQLMockHandler(t)
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(1, 1))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := h.AgentUpdate(ctx, id, map[agent.Field]any{agent.FieldName: "name"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// Test_AfterCommit_RefreshFailureNotPropagated pins the current behavior: the
// writes are committed in the transaction and the cache refresh afterwards is
// best effort, so its failure (here the refresh query fails with
// context.Canceled) is not returned to the caller. sqlmock has no hook to
// cancel right after Commit, so the failure is injected on the refresh query.
func Test_AfterCommit_RefreshFailureNotPropagated(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
	customerID := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e78")
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+123"}}

	t.Run("AgentCreate", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO agent_agents").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO agent_addresses").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		mock.ExpectQuery("FROM agent_agents").WillReturnError(context.Canceled)

		a := &agent.Agent{Addresses: addrs}
		a.ID = id
		a.CustomerID = customerID
		if err := h.AgentCreate(context.Background(), a); err != nil {
			t.Errorf("expected nil after a committed write, got: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("AgentSetAddresses", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT customer_id").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow(customerID.Bytes()))
		mock.ExpectExec("DELETE FROM agent_addresses").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO agent_addresses").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("UPDATE agent_agents").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		mock.ExpectQuery("FROM agent_agents").WillReturnError(context.Canceled)
		mockCache.EXPECT().AgentDel(gomock.Any(), id).Return(nil)

		if err := h.AgentSetAddresses(context.Background(), id, addrs); err != nil {
			t.Errorf("expected nil after a committed write, got: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}
