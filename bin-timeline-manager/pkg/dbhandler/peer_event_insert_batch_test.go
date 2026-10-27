package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
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
