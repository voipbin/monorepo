package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
)

func TestNewListenHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := NewListenHandler(mockSock, mockEvent)
	if handler == nil {
		t.Error("NewListenHandler() returned nil")
	}
}

func TestSimpleResponse(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{name: "200 OK", code: 200, expected: 200},
		{name: "400 Bad Request", code: 400, expected: 400},
		{name: "404 Not Found", code: 404, expected: 404},
		{name: "500 Internal Server Error", code: 500, expected: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := simpleResponse(tt.code)
			if resp.StatusCode != tt.expected {
				t.Errorf("simpleResponse(%d) = %d, want %d", tt.code, resp.StatusCode, tt.expected)
			}
		})
	}
}

func TestProcessRequest_V1EventsPost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  10,
	}
	reqData, _ := json.Marshal(req)

	ts := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	expectedResponse := &response.V1DataEventsPost{
		Result: []*event.Event{
			{Timestamp: ts, EventType: "activeflow_created"},
		},
	}

	mockEvent.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(expectedResponse, nil)

	sockReq := &sock.Request{
		URI:    "/v1/events",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.processRequest(sockReq)
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("processRequest() StatusCode = %d, want 200", resp.StatusCode)
	}

	if resp.DataType != "application/json" {
		t.Errorf("processRequest() DataType = %q, want %q", resp.DataType, "application/json")
	}
}

func TestProcessRequest_V1EventsPost_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	sockReq := &sock.Request{
		URI:    "/v1/events",
		Method: sock.RequestMethodPost,
		Data:   []byte("invalid json"),
	}

	resp, err := handler.processRequest(sockReq)
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("processRequest() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestProcessRequest_V1EventsPost_HandlerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	testID := uuid.Must(uuid.NewV4())
	req := &request.V1DataEventsPost{
		Publisher: commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:    []string{"activeflow_*"},
		PageSize:  10,
	}
	reqData, _ := json.Marshal(req)

	mockEvent.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("handler error"))

	sockReq := &sock.Request{
		URI:    "/v1/events",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.processRequest(sockReq)
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %d, want 500", resp.StatusCode)
	}
}

func TestProcessRequest_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	tests := []struct {
		name   string
		uri    string
		method sock.RequestMethod
	}{
		{name: "unknown URI", uri: "/v1/unknown", method: sock.RequestMethodPost},
		{name: "wrong method on events", uri: "/v1/events", method: sock.RequestMethodGet},
		{name: "wrong method DELETE", uri: "/v1/events", method: sock.RequestMethodDelete},
		{name: "empty URI", uri: "", method: sock.RequestMethodPost},
		{name: "root URI", uri: "/", method: sock.RequestMethodPost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sockReq := &sock.Request{
				URI:    tt.uri,
				Method: tt.method,
				Data:   []byte("{}"),
			}

			resp, err := handler.processRequest(sockReq)
			if err != nil {
				t.Fatalf("processRequest() error = %v", err)
			}

			if resp.StatusCode != 404 {
				t.Errorf("processRequest() StatusCode = %d, want 404", resp.StatusCode)
			}
		})
	}
}

func TestRun_QueueCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := NewListenHandler(mockSock, mockEvent)

	mockSock.EXPECT().
		QueueCreate("test-queue", "normal").
		Return(errors.New("queue create failed"))

	err := handler.Run("test-queue")
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}
}

func TestRun_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := NewListenHandler(mockSock, mockEvent)

	mockSock.EXPECT().
		QueueCreate("test-queue", "normal").
		Return(nil)

	// Note: ConsumeRPC is called in a goroutine, so we use AnyTimes() to avoid
	// test failures due to timing. The main path we test is that Run() returns nil
	// when QueueCreate succeeds.
	mockSock.EXPECT().
		ConsumeRPC(gomock.Any(), "test-queue", "timeline-manager", false, false, false, 10, gomock.Any()).
		AnyTimes().
		Return(nil)

	err := handler.Run("test-queue")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRegexPatterns(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		matches bool
	}{
		{name: "exact match", uri: "/v1/events", matches: true},
		{name: "with query params should not match", uri: "/v1/events?page=1", matches: false},
		{name: "with trailing slash should not match", uri: "/v1/events/", matches: false},
		{name: "with extra path should not match", uri: "/v1/events/123", matches: false},
		{name: "different path", uri: "/v1/other", matches: false},
		{name: "v2 version", uri: "/v2/events", matches: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := regV1Events.MatchString(tt.uri)
			if result != tt.matches {
				t.Errorf("regV1Events.MatchString(%q) = %v, want %v", tt.uri, result, tt.matches)
			}
		})
	}
}

func TestV1EventsPost_EmptyData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	sockReq := &sock.Request{
		URI:    "/v1/events",
		Method: sock.RequestMethodPost,
		Data:   []byte(""),
	}

	resp, err := handler.v1EventsPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1EventsPost() error = %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("v1EventsPost() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestV1EventsPost_NilData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)

	handler := &listenHandler{
		sockHandler:  mockSock,
		eventHandler: mockEvent,
	}

	sockReq := &sock.Request{
		URI:    "/v1/events",
		Method: sock.RequestMethodPost,
		Data:   nil,
	}

	resp, err := handler.v1EventsPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1EventsPost() error = %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("v1EventsPost() StatusCode = %d, want 400", resp.StatusCode)
	}
}
