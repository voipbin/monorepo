package listenhandler

import (
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-message-manager/pkg/messagehandler"

	gomock "go.uber.org/mock/gomock"
)

func TestNewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	h := NewListenHandler(mockSock, mockMessage)

	if h == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestSimpleResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"status_200", 200},
		{"status_400", 400},
		{"status_404", 404},
		{"status_500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := simpleResponse(tt.statusCode)

			if res == nil {
				t.Error("Expected non-nil response")
				return
			}

			if res.StatusCode != tt.statusCode {
				t.Errorf("StatusCode mismatch: got %d, want %d", res.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name          string
		queue         string
		exchangeDelay string
		queueError    error
		expectError   bool
	}{
		{
			name:          "successful_run",
			queue:         "test-queue",
			exchangeDelay: "test-exchange",
			queueError:    nil,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				messageHandler: mockMessage,
			}

			mockSock.EXPECT().QueueCreate(tt.queue, "normal").Return(tt.queueError)
			// The ConsumeRPC call happens in a goroutine, so we need to expect it
			// We use AnyTimes() because the goroutine may or may not run before test completes
			mockSock.EXPECT().ConsumeRPC(gomock.Any(), tt.queue, constCosumerName, false, false, false, 10, gomock.Any()).
				Return(nil).AnyTimes()

			err := h.Run(tt.queue, tt.exchangeDelay)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Allow goroutine to start before finishing
			time.Sleep(10 * time.Millisecond)
			mc.Finish()
		})
	}
}

func TestProcessRequestNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	h := &listenHandler{
		sockHandler:    mockSock,
		messageHandler: mockMessage,
	}

	req := &sock.Request{
		Method: sock.RequestMethodGet,
		URI:    "/unknown/path",
		Data:   []byte(""),
	}

	res, err := h.processRequest(req)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if res == nil {
		t.Error("Expected non-nil response")
		return
	}

	if res.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", res.StatusCode)
	}
}

func TestRegexPatterns(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		regexMatch  func(string) bool
		shouldMatch bool
	}{
		{
			name:        "messages_get_with_query",
			uri:         "/v1/messages?page_size=10",
			regexMatch:  regV1MessagesGet.MatchString,
			shouldMatch: true,
		},
		{
			name:        "messages_post",
			uri:         "/v1/messages",
			regexMatch:  regV1Messages.MatchString,
			shouldMatch: true,
		},
		{
			name:        "messages_id",
			uri:         "/v1/messages/123e4567-e89b-12d3-a456-426614174000",
			regexMatch:  regV1MessagesID.MatchString,
			shouldMatch: true,
		},
		{
			name:        "hooks_post",
			uri:         "/v1/hooks",
			regexMatch:  regV1Hooks.MatchString,
			shouldMatch: true,
		},
		{
			name:        "invalid_messages_id",
			uri:         "/v1/messages/invalid-uuid",
			regexMatch:  regV1MessagesID.MatchString,
			shouldMatch: false,
		},
		{
			name:        "messages_get_no_query",
			uri:         "/v1/messages",
			regexMatch:  regV1MessagesGet.MatchString,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := tt.regexMatch(tt.uri)

			if matched != tt.shouldMatch {
				t.Errorf("Regex match mismatch for URI %s: got %v, want %v", tt.uri, matched, tt.shouldMatch)
			}
		})
	}
}
