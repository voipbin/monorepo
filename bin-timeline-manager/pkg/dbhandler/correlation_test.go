package dbhandler

import (
	"context"
	"testing"
)

func TestResourceActiveflowIDGet_NoConnection(t *testing.T) {
	handler := &dbHandler{address: "localhost:9000", database: "test", conn: nil}

	_, err := handler.ResourceActiveflowIDGet(context.Background(), "res-1")
	if err == nil {
		t.Fatal("expected error when conn is nil")
	}
	if err.Error() != "clickhouse connection not established" {
		t.Errorf("error = %q, want %q", err.Error(), "clickhouse connection not established")
	}
}

func TestResourceExists_NoConnection(t *testing.T) {
	handler := &dbHandler{address: "localhost:9000", database: "test", conn: nil}

	_, err := handler.ResourceExists(context.Background(), "res-1")
	if err == nil {
		t.Fatal("expected error when conn is nil")
	}
	if err.Error() != "clickhouse connection not established" {
		t.Errorf("error = %q, want %q", err.Error(), "clickhouse connection not established")
	}
}

func TestCorrelatedResourceList_NoConnection(t *testing.T) {
	handler := &dbHandler{address: "localhost:9000", database: "test", conn: nil}

	_, err := handler.CorrelatedResourceList(context.Background(), "af-1", 101)
	if err == nil {
		t.Fatal("expected error when conn is nil")
	}
	if err.Error() != "clickhouse connection not established" {
		t.Errorf("error = %q, want %q", err.Error(), "clickhouse connection not established")
	}
}
