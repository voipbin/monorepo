package dbhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
)

func TestPeerEventBatchInsert_NoConnection(t *testing.T) {
	handler := &dbHandler{
		address:  "localhost:9000",
		database: "test",
		conn:     nil, // No connection established
	}

	ctx := context.Background()
	rows := []PeerEventRow{
		{
			CustomerID:  uuid.Must(uuid.NewV4()),
			Publisher:   "call",
			EventType:   "call_hangup",
			ReferenceID: uuid.Must(uuid.NewV4()),
		},
	}

	err := handler.PeerEventBatchInsert(ctx, rows)
	if err == nil {
		t.Error("PeerEventBatchInsert() expected error when conn is nil, got nil")
	}
	if err.Error() != "clickhouse connection not established" {
		t.Errorf("PeerEventBatchInsert() error = %q, want %q", err.Error(), "clickhouse connection not established")
	}
}

func TestPeerEventBatchInsert_EmptyBatch(t *testing.T) {
	// Empty rows is a no-op ONLY when a connection exists (mirrors
	// EventBatchInsert's ordering: the nil-connection check runs first, so an
	// empty batch against a nil connection still returns the connection
	// error, not a silent nil). This test cannot exercise the true empty-
	// batch-with-connection no-op path without a live ClickHouse connection;
	// it instead documents and locks in the checked ordering by asserting the
	// nil-connection error takes precedence even when rows is empty.
	handler := &dbHandler{
		address:  "localhost:9000",
		database: "test",
		conn:     nil,
	}

	ctx := context.Background()
	err := handler.PeerEventBatchInsert(ctx, []PeerEventRow{})
	if err == nil {
		t.Error("PeerEventBatchInsert() with nil conn expected error even for empty rows, got nil")
	}
	if err.Error() != "clickhouse connection not established" {
		t.Errorf("PeerEventBatchInsert() error = %q, want %q", err.Error(), "clickhouse connection not established")
	}
}
