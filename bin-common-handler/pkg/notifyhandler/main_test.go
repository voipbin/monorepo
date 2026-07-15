package notifyhandler

import (
	"encoding/json"
	"os"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/sockhandler"
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

func TestNewNotifyHandlerForExistingExchange_SkipsDeclare(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	// TopicCreate/TopicCreateWithKind must NOT be called -- the exchange is assumed
	// already declared by the caller.
	mockSock.EXPECT().TopicCreate(gomock.Any()).Times(0)
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Times(0)

	h := NewNotifyHandlerForExistingExchange(mockSock, nil, "some.exchange.name", "test-service")

	if h == nil {
		t.Fatal("Expected non-nil NotifyHandler")
	}
}
