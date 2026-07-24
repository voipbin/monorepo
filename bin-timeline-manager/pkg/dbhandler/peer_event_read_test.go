package dbhandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
)

func TestBuildPeerEventQuery_SinglePair(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	pairs := []PeerPairFilter{{PeerType: "tel", PeerTarget: "+15551234567"}}

	query, args := buildPeerEventQuery(testID, pairs, "", 10)

	if !strings.Contains(query, "FROM peer_events") {
		t.Error("Query missing FROM clause")
	}
	if !strings.Contains(query, "WHERE customer_id = ?") {
		t.Error("Query missing customer_id condition")
	}
	if !strings.Contains(query, "(peer_type = ? AND peer_target = ?)") {
		t.Error("Query missing single-pair condition")
	}
	if strings.Contains(query, " OR ") {
		t.Error("Query should not contain OR for a single pair")
	}
	if !strings.Contains(query, "ORDER BY timestamp DESC LIMIT ?") {
		t.Error("Query missing ORDER BY and LIMIT clause")
	}

	// args: customerID, peerType, peerTarget, pageSize
	if len(args) != 4 {
		t.Fatalf("Expected 4 args, got %d", len(args))
	}
	if args[0] != testID.String() {
		t.Errorf("First arg should be customerID, got %v", args[0])
	}
	if args[1] != "tel" || args[2] != "+15551234567" {
		t.Errorf("Expected peer pair args, got %v %v", args[1], args[2])
	}
	if args[3] != 10 {
		t.Errorf("Last arg should be pageSize, got %v", args[3])
	}
}

func TestBuildPeerEventQuery_MultiPairORExpansion(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	pairs := []PeerPairFilter{
		{PeerType: "tel", PeerTarget: "+15551234567"},
		{PeerType: "email", PeerTarget: "test@example.com"},
	}

	query, args := buildPeerEventQuery(testID, pairs, "", 10)

	if !strings.Contains(query, "(peer_type = ? AND peer_target = ?) OR (peer_type = ? AND peer_target = ?)") {
		t.Error("Query missing OR-expanded multi-pair condition")
	}

	// args: customerID, (peerType, peerTarget) x2, pageSize
	if len(args) != 6 {
		t.Fatalf("Expected 6 args, got %d", len(args))
	}
}

func TestBuildPeerEventQuery_WithPageToken(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	pairs := []PeerPairFilter{{PeerType: "tel", PeerTarget: "+15551234567"}}
	pageToken := "2026-01-15T10:29:00.123000Z"

	query, args := buildPeerEventQuery(testID, pairs, pageToken, 10)

	if !strings.Contains(query, "AND timestamp < ?") {
		t.Error("Query missing pagination condition")
	}

	// args: customerID, peerType, peerTarget, pageToken, pageSize
	if len(args) != 5 {
		t.Fatalf("Expected 5 args, got %d", len(args))
	}
	if args[3] != pageToken {
		t.Errorf("Expected pageToken arg, got %v", args[3])
	}
}

func TestBuildPeerEventQuery_NoPairs(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())

	query, args := buildPeerEventQuery(testID, nil, "", 10)

	if strings.Contains(query, "peer_type = ?") || strings.Contains(query, "peer_type = ? AND") {
		t.Error("Query should not have a peer_type filter clause when no pairs are given")
	}
	// args: customerID, pageSize only (no peer pair args appended)
	if len(args) != 2 {
		t.Fatalf("Expected 2 args, got %d", len(args))
	}
}

func TestPeerEventList_NoConnection(t *testing.T) {
	handler := &dbHandler{
		address:  "localhost:9000",
		database: "test",
		conn:     nil,
	}

	ctx := context.Background()
	testID := uuid.Must(uuid.NewV4())

	_, err := handler.PeerEventList(ctx, testID, []PeerPairFilter{{PeerType: "tel", PeerTarget: "+15551234567"}}, "", 10)
	if err == nil {
		t.Error("PeerEventList() expected error when conn is nil, got nil")
	}
	if !strings.Contains(err.Error(), "clickhouse connection not established") {
		t.Errorf("PeerEventList() error = %q, want to contain %q", err.Error(), "clickhouse connection not established")
	}
}
