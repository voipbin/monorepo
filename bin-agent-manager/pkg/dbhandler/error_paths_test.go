package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/DATA-DOG/go-sqlmock"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
)

// The tests in this file drive the database error paths through sqlmock, so
// they depend on neither the shared sqlite database nor on the wall clock.

var errTestBoom = errors.New("simulated database failure")

// testSeqID returns a deterministic uuid derived from n.
func testSeqID(n int) uuid.UUID {
	b := make([]byte, 16)
	b[0] = 0xe5
	b[12] = byte(n >> 24)
	b[13] = byte(n >> 16)
	b[14] = byte(n >> 8)
	b[15] = byte(n)
	return uuid.FromBytesOrNil(b)
}

// testAgentRow returns a result set holding one agent row with the given id
// (the remaining columns are NULL).
func testAgentRow(id uuid.UUID) *sqlmock.Rows {
	columns := commondatabasehandler.GetDBFields(&agent.Agent{})
	values := make([]driver.Value, len(columns))
	values[0] = id.Bytes()
	return sqlmock.NewRows(columns).AddRow(values...)
}

// testEmptyAddressRows returns an empty address result set for the single
// agent address query.
func testEmptyAddressRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"type", "target", "target_name", "name", "detail"})
}

// testExpectRefresh registers the queries of agentUpdateToCache for the agent
// id, followed by the cache write that is made to fail.
func testExpectRefresh(mock sqlmock.Sqlmock, mockCache *cachehandler.MockCacheHandler, id uuid.UUID) {
	mock.ExpectQuery("FROM agent_agents").WithArgs(id.Bytes()).WillReturnRows(testAgentRow(id))
	mock.ExpectQuery("FROM agent_addresses").WithArgs(id.Bytes()).WillReturnRows(testEmptyAddressRows())
	mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any()).Return(errTestBoom)
}

func Test_isDuplicateKeyErr(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil", nil, false},
		{"mysql duplicate entry", &gomysql.MySQLError{Number: 1062, Message: "Duplicate entry"}, true},
		{"wrapped mysql duplicate entry", fmt.Errorf("insert failed: %w", &gomysql.MySQLError{Number: 1062}), true},
		{"other mysql error", &gomysql.MySQLError{Number: 1048, Message: "Column cannot be null"}, false},
		{"text mentions 1062 but is not a mysql error", errors.New("error 1062: unrelated message"), false},
		{"sqlite unique constraint", errors.New("UNIQUE constraint failed: agent_addresses.id"), true},
		{"unrelated error", errTestBoom, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDuplicateKeyErr(tt.err); got != tt.expect {
				t.Errorf("isDuplicateKeyErr() = %v, expected: %v", got, tt.expect)
			}
		})
	}
}

func Test_agentAddress_errors(t *testing.T) {
	agentID := testSeqID(1)
	customerID := testSeqID(2)
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100"}}

	addressColumns := []string{"type", "target", "target_name", "name", "detail"}

	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)
		fn     func(h *handler) error

		expectIs       error
		expectNotIs    error
		expectContains string
	}{
		{
			name: "insert duplicate key maps to ErrAlreadyExists",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO agent_addresses").WillReturnError(&gomysql.MySQLError{Number: 1062})
			},
			fn: func(h *handler) error {
				return h.agentAddressInsert(context.Background(), h.db, agentID, customerID, addrs)
			},
			expectIs: ErrAlreadyExists,
		},
		{
			name: "insert generic failure is wrapped and not ErrAlreadyExists",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO agent_addresses").WillReturnError(errTestBoom)
			},
			fn: func(h *handler) error {
				return h.agentAddressInsert(context.Background(), h.db, agentID, customerID, addrs)
			},
			expectIs:       errTestBoom,
			expectNotIs:    ErrAlreadyExists,
			expectContains: "agentAddressInsert",
		},
		{
			name: "delete exec failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM agent_addresses").WillReturnError(errTestBoom)
			},
			fn: func(h *handler) error {
				return h.agentAddressDeleteByAgentID(context.Background(), h.db, agentID)
			},
			expectIs:       errTestBoom,
			expectContains: "agentAddressDeleteByAgentID",
		},
		{
			name: "list by agent id query failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_addresses").WillReturnError(errTestBoom)
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressListByAgentID(context.Background(), h.db, agentID)
				return err
			},
			expectIs:       errTestBoom,
			expectContains: "agentAddressListByAgentID",
		},
		{
			name: "list by agent id scan failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_addresses").WillReturnRows(sqlmock.NewRows([]string{"type"}).AddRow("tel"))
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressListByAgentID(context.Background(), h.db, agentID)
				return err
			},
			expectContains: "could not scan row. agentAddressListByAgentID",
		},
		{
			name: "list by agent ids query failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_addresses").WillReturnError(errTestBoom)
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressListByAgentIDs(context.Background(), h.db, []uuid.UUID{agentID})
				return err
			},
			expectIs:       errTestBoom,
			expectContains: "agentAddressListByAgentIDs",
		},
		{
			name: "list by agent ids scan failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_addresses").WillReturnRows(sqlmock.NewRows([]string{"agent_id"}).AddRow(agentID.Bytes()))
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressListByAgentIDs(context.Background(), h.db, []uuid.UUID{agentID})
				return err
			},
			expectContains: "could not scan row. agentAddressListByAgentIDs",
		},
		{
			name: "list by agent ids parse failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_addresses").WillReturnRows(
					sqlmock.NewRows(append([]string{"agent_id"}, addressColumns...)).AddRow([]byte{1, 2}, "tel", "t", "tn", "n", "d"),
				)
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressListByAgentIDs(context.Background(), h.db, []uuid.UUID{agentID})
				return err
			},
			expectContains: "could not parse agent_id. agentAddressListByAgentIDs",
		},
		{
			name: "lookup query failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_addresses").WillReturnError(errTestBoom)
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressLookupOwnerAgentID(context.Background(), h.db, customerID, commonaddress.TypeTel, "+15550100")
				return err
			},
			expectIs:       errTestBoom,
			expectNotIs:    ErrNotFound,
			expectContains: "agentAddressLookupOwnerAgentID",
		},
		{
			name: "lookup scan failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_addresses").WillReturnRows(sqlmock.NewRows([]string{"agent_id", "extra"}).AddRow(agentID.Bytes(), "x"))
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressLookupOwnerAgentID(context.Background(), h.db, customerID, commonaddress.TypeTel, "+15550100")
				return err
			},
			expectNotIs:    ErrNotFound,
			expectContains: "could not scan row. agentAddressLookupOwnerAgentID",
		},
		{
			name: "lookup parse failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_addresses").WillReturnRows(sqlmock.NewRows([]string{"agent_id"}).AddRow([]byte{1}))
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressLookupOwnerAgentID(context.Background(), h.db, customerID, commonaddress.TypeTel, "+15550100")
				return err
			},
			expectNotIs:    ErrNotFound,
			expectContains: "could not parse agent_id. agentAddressLookupOwnerAgentID",
		},
		{
			name: "lookup without a match is ErrNotFound",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_addresses").WillReturnRows(sqlmock.NewRows([]string{"agent_id"}))
			},
			fn: func(h *handler) error {
				_, err := h.agentAddressLookupOwnerAgentID(context.Background(), h.db, customerID, commonaddress.TypeTel, "+15550100")
				return err
			},
			expectIs: ErrNotFound,
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
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("expected errors.Is(%v), got: %v", tt.expectIs, err)
			}
			if tt.expectNotIs != nil && errors.Is(err, tt.expectNotIs) {
				t.Errorf("expected the error not to be %v, got: %v", tt.expectNotIs, err)
			}
			if tt.expectContains != "" && !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("expected the error to contain %q, got: %v", tt.expectContains, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func Test_agentAddressLookupOwnerAgentID_found(t *testing.T) {
	h, mock, _ := newSQLMockHandler(t)
	agentID := testSeqID(3)
	customerID := testSeqID(4)

	mock.ExpectQuery("FROM agent_addresses").
		WithArgs(customerID.Bytes(), "tel", "+15550101").
		WillReturnRows(sqlmock.NewRows([]string{"agent_id"}).AddRow(agentID.Bytes()))

	res, err := h.agentAddressLookupOwnerAgentID(context.Background(), h.db, customerID, commonaddress.TypeTel, "+15550101")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != agentID {
		t.Errorf("owner mismatch. expected: %s, got: %s", agentID, res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// Test_agentAddressListByAgentIDs_chunks verifies that the IN(...) list is
// split at agentAddressINChunkSize and that the chunk contents and the merge
// of the per-chunk results are exact.
func Test_agentAddressListByAgentIDs_chunks(t *testing.T) {
	tests := []struct {
		name         string
		count        int
		expectChunks []int
	}{
		{"empty input issues no query", 0, nil},
		{"exactly one full chunk", 900, []int{900}},
		{"one id over the chunk size", 901, []int{900, 1}},
		{"three chunks", 1801, []int{900, 900, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newSQLMockHandler(t)

			ids := make([]uuid.UUID, tt.count)
			for i := range ids {
				ids[i] = testSeqID(i + 1)
			}

			start := 0
			for _, size := range tt.expectChunks {
				args := make([]driver.Value, 0, size)
				for _, id := range ids[start : start+size] {
					args = append(args, id.Bytes())
				}

				// one address row for the last id of the chunk
				last := ids[start+size-1]
				rows := sqlmock.NewRows([]string{"agent_id", "type", "target", "target_name", "name", "detail"}).
					AddRow(last.Bytes(), "tel", fmt.Sprintf("+1555%07d", start+size), "", "", "")

				mock.ExpectQuery(regexp.QuoteMeta("FROM agent_addresses WHERE agent_id IN (")).WithArgs(args...).WillReturnRows(rows)
				start += size
			}

			res, err := h.agentAddressListByAgentIDs(context.Background(), h.db, ids)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}

			if len(res) != len(tt.expectChunks) {
				t.Fatalf("result size mismatch. expected: %d, got: %d", len(tt.expectChunks), len(res))
			}
			start = 0
			for _, size := range tt.expectChunks {
				last := ids[start+size-1]
				got := res[last]
				if len(got) != 1 || got[0].Target != fmt.Sprintf("+1555%07d", start+size) {
					t.Errorf("merged addresses of chunk ending at %d mismatch: %v", start+size, got)
				}
				start += size
			}
		})
	}
}

func Test_AgentReserve_errors(t *testing.T) {
	agentID := testSeqID(10)
	refID := testSeqID(11)

	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)
		fn     func(t *testing.T, h *handler) error

		expectContains string
	}{
		{
			name: "reserve exec failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE agent_agents").WillReturnError(errTestBoom)
			},
			fn: func(t *testing.T, h *handler) error {
				ok, err := h.AgentReserve(context.Background(), agentID, "queuecall", refID)
				if ok {
					t.Errorf("expected no reservation on a failed update")
				}
				return err
			},
			expectContains: "could not execute query. AgentReserve",
		},
		{
			name: "reserve rows affected failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE agent_agents").WillReturnResult(sqlmock.NewErrorResult(errTestBoom))
			},
			fn: func(t *testing.T, h *handler) error {
				ok, err := h.AgentReserve(context.Background(), agentID, "queuecall", refID)
				if ok {
					t.Errorf("expected no reservation when the affected rows are unknown")
				}
				return err
			},
			expectContains: "could not get rows affected. AgentReserve",
		},
		{
			name: "release exec failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE agent_agents").WillReturnError(errTestBoom)
			},
			fn: func(t *testing.T, h *handler) error {
				return h.AgentReserveRelease(context.Background(), agentID, refID)
			},
			expectContains: "could not execute query. AgentReserveRelease",
		},
		{
			name: "sweep query failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM agent_agents").WillReturnError(errTestBoom)
			},
			fn: func(t *testing.T, h *handler) error {
				_, err := h.AgentReserveSweep(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
				return err
			},
			expectContains: "could not query. AgentReserveSweep",
		},
		{
			name: "sweep scan failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM agent_agents").WillReturnRows(sqlmock.NewRows([]string{"id", "extra"}).AddRow(agentID.Bytes(), "x"))
			},
			fn: func(t *testing.T, h *handler) error {
				_, err := h.AgentReserveSweep(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
				return err
			},
			expectContains: "could not scan row. AgentReserveSweep",
		},
		{
			name: "sweep parse failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM agent_agents").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow([]byte{1, 2, 3}))
			},
			fn: func(t *testing.T, h *handler) error {
				_, err := h.AgentReserveSweep(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
				return err
			},
			expectContains: "could not parse id. AgentReserveSweep",
		},
		{
			name: "sweep update failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id FROM agent_agents").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(agentID.Bytes()))
				mock.ExpectExec("UPDATE agent_agents").WillReturnError(errTestBoom)
			},
			fn: func(t *testing.T, h *handler) error {
				n, err := h.AgentReserveSweep(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
				if n != 0 {
					t.Errorf("expected nothing reclaimed on a failed update, got: %d", n)
				}
				return err
			},
			expectContains: "could not execute update. AgentReserveSweep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newSQLMockHandler(t)
			tt.expect(mock)

			err := tt.fn(t, h)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("expected the error to contain %q, got: %v", tt.expectContains, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// Test_AgentReserveSweep_refreshesEachAgent verifies that every reclaimed
// agent gets exactly one cache refresh, in the order the rows were selected.
func Test_AgentReserveSweep_refreshesEachAgent(t *testing.T) {
	h, mock, mockCache := newSQLMockHandler(t)

	id1 := testSeqID(20)
	id2 := testSeqID(21)

	mock.ExpectQuery("SELECT id FROM agent_agents").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id1.Bytes()).AddRow(id2.Bytes()))
	mock.ExpectExec("UPDATE agent_agents").WillReturnResult(sqlmock.NewResult(0, 2))

	cached := []uuid.UUID{}
	for _, id := range []uuid.UUID{id1, id2} {
		mock.ExpectQuery("FROM agent_agents").WithArgs(id.Bytes()).WillReturnRows(testAgentRow(id))
		mock.ExpectQuery("FROM agent_addresses").WithArgs(id.Bytes()).WillReturnRows(testEmptyAddressRows())
	}
	mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, a *agent.Agent) error {
		cached = append(cached, a.ID)
		return nil
	}).Times(2)

	n, err := h.AgentReserveSweep(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("reclaimed count mismatch. expected: 2, got: %d", n)
	}
	if len(cached) != 2 || cached[0] != id1 || cached[1] != id2 {
		t.Errorf("cache refresh order mismatch. expected: [%s %s], got: %v", id1, id2, cached)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_AgentCountByCustomerID_scanFailure(t *testing.T) {
	h, mock, _ := newSQLMockHandler(t)
	customerID := testSeqID(30)

	mock.ExpectQuery("SELECT COUNT").WithArgs(customerID.Bytes()).WillReturnError(errTestBoom)

	n, err := h.AgentCountByCustomerID(context.Background(), customerID)
	if !errors.Is(err, errTestBoom) {
		t.Errorf("expected the query failure, got: %v", err)
	}
	if n != 0 {
		t.Errorf("expected a zero count on failure, got: %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_AgentCreate_errors(t *testing.T) {
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550102"}}

	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)

		expectIs       error
		expectNotIs    error
		expectContains string
	}{
		{
			name: "begin failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errTestBoom)
			},
			expectIs:       errTestBoom,
			expectContains: "could not begin transaction",
		},
		{
			name: "agent insert failure rolls back",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO agent_agents").WillReturnError(errTestBoom)
				mock.ExpectRollback()
			},
			expectIs:       errTestBoom,
			expectContains: "could not execute query. AgentCreate",
		},
		{
			name: "duplicate address maps to ErrAlreadyExists and rolls back",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO agent_agents").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("INSERT INTO agent_addresses").WillReturnError(&gomysql.MySQLError{Number: 1062})
				mock.ExpectRollback()
			},
			expectIs: ErrAlreadyExists,
		},
		{
			name: "other address failure rolls back and is not ErrAlreadyExists",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO agent_agents").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("INSERT INTO agent_addresses").WillReturnError(errTestBoom)
				mock.ExpectRollback()
			},
			expectIs:    errTestBoom,
			expectNotIs: ErrAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no cache expectation is registered: a failed create must not touch the cache
			h, mock, _ := newSQLMockHandler(t)
			tt.expect(mock)

			a := &agent.Agent{Addresses: addrs}
			a.ID = testSeqID(40)
			a.CustomerID = testSeqID(41)

			err := h.AgentCreate(context.Background(), a)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("expected errors.Is(%v), got: %v", tt.expectIs, err)
			}
			if tt.expectNotIs != nil && errors.Is(err, tt.expectNotIs) {
				t.Errorf("expected the error not to be %v, got: %v", tt.expectNotIs, err)
			}
			if tt.expectContains != "" && !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("expected the error to contain %q, got: %v", tt.expectContains, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func Test_AgentDelete_errors(t *testing.T) {
	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)

		expectIs       error
		expectContains string
	}{
		{
			name: "update failure rolls back",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE agent_agents").WillReturnError(errTestBoom)
				mock.ExpectRollback()
			},
			expectIs:       errTestBoom,
			expectContains: "could not execute update. AgentDelete",
		},
		{
			name: "address delete failure rolls back",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE agent_agents").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("DELETE FROM agent_addresses").WillReturnError(errTestBoom)
				mock.ExpectRollback()
			},
			expectIs:       errTestBoom,
			expectContains: "agentAddressDeleteByAgentID",
		},
		{
			name: "commit failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE agent_agents").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("DELETE FROM agent_addresses").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit().WillReturnError(errTestBoom)
			},
			expectIs:       errTestBoom,
			expectContains: "could not commit transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no cache expectation is registered: a failed delete must not invalidate the cache
			h, mock, _ := newSQLMockHandler(t)
			tt.expect(mock)

			err := h.AgentDelete(context.Background(), testSeqID(50))
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !errors.Is(err, tt.expectIs) {
				t.Errorf("expected errors.Is(%v), got: %v", tt.expectIs, err)
			}
			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("expected the error to contain %q, got: %v", tt.expectContains, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func Test_agentGetFromDB_errors(t *testing.T) {
	id := testSeqID(60)

	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)

		expectNotFound bool
		expectIs       error
		expectContains string
	}{
		{
			name: "query failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_agents").WillReturnError(errTestBoom)
			},
			expectIs:       errTestBoom,
			expectContains: "could not query. agentGetFromDB",
		},
		{
			name: "no row is ErrNotFound",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_agents").WillReturnRows(sqlmock.NewRows(commondatabasehandler.GetDBFields(&agent.Agent{})))
			},
			expectNotFound: true,
		},
		{
			name: "row conversion failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_agents").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()))
			},
			expectContains: "could not get data from row. agentGetFromDB",
		},
		{
			name: "closing the agent row fails",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_agents").WillReturnRows(testAgentRow(id).CloseError(errTestBoom))
			},
			expectIs:       errTestBoom,
			expectContains: "could not close agent row. agentGetFromDB",
		},
		{
			name: "address hydrate failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_agents").WillReturnRows(testAgentRow(id))
				mock.ExpectQuery("FROM agent_addresses").WillReturnError(errTestBoom)
			},
			expectIs:       errTestBoom,
			expectContains: "could not hydrate addresses. agentGetFromDB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newSQLMockHandler(t)
			tt.expect(mock)

			res, err := h.agentGetFromDB(context.Background(), id)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if res != nil {
				t.Errorf("expected no agent on failure, got: %v", res)
			}
			if tt.expectNotFound {
				if err != ErrNotFound {
					t.Errorf("expected ErrNotFound, got: %v", err)
				}
			} else if errors.Is(err, ErrNotFound) {
				t.Errorf("a failure must not be reported as ErrNotFound. err: %v", err)
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("expected errors.Is(%v), got: %v", tt.expectIs, err)
			}
			if tt.expectContains != "" && !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("expected the error to contain %q, got: %v", tt.expectContains, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func Test_AgentGet_cache(t *testing.T) {
	id := testSeqID(70)

	t.Run("cache hit does not touch the database", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)

		cached := &agent.Agent{}
		cached.ID = id
		mockCache.EXPECT().AgentGet(gomock.Any(), id).Return(cached, nil)

		res, err := h.AgentGet(context.Background(), id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != cached {
			t.Errorf("expected the cached object to be returned as is")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("cache miss and database failure", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)

		mockCache.EXPECT().AgentGet(gomock.Any(), id).Return(nil, errTestBoom)
		mock.ExpectQuery("FROM agent_agents").WillReturnError(errTestBoom)

		res, err := h.AgentGet(context.Background(), id)
		if !errors.Is(err, errTestBoom) {
			t.Errorf("expected the database failure, got: %v", err)
		}
		if res != nil {
			t.Errorf("expected no agent on failure, got: %v", res)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("cache miss and not found", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)

		mockCache.EXPECT().AgentGet(gomock.Any(), id).Return(nil, errTestBoom)
		mock.ExpectQuery("FROM agent_agents").WillReturnRows(sqlmock.NewRows(commondatabasehandler.GetDBFields(&agent.Agent{})))

		if _, err := h.AgentGet(context.Background(), id); err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("cache write failure is ignored", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)

		mockCache.EXPECT().AgentGet(gomock.Any(), id).Return(nil, errTestBoom)
		mock.ExpectQuery("FROM agent_agents").WillReturnRows(testAgentRow(id))
		mock.ExpectQuery("FROM agent_addresses").WillReturnRows(testEmptyAddressRows())
		mockCache.EXPECT().AgentSet(gomock.Any(), gomock.Any()).Return(errTestBoom)

		res, err := h.AgentGet(context.Background(), id)
		if err != nil {
			t.Fatalf("a cache write failure must not fail the read, got: %v", err)
		}
		if res == nil || res.ID != id {
			t.Errorf("expected the database agent, got: %v", res)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}

func Test_AgentList_errors(t *testing.T) {
	tests := []struct {
		name    string
		filters map[agent.Field]any
		expect  func(mock sqlmock.Sqlmock)

		expectIs       error
		expectContains string
	}{
		{
			name:           "unsupported filter type",
			filters:        map[agent.Field]any{agent.FieldName: struct{}{}},
			expect:         func(mock sqlmock.Sqlmock) {},
			expectContains: "could not apply filters. AgentList",
		},
		{
			name:           "tag_ids filter of the wrong type",
			filters:        map[agent.Field]any{agent.FieldTagIDs: 123},
			expect:         func(mock sqlmock.Sqlmock) {},
			expectContains: "invalid tag_ids filter type. AgentList",
		},
		{
			name:           "tag_ids filter with an invalid id",
			filters:        map[agent.Field]any{agent.FieldTagIDs: "not-a-uuid"},
			expect:         func(mock sqlmock.Sqlmock) {},
			expectContains: "invalid tag_ids filter. AgentList",
		},
		{
			name:    "query failure",
			filters: nil,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_agents").WillReturnError(errTestBoom)
			},
			expectIs:       errTestBoom,
			expectContains: "could not query. AgentList",
		},
		{
			name:    "row conversion failure",
			filters: nil,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM agent_agents").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testSeqID(80).Bytes()))
			},
			expectContains: "could not get data. AgentList",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newSQLMockHandler(t)
			tt.expect(mock)

			res, err := h.AgentList(context.Background(), 10, "", tt.filters)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if res != nil {
				t.Errorf("expected no result on failure, got: %v", res)
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("expected errors.Is(%v), got: %v", tt.expectIs, err)
			}
			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("expected the error to contain %q, got: %v", tt.expectContains, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// Test_AgentList_hydratesAddresses verifies that the batch hydration assigns
// each agent its own addresses and gives an agent without any an empty,
// non-nil slice.
func Test_AgentList_hydratesAddresses(t *testing.T) {
	h, mock, _ := newSQLMockHandler(t)

	withAddr := testSeqID(90)
	withoutAddr := testSeqID(91)

	columns := commondatabasehandler.GetDBFields(&agent.Agent{})
	rowValues := func(id uuid.UUID) []driver.Value {
		values := make([]driver.Value, len(columns))
		values[0] = id.Bytes()
		return values
	}
	mock.ExpectQuery("FROM agent_agents").WillReturnRows(
		sqlmock.NewRows(columns).AddRow(rowValues(withAddr)...).AddRow(rowValues(withoutAddr)...),
	)
	mock.ExpectQuery("FROM agent_addresses").WithArgs(withAddr.Bytes(), withoutAddr.Bytes()).WillReturnRows(
		sqlmock.NewRows([]string{"agent_id", "type", "target", "target_name", "name", "detail"}).
			AddRow(withAddr.Bytes(), "tel", "+15550103", "tn", "n", "d"),
	)

	res, err := h.AgentList(context.Background(), 10, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("result size mismatch. expected: 2, got: %d", len(res))
	}

	expectAddrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550103", TargetName: "tn", Name: "n", Detail: "d"}}
	if res[0].ID != withAddr || len(res[0].Addresses) != 1 || res[0].Addresses[0] != expectAddrs[0] {
		t.Errorf("addresses of the first agent mismatch. got: %v", res[0].Addresses)
	}
	if res[1].ID != withoutAddr || res[1].Addresses == nil || len(res[1].Addresses) != 0 {
		t.Errorf("expected an empty non-nil address slice for the second agent, got: %#v", res[1].Addresses)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_AgentGetByCustomerIDAndAddress_paths(t *testing.T) {
	customerID := testSeqID(100)
	agentID := testSeqID(101)
	address := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550104"}

	t.Run("owner lookup failure is wrapped", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectQuery("FROM agent_addresses").WillReturnError(errTestBoom)

		_, err := h.AgentGetByCustomerIDAndAddress(context.Background(), customerID, address)
		if err == nil || err == ErrNotFound || errors.Is(err, ErrNotFound) {
			t.Fatalf("expected a wrapped lookup failure, got: %v", err)
		}
		if !errors.Is(err, errTestBoom) || !strings.Contains(err.Error(), "could not lookup owner") {
			t.Errorf("expected the wrapped cause, got: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("no owner is ErrNotFound as is", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectQuery("FROM agent_addresses").WillReturnRows(sqlmock.NewRows([]string{"agent_id"}))

		if _, err := h.AgentGetByCustomerIDAndAddress(context.Background(), customerID, address); err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("agent load failure is returned", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)
		mock.ExpectQuery("FROM agent_addresses").WillReturnRows(sqlmock.NewRows([]string{"agent_id"}).AddRow(agentID.Bytes()))
		mockCache.EXPECT().AgentGet(gomock.Any(), agentID).Return(nil, errTestBoom)
		mock.ExpectQuery("FROM agent_agents").WillReturnError(errTestBoom)

		res, err := h.AgentGetByCustomerIDAndAddress(context.Background(), customerID, address)
		if !errors.Is(err, errTestBoom) || res != nil {
			t.Errorf("expected the load failure and no agent, got: %v, %v", res, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("found", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)
		mock.ExpectQuery("FROM agent_addresses").WithArgs(customerID.Bytes(), "tel", "+15550104").
			WillReturnRows(sqlmock.NewRows([]string{"agent_id"}).AddRow(agentID.Bytes()))

		cached := &agent.Agent{}
		cached.ID = agentID
		mockCache.EXPECT().AgentGet(gomock.Any(), agentID).Return(cached, nil)

		res, err := h.AgentGetByCustomerIDAndAddress(context.Background(), customerID, address)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != cached {
			t.Errorf("expected the agent of the address owner")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}

func Test_AgentGetByUsername_paths(t *testing.T) {
	t.Run("list failure is propagated", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectQuery("FROM agent_agents").WithArgs("2026-01-01 00:00:00.000000", "alice").WillReturnError(errTestBoom)

		res, err := h.AgentGetByUsername(context.Background(), "alice")
		if !errors.Is(err, errTestBoom) || res != nil {
			t.Errorf("expected the list failure and no agent, got: %v, %v", res, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("no match is ErrNotFound", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectQuery("FROM agent_agents").WithArgs("2026-01-01 00:00:00.000000", "alice").WillReturnRows(sqlmock.NewRows(commondatabasehandler.GetDBFields(&agent.Agent{})))

		if _, err := h.AgentGetByUsername(context.Background(), "alice"); err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}

func Test_agentUpdate_errors(t *testing.T) {
	t.Run("fields that cannot be serialized", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)

		err := h.agentUpdate(context.Background(), testSeqID(110), map[agent.Field]any{agent.FieldName: []any{make(chan int)}})
		if err == nil || !strings.Contains(err.Error(), "prepare fields failed") {
			t.Errorf("expected a prepare fields failure, got: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("no query was expected: %v", err)
		}
	})

	t.Run("empty fields issue no query", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)

		if err := h.agentUpdate(context.Background(), testSeqID(110), map[agent.Field]any{}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("no query was expected: %v", err)
		}
	})
}

func Test_AgentSetAddresses_errors(t *testing.T) {
	id := testSeqID(120)
	customerID := testSeqID(121)
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550105"}}
	lockRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"customer_id"}).AddRow(customerID.Bytes())
	}

	tests := []struct {
		name   string
		expect func(mock sqlmock.Sqlmock)

		expectNotFound bool
		expectIs       error
		expectContains string
	}{
		{
			name: "agent row missing",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT customer_id").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}))
				mock.ExpectRollback()
			},
			expectNotFound: true,
		},
		{
			name: "lock query failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT customer_id").WillReturnError(errTestBoom)
				mock.ExpectRollback()
			},
			expectIs:       errTestBoom,
			expectContains: "could not lock agent row. AgentSetAddresses",
		},
		{
			name: "customer id cannot be parsed",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT customer_id").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow([]byte{1, 2}))
				mock.ExpectRollback()
			},
			expectContains: "could not parse customer_id. AgentSetAddresses",
		},
		{
			name: "address delete failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT customer_id").WillReturnRows(lockRows())
				mock.ExpectExec("DELETE FROM agent_addresses").WillReturnError(errTestBoom)
				mock.ExpectRollback()
			},
			expectIs:       errTestBoom,
			expectContains: "agentAddressDeleteByAgentID",
		},
		{
			name: "address insert failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT customer_id").WillReturnRows(lockRows())
				mock.ExpectExec("DELETE FROM agent_addresses").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("INSERT INTO agent_addresses").WillReturnError(errTestBoom)
				mock.ExpectRollback()
			},
			expectIs:       errTestBoom,
			expectContains: "agentAddressInsert",
		},
		{
			name: "agent update failure",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT customer_id").WillReturnRows(lockRows())
				mock.ExpectExec("DELETE FROM agent_addresses").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("INSERT INTO agent_addresses").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE agent_agents").WillReturnError(errTestBoom)
				mock.ExpectRollback()
			},
			expectIs:       errTestBoom,
			expectContains: "could not execute update. AgentSetAddresses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no cache expectation is registered: a rolled back change must not touch the cache
			h, mock, _ := newSQLMockHandler(t)
			tt.expect(mock)

			err := h.AgentSetAddresses(context.Background(), id, addrs)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if tt.expectNotFound {
				if err != ErrNotFound {
					t.Errorf("expected ErrNotFound, got: %v", err)
				}
			} else if errors.Is(err, ErrNotFound) {
				t.Errorf("a failure must not be reported as ErrNotFound. err: %v", err)
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("expected errors.Is(%v), got: %v", tt.expectIs, err)
			}
			if tt.expectContains != "" && !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("expected the error to contain %q, got: %v", tt.expectContains, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// Test_cacheRefreshFailure_notPropagated verifies the behavior when the
// refresh query succeeds but only the cache write fails: the committed change
// is reported as a success, and AgentSetAddresses additionally invalidates the
// cache entry.
func Test_cacheRefreshFailure_notPropagated(t *testing.T) {
	id := testSeqID(130)
	customerID := testSeqID(131)

	t.Run("AgentUpdate", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)
		mock.ExpectExec("UPDATE agent_agents").WillReturnResult(sqlmock.NewResult(0, 1))
		testExpectRefresh(mock, mockCache, id)

		if err := h.AgentUpdate(context.Background(), id, map[agent.Field]any{agent.FieldName: "name"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("AgentCreate", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO agent_agents").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		testExpectRefresh(mock, mockCache, id)

		a := &agent.Agent{}
		a.ID = id
		a.CustomerID = customerID
		if err := h.AgentCreate(context.Background(), a); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("AgentReserve", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)
		mock.ExpectExec("UPDATE agent_agents").WillReturnResult(sqlmock.NewResult(0, 1))
		testExpectRefresh(mock, mockCache, id)

		ok, err := h.AgentReserve(context.Background(), id, "queuecall", testSeqID(132))
		if err != nil || !ok {
			t.Errorf("expected a successful reservation, got: %v, %v", ok, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("AgentSetAddresses invalidates the cache entry", func(t *testing.T) {
		h, mock, mockCache := newSQLMockHandler(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT customer_id").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow(customerID.Bytes()))
		mock.ExpectExec("DELETE FROM agent_addresses").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("UPDATE agent_agents").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		testExpectRefresh(mock, mockCache, id)
		mockCache.EXPECT().AgentDel(gomock.Any(), id).Return(nil)

		if err := h.AgentSetAddresses(context.Background(), id, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}

func Test_NewHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	res := NewHandler(db, mockCache)
	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("expected a *handler, got: %T", res)
	}
	if h.db != db {
		t.Errorf("expected the injected database to be used")
	}
	if h.cache != mockCache {
		t.Errorf("expected the injected cache to be used")
	}
	if h.utilHandler == nil {
		t.Errorf("expected the util handler to be set")
	}
}
