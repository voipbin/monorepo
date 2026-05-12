package main

import (
	"encoding/json"
	"testing"
)

func TestBuildRequest_basic(t *testing.T) {
	req, err := buildRequest("/v1/numbers/renew", "POST", "application/json", `{"days":28}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.URI != "/v1/numbers/renew" {
		t.Errorf("URI: got %q, want %q", req.URI, "/v1/numbers/renew")
	}
	if req.Method != "POST" {
		t.Errorf("Method: got %q, want %q", req.Method, "POST")
	}
	if req.Publisher != "bin-trigger-sender" {
		t.Errorf("Publisher: got %q, want %q", req.Publisher, "bin-trigger-sender")
	}
	var data map[string]int
	if err := json.Unmarshal(req.Data, &data); err != nil {
		t.Fatalf("Data unmarshal: %v", err)
	}
	if data["days"] != 28 {
		t.Errorf("data.days: got %d, want 28", data["days"])
	}
}

func TestBuildRequest_emptyData(t *testing.T) {
	req, err := buildRequest("/v1/numbers/renew", "POST", "application/json", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Data != nil {
		t.Errorf("expected nil Data for empty input, got %s", req.Data)
	}
}

func TestBuildRequest_marshalsToJSON(t *testing.T) {
	req, err := buildRequest("/v1/numbers/renew", "POST", "application/json", `{"days":28}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["publisher"] != "bin-trigger-sender" {
		t.Errorf("publisher: got %v", m["publisher"])
	}
	if m["request_id"] != "bin-trigger-sender-cronjob" {
		t.Errorf("request_id: got %v", m["request_id"])
	}
}

func TestBuildRequest_setsRequestID(t *testing.T) {
	req, _ := buildRequest("/v1/numbers/renew", "POST", "application/json", "")
	if req.RequestID != "bin-trigger-sender-cronjob" {
		t.Errorf("RequestID: got %q, want %q", req.RequestID, "bin-trigger-sender-cronjob")
	}
}

// TestRunDialFailure verifies run returns an error when RabbitMQ is unreachable.
// Note: the ctx.Done() timeout branch inside run requires a live broker to exercise.
// That path is not unit-tested here; it is covered by integration testing against a real broker.
func TestRunDialFailure(t *testing.T) {
	err := run("amqp://127.0.0.1:1", "queue", "/v1/test", "POST", "application/json", "", 5000)
	if err == nil {
		t.Fatal("expected error for unreachable broker, got nil")
	}
}
