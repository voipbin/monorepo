package dbhandler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-timeline-manager/models/peerevent"
)

func Test_PeerEventBatchInsert(t *testing.T) {
	ts1 := time.Date(2026, 4, 5, 6, 7, 8, 0, time.UTC)
	ts2 := time.Date(2026, 4, 5, 6, 7, 9, 0, time.UTC)
	cust1 := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")
	cust2 := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002")
	ref1 := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001")
	ref2 := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002")

	row1 := PeerEventRow{
		Timestamp: ts1, CustomerID: cust1, Publisher: "call", EventType: "call_created", ReferenceID: ref1,
		Direction: "incoming", Peer: `{"p":1}`, Local: `{"l":1}`, PeerType: "tel", PeerTarget: "+1000001",
		LocalType: "sip", LocalTarget: "ext-1", Data: `{"d":1}`,
	}
	row2 := PeerEventRow{
		Timestamp: ts2, CustomerID: cust2, Publisher: "conversation", EventType: "conversation_created", ReferenceID: ref2,
		Direction: "outgoing", Peer: `{"p":2}`, Local: `{"l":2}`, PeerType: "email", PeerTarget: "a@example.com",
		LocalType: "line", LocalTarget: "line-2", Data: `{"d":2}`,
	}
	args1 := []any{ts1, cust1, "call", "call_created", ref1, "incoming", `{"p":1}`, `{"l":1}`, "tel", "+1000001", "sip", "ext-1", `{"d":1}`}
	args2 := []any{ts2, cust2, "conversation", "conversation_created", ref2, "outgoing", `{"p":2}`, `{"l":2}`, "email", "a@example.com", "line", "line-2", `{"d":2}`}

	tests := []struct {
		name string

		rows       []PeerEventRow
		prepareErr error
		appendErr  error
		appendAt   int
		sendErr    error

		wantPrepareCalls int
		wantAppended     [][]any
		wantSendCalls    int
		wantErrMsg       string
		wantCause        error
	}{
		{
			name:             "empty rows is a no-op",
			rows:             []PeerEventRow{},
			wantPrepareCalls: 0,
		},
		{
			name:             "prepare error",
			rows:             []PeerEventRow{row1},
			prepareErr:       errTestPrepare,
			wantPrepareCalls: 1,
			wantErrMsg:       "could not prepare ClickHouse batch",
			wantCause:        errTestPrepare,
		},
		{
			name:             "append error on the first row",
			rows:             []PeerEventRow{row1, row2},
			appendErr:        errTestAppend,
			appendAt:         1,
			wantPrepareCalls: 1,
			wantAppended:     [][]any{args1},
			wantErrMsg:       "could not append to ClickHouse batch",
			wantCause:        errTestAppend,
		},
		{
			name:             "append error on the second row",
			rows:             []PeerEventRow{row1, row2},
			appendErr:        errTestAppend,
			appendAt:         2,
			wantPrepareCalls: 1,
			wantAppended:     [][]any{args1, args2},
			wantErrMsg:       "could not append to ClickHouse batch",
			wantCause:        errTestAppend,
		},
		{
			name:             "send error",
			rows:             []PeerEventRow{row1},
			sendErr:          errTestSend,
			wantPrepareCalls: 1,
			wantAppended:     [][]any{args1},
			wantSendCalls:    1,
			wantErrMsg:       "could not send ClickHouse batch",
			wantCause:        errTestSend,
		},
		{
			name:             "one row",
			rows:             []PeerEventRow{row2},
			wantPrepareCalls: 1,
			wantAppended:     [][]any{args2},
			wantSendCalls:    1,
		},
		{
			name:             "two rows are appended in order and sent once",
			rows:             []PeerEventRow{row1, row2},
			wantPrepareCalls: 1,
			wantAppended:     [][]any{args1, args2},
			wantSendCalls:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &fakeBatch{appendErr: tt.appendErr, appendErrAt: tt.appendAt, sendErr: tt.sendErr}
			conn := &fakeConn{batch: batch, prepareErr: tt.prepareErr}
			h := &dbHandler{conn: conn}

			err := h.PeerEventBatchInsert(context.Background(), tt.rows)

			if tt.wantErrMsg != "" {
				assertWrapped(t, err, tt.wantCause, tt.wantErrMsg)
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if conn.prepareCalls != tt.wantPrepareCalls {
				t.Errorf("PrepareBatch called %d times, want %d", conn.prepareCalls, tt.wantPrepareCalls)
			}
			if tt.wantPrepareCalls > 0 {
				wantSQL := "INSERT INTO peer_events (timestamp, customer_id, publisher, event_type, reference_id, direction, " +
					"peer, local, peer_type, peer_target, local_type, local_target, data)"
				if normSQL(conn.prepareSQL) != wantSQL {
					t.Errorf("prepare SQL = %q, want %q", normSQL(conn.prepareSQL), wantSQL)
				}
			}
			if !reflect.DeepEqual(batch.appended, tt.wantAppended) {
				t.Errorf("appended = %v, want %v", batch.appended, tt.wantAppended)
			}
			if batch.sendCalls != tt.wantSendCalls {
				t.Errorf("Send called %d times, want %d", batch.sendCalls, tt.wantSendCalls)
			}
		})
	}
}

func Test_PeerEventList(t *testing.T) {
	ts1 := time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)
	ts2 := time.Date(2026, 5, 6, 7, 8, 8, 0, time.UTC)
	custID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000009")
	ref1 := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000011")
	ref2 := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000012")

	baseSQL := "SELECT timestamp, customer_id, publisher, event_type, reference_id, direction, peer, local, data " +
		"FROM peer_events WHERE customer_id = ?"
	tailSQL := " ORDER BY timestamp DESC LIMIT ?"

	okPeer := `{"type":"tel","target":"+1000001"}`
	okLocal := `{"type":"sip","target":"ext-1"}`
	// The numeric target cannot be decoded into a string. The decoder fills
	// "type" first and then reports the error, so the reset is observable.
	badPeer := `{"type":"tel","target":123}`
	badLocal := `{"type":"sip","target":456}`

	tests := []struct {
		name string

		addrs     []commonaddress.Address
		pageToken string
		pageSize  int

		queryErr error
		rows     *fakeRows

		wantSQL    string
		wantArgs   []any
		wantErrMsg string
		wantCause  error
		wantClosed int
		wantResult []*peerevent.PeerEvent
	}{
		{
			name:       "query error",
			pageSize:   4,
			queryErr:   errTestQuery,
			wantSQL:    baseSQL + tailSQL,
			wantArgs:   []any{custID.String(), 4},
			wantErrMsg: "could not query peer_events",
			wantCause:  errTestQuery,
		},
		{
			name:     "scan error",
			pageSize: 4,
			rows: &fakeRows{
				data:      [][]any{{ts1, custID.String(), "call", "call_created", ref1.String(), "incoming", okPeer, okLocal, `{}`}},
				scanErr:   errTestScan,
				scanErrAt: 1,
			},
			wantSQL:    baseSQL + tailSQL,
			wantArgs:   []any{custID.String(), 4},
			wantErrMsg: "could not scan peer_event row",
			wantCause:  errTestScan,
			wantClosed: 1,
		},
		{
			name:     "iteration error after one row",
			pageSize: 4,
			rows: &fakeRows{
				data: [][]any{{ts1, custID.String(), "call", "call_created", ref1.String(), "incoming", okPeer, okLocal, `{}`}},
				err:  errTestIter,
			},
			wantSQL:    baseSQL + tailSQL,
			wantArgs:   []any{custID.String(), 4},
			wantErrMsg: "error iterating rows",
			wantCause:  errTestIter,
			wantClosed: 1,
		},
		{
			name:       "no rows gives a non-nil empty slice",
			pageSize:   4,
			rows:       &fakeRows{},
			wantSQL:    baseSQL + tailSQL,
			wantArgs:   []any{custID.String(), 4},
			wantClosed: 1,
			wantResult: []*peerevent.PeerEvent{},
		},
		{
			name: "addresses and page token with mapped rows",
			addrs: []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+1000001"},
				{Type: commonaddress.TypeEmail, Target: "a@example.com"},
			},
			pageToken: "2026-05-06T07:08:10Z",
			pageSize:  2,
			rows: &fakeRows{
				data: [][]any{
					{ts1, custID.String(), "call", "call_created", ref1.String(), "incoming", okPeer, okLocal, `{"d":1}`},
					{ts2, custID.String(), "conversation", "conversation_created", ref2.String(), "outgoing",
						`{"type":"email","target":"a@example.com"}`, `{"type":"line","target":"line-2"}`, `{"d":2}`},
				},
			},
			wantSQL: baseSQL + " AND ((peer_type = ? AND peer_target = ?) OR (peer_type = ? AND peer_target = ?))" +
				" AND timestamp < ?" + tailSQL,
			wantArgs:   []any{custID.String(), "tel", "+1000001", "email", "a@example.com", "2026-05-06T07:08:10Z", 2},
			wantClosed: 1,
			wantResult: []*peerevent.PeerEvent{
				{
					Timestamp: ts1, CustomerID: custID, Publisher: "call", EventType: "call_created", ReferenceID: ref1,
					Direction: "incoming", Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1000001"},
					Local: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "ext-1"}, Data: json.RawMessage(`{"d":1}`),
				},
				{
					Timestamp: ts2, CustomerID: custID, Publisher: "conversation", EventType: "conversation_created", ReferenceID: ref2,
					Direction: "outgoing", Peer: commonaddress.Address{Type: commonaddress.TypeEmail, Target: "a@example.com"},
					Local: commonaddress.Address{Type: commonaddress.TypeLine, Target: "line-2"}, Data: json.RawMessage(`{"d":2}`),
				},
			},
		},
		{
			name:     "a peer that cannot be decoded resets only the peer",
			pageSize: 4,
			rows: &fakeRows{
				data: [][]any{{ts1, custID.String(), "call", "call_created", ref1.String(), "incoming", badPeer, okLocal, `{}`}},
			},
			wantSQL:    baseSQL + tailSQL,
			wantArgs:   []any{custID.String(), 4},
			wantClosed: 1,
			wantResult: []*peerevent.PeerEvent{
				{
					Timestamp: ts1, CustomerID: custID, Publisher: "call", EventType: "call_created", ReferenceID: ref1,
					Direction: "incoming", Local: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "ext-1"},
					Data: json.RawMessage(`{}`),
				},
			},
		},
		{
			name:     "a local that cannot be decoded resets only the local",
			pageSize: 4,
			rows: &fakeRows{
				data: [][]any{{ts1, custID.String(), "call", "call_created", ref1.String(), "incoming", okPeer, badLocal, `{}`}},
			},
			wantSQL:    baseSQL + tailSQL,
			wantArgs:   []any{custID.String(), 4},
			wantClosed: 1,
			wantResult: []*peerevent.PeerEvent{
				{
					Timestamp: ts1, CustomerID: custID, Publisher: "call", EventType: "call_created", ReferenceID: ref1,
					Direction: "incoming", Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1000001"},
					Data: json.RawMessage(`{}`),
				},
			},
		},
		{
			name:     "invalid uuid strings become the nil uuid and the row is kept",
			pageSize: 4,
			rows: &fakeRows{
				data: [][]any{{ts1, "not-a-uuid", "call", "call_created", "also-not-a-uuid", "incoming", okPeer, okLocal, `{}`}},
			},
			wantSQL:    baseSQL + tailSQL,
			wantArgs:   []any{custID.String(), 4},
			wantClosed: 1,
			wantResult: []*peerevent.PeerEvent{
				{
					Timestamp: ts1, Publisher: "call", EventType: "call_created",
					Direction: "incoming", Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1000001"},
					Local: commonaddress.Address{Type: commonaddress.TypeSIP, Target: "ext-1"}, Data: json.RawMessage(`{}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakeConn{queryErr: tt.queryErr}
			if tt.rows != nil {
				tt.rows.t = t
				conn.rows = tt.rows
			}
			h := &dbHandler{conn: conn}

			ctx := context.WithValue(context.Background(), ctxKeyTest{}, "marker")
			got, err := h.PeerEventList(ctx, custID, tt.addrs, tt.pageToken, tt.pageSize)

			if tt.wantErrMsg != "" {
				assertWrapped(t, err, tt.wantCause, tt.wantErrMsg)
				if got != nil {
					t.Errorf("result = %v, want nil on error", got)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Fatal("result is nil, want a non-nil slice")
				}
				if !reflect.DeepEqual(got, tt.wantResult) {
					t.Errorf("result = %+v, want %+v", got, tt.wantResult)
				}
			}
			if tt.rows != nil && tt.rows.closed != tt.wantClosed {
				t.Errorf("rows closed %d times, want %d", tt.rows.closed, tt.wantClosed)
			}
			if normSQL(conn.querySQL) != tt.wantSQL {
				t.Errorf("query = %q, want %q", normSQL(conn.querySQL), tt.wantSQL)
			}
			if !reflect.DeepEqual(conn.queryArgs, tt.wantArgs) {
				t.Errorf("args = %v, want %v", conn.queryArgs, tt.wantArgs)
			}
			if conn.queryCtx != ctx {
				t.Error("query did not receive the caller's context")
			}
		})
	}
}

func Test_WaitForConnection_Connected(t *testing.T) {
	h := &dbHandler{conn: &fakeConn{}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.WaitForConnection(ctx); err != nil {
		t.Errorf("WaitForConnection() = %v, want nil", err)
	}
}
