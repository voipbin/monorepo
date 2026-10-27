package dbhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

var (
	errTestPrepare = errors.New("injected prepare failure")
	errTestAppend  = errors.New("injected append failure")
	errTestSend    = errors.New("injected send failure")
)

func Test_EventBatchInsert_NoConnection(t *testing.T) {
	h := &dbHandler{}

	err := h.EventBatchInsert(context.Background(), []EventRow{{EventType: "call_created"}})
	if err == nil || err.Error() != "clickhouse connection not established" {
		t.Errorf("error = %v, want connection error", err)
	}
}

func Test_EventBatchInsert(t *testing.T) {
	ts1 := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	ts2 := time.Date(2026, 3, 4, 5, 6, 8, 0, time.UTC)
	row1 := EventRow{Timestamp: ts1, EventType: "call_created", Publisher: "call-manager", DataType: "call", Data: `{"a":1}`}
	row2 := EventRow{Timestamp: ts2, EventType: "flow_created", Publisher: "flow-manager", DataType: "flow", Data: `{"b":2}`}

	tests := []struct {
		name string

		rows       []EventRow
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
			rows:             []EventRow{},
			wantPrepareCalls: 0,
		},
		{
			name:             "prepare error",
			rows:             []EventRow{row1},
			prepareErr:       errTestPrepare,
			wantPrepareCalls: 1,
			wantErrMsg:       "could not prepare ClickHouse batch",
			wantCause:        errTestPrepare,
		},
		{
			name:             "append error on the first row",
			rows:             []EventRow{row1, row2},
			appendErr:        errTestAppend,
			appendAt:         1,
			wantPrepareCalls: 1,
			wantAppended:     [][]any{{ts1, "call_created", "call-manager", "call", `{"a":1}`}},
			wantErrMsg:       "could not append to ClickHouse batch",
			wantCause:        errTestAppend,
		},
		{
			name:             "append error on the second row",
			rows:             []EventRow{row1, row2},
			appendErr:        errTestAppend,
			appendAt:         2,
			wantPrepareCalls: 1,
			wantAppended: [][]any{
				{ts1, "call_created", "call-manager", "call", `{"a":1}`},
				{ts2, "flow_created", "flow-manager", "flow", `{"b":2}`},
			},
			wantErrMsg: "could not append to ClickHouse batch",
			wantCause:  errTestAppend,
		},
		{
			name:             "send error",
			rows:             []EventRow{row1},
			sendErr:          errTestSend,
			wantPrepareCalls: 1,
			wantAppended:     [][]any{{ts1, "call_created", "call-manager", "call", `{"a":1}`}},
			wantSendCalls:    1,
			wantErrMsg:       "could not send ClickHouse batch",
			wantCause:        errTestSend,
		},
		{
			name:             "one row",
			rows:             []EventRow{row2},
			wantPrepareCalls: 1,
			wantAppended:     [][]any{{ts2, "flow_created", "flow-manager", "flow", `{"b":2}`}},
			wantSendCalls:    1,
		},
		{
			name:             "two rows are appended in order and sent once",
			rows:             []EventRow{row1, row2},
			wantPrepareCalls: 1,
			wantAppended: [][]any{
				{ts1, "call_created", "call-manager", "call", `{"a":1}`},
				{ts2, "flow_created", "flow-manager", "flow", `{"b":2}`},
			},
			wantSendCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &fakeBatch{appendErr: tt.appendErr, appendErrAt: tt.appendAt, sendErr: tt.sendErr}
			conn := &fakeConn{batch: batch, prepareErr: tt.prepareErr}
			h := &dbHandler{conn: conn}

			err := h.EventBatchInsert(context.Background(), tt.rows)

			if tt.wantErrMsg != "" {
				assertWrapped(t, err, tt.wantCause, tt.wantErrMsg)
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if conn.prepareCalls != tt.wantPrepareCalls {
				t.Errorf("PrepareBatch called %d times, want %d", conn.prepareCalls, tt.wantPrepareCalls)
			}
			if tt.wantPrepareCalls > 0 {
				wantSQL := "INSERT INTO events (timestamp, event_type, publisher, data_type, data)"
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
