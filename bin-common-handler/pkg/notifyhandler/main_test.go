package notifyhandler

import (
	"encoding/json"
	"os"
	"testing"
)

const (
	testPublisher = "test"
)

type testEvent struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

func (h *testEvent) CreateWebhookEvent() ([]byte, error) {
	m, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func TestMain(m *testing.M) {
	initPrometheus("test")

	os.Exit(m.Run())
}
