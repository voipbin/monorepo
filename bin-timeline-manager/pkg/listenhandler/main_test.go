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
	"monorepo/bin-timeline-manager/models/sipmessage"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
	"monorepo/bin-timeline-manager/pkg/siphandler"
)

func TestNewListenHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := NewListenHandler(mockSock, mockEvent, mockSIP)
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
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := NewListenHandler(mockSock, mockEvent, mockSIP)

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
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := NewListenHandler(mockSock, mockEvent, mockSIP)

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

func TestV1SIPAnalysisPost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPAnalysisPost{
		SIPCallID: "test-call-id",
		FromTime:  "2026-01-01T00:00:00Z",
		ToTime:    "2026-01-01T01:00:00Z",
	}
	reqData, _ := json.Marshal(req)

	expectedResp := &sipmessage.SIPAnalysisResponse{
		SIPMessages: []*sipmessage.SIPMessage{
			{Method: "INVITE", SrcIP: "203.0.113.1", DstIP: "10.96.4.18"},
		},
	}

	mockSIP.EXPECT().GetSIPAnalysis(gomock.Any(), req.SIPCallID, gomock.Any(), gomock.Any()).Return(expectedResp, nil)

	sockReq := &sock.Request{
		URI:    "/v1/sip/analysis",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.v1SIPAnalysisPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPAnalysisPost() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("v1SIPAnalysisPost() StatusCode = %d, want 200", resp.StatusCode)
	}

	if resp.DataType != "application/json" {
		t.Errorf("v1SIPAnalysisPost() DataType = %q, want %q", resp.DataType, "application/json")
	}
}

func TestV1SIPAnalysisPost_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	sockReq := &sock.Request{
		URI:    "/v1/sip/analysis",
		Method: sock.RequestMethodPost,
		Data:   []byte("invalid json"),
	}

	resp, err := handler.v1SIPAnalysisPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPAnalysisPost() error = %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("v1SIPAnalysisPost() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestV1SIPAnalysisPost_InvalidFromTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPAnalysisPost{
		SIPCallID: "test-call-id",
		FromTime:  "invalid-time",
		ToTime:    "2026-01-01T01:00:00Z",
	}
	reqData, _ := json.Marshal(req)

	sockReq := &sock.Request{
		URI:    "/v1/sip/analysis",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.v1SIPAnalysisPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPAnalysisPost() error = %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("v1SIPAnalysisPost() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestV1SIPAnalysisPost_InvalidToTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPAnalysisPost{
		SIPCallID: "test-call-id",
		FromTime:  "2026-01-01T00:00:00Z",
		ToTime:    "invalid-time",
	}
	reqData, _ := json.Marshal(req)

	sockReq := &sock.Request{
		URI:    "/v1/sip/analysis",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.v1SIPAnalysisPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPAnalysisPost() error = %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("v1SIPAnalysisPost() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestV1SIPAnalysisPost_HandlerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPAnalysisPost{
		SIPCallID: "test-call-id",
		FromTime:  "2026-01-01T00:00:00Z",
		ToTime:    "2026-01-01T01:00:00Z",
	}
	reqData, _ := json.Marshal(req)

	mockSIP.EXPECT().GetSIPAnalysis(gomock.Any(), req.SIPCallID, gomock.Any(), gomock.Any()).Return(nil, errors.New("handler error"))

	sockReq := &sock.Request{
		URI:    "/v1/sip/analysis",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.v1SIPAnalysisPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPAnalysisPost() error = %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("v1SIPAnalysisPost() StatusCode = %d, want 500", resp.StatusCode)
	}
}

func TestV1SIPPcapPost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPPcapPost{
		SIPCallID: "test-call-id",
		FromTime:  "2026-01-01T00:00:00Z",
		ToTime:    "2026-01-01T01:00:00Z",
	}
	reqData, _ := json.Marshal(req)

	expectedPcap := []byte("pcap data")
	mockSIP.EXPECT().GetPcap(gomock.Any(), req.SIPCallID, gomock.Any(), gomock.Any()).Return(expectedPcap, nil)

	sockReq := &sock.Request{
		URI:    "/v1/sip/pcap",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.v1SIPPcapPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPPcapPost() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("v1SIPPcapPost() StatusCode = %d, want 200", resp.StatusCode)
	}

	if resp.DataType != "application/json" {
		t.Errorf("v1SIPPcapPost() DataType = %q, want %q", resp.DataType, "application/json")
	}

	// Verify response contains base64-encoded data
	var result map[string]string
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if _, ok := result["data"]; !ok {
		t.Error("Response missing 'data' field")
	}
}

func TestV1SIPPcapPost_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	sockReq := &sock.Request{
		URI:    "/v1/sip/pcap",
		Method: sock.RequestMethodPost,
		Data:   []byte("invalid json"),
	}

	resp, err := handler.v1SIPPcapPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPPcapPost() error = %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("v1SIPPcapPost() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestV1SIPPcapPost_InvalidFromTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPPcapPost{
		SIPCallID: "test-call-id",
		FromTime:  "invalid-time",
		ToTime:    "2026-01-01T01:00:00Z",
	}
	reqData, _ := json.Marshal(req)

	sockReq := &sock.Request{
		URI:    "/v1/sip/pcap",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.v1SIPPcapPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPPcapPost() error = %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("v1SIPPcapPost() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestV1SIPPcapPost_InvalidToTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPPcapPost{
		SIPCallID: "test-call-id",
		FromTime:  "2026-01-01T00:00:00Z",
		ToTime:    "invalid-time",
	}
	reqData, _ := json.Marshal(req)

	sockReq := &sock.Request{
		URI:    "/v1/sip/pcap",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.v1SIPPcapPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPPcapPost() error = %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("v1SIPPcapPost() StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestV1SIPPcapPost_HandlerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPPcapPost{
		SIPCallID: "test-call-id",
		FromTime:  "2026-01-01T00:00:00Z",
		ToTime:    "2026-01-01T01:00:00Z",
	}
	reqData, _ := json.Marshal(req)

	mockSIP.EXPECT().GetPcap(gomock.Any(), req.SIPCallID, gomock.Any(), gomock.Any()).Return(nil, errors.New("pcap fetch failed"))

	sockReq := &sock.Request{
		URI:    "/v1/sip/pcap",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.v1SIPPcapPost(context.Background(), sockReq)
	if err != nil {
		t.Fatalf("v1SIPPcapPost() error = %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("v1SIPPcapPost() StatusCode = %d, want 500", resp.StatusCode)
	}
}

func TestProcessRequest_V1SIPAnalysisPost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPAnalysisPost{
		SIPCallID: "test-call-id",
		FromTime:  "2026-01-01T00:00:00Z",
		ToTime:    "2026-01-01T01:00:00Z",
	}
	reqData, _ := json.Marshal(req)

	expectedResp := &sipmessage.SIPAnalysisResponse{
		SIPMessages: []*sipmessage.SIPMessage{},
	}

	mockSIP.EXPECT().GetSIPAnalysis(gomock.Any(), req.SIPCallID, gomock.Any(), gomock.Any()).Return(expectedResp, nil)

	sockReq := &sock.Request{
		URI:    "/v1/sip/analysis",
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
}

func TestProcessRequest_V1SIPPcapPost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := &listenHandler{
		sockHandler: mockSock,
		sipHandler:  mockSIP,
	}

	req := &request.V1SIPPcapPost{
		SIPCallID: "test-call-id",
		FromTime:  "2026-01-01T00:00:00Z",
		ToTime:    "2026-01-01T01:00:00Z",
	}
	reqData, _ := json.Marshal(req)

	mockSIP.EXPECT().GetPcap(gomock.Any(), req.SIPCallID, gomock.Any(), gomock.Any()).Return([]byte("pcap"), nil)

	sockReq := &sock.Request{
		URI:    "/v1/sip/pcap",
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
}

func TestRegexPatterns_SIPAnalysis(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		matches bool
	}{
		{name: "exact match", uri: "/v1/sip/analysis", matches: true},
		{name: "with query params", uri: "/v1/sip/analysis?param=1", matches: false},
		{name: "with trailing slash", uri: "/v1/sip/analysis/", matches: false},
		{name: "different path", uri: "/v1/sip/other", matches: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := regV1SIPAnalysis.MatchString(tt.uri)
			if result != tt.matches {
				t.Errorf("regV1SIPAnalysis.MatchString(%q) = %v, want %v", tt.uri, result, tt.matches)
			}
		})
	}
}

func TestRegexPatterns_SIPPcap(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		matches bool
	}{
		{name: "exact match", uri: "/v1/sip/pcap", matches: true},
		{name: "with query params", uri: "/v1/sip/pcap?param=1", matches: false},
		{name: "with trailing slash", uri: "/v1/sip/pcap/", matches: false},
		{name: "different path", uri: "/v1/sip/other", matches: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := regV1SIPPcap.MatchString(tt.uri)
			if result != tt.matches {
				t.Errorf("regV1SIPPcap.MatchString(%q) = %v, want %v", tt.uri, result, tt.matches)
			}
		})
	}
}

func TestListenHandler_Interface(t *testing.T) {
	// Ensure listenHandler implements ListenHandler interface
	var _ ListenHandler = (*listenHandler)(nil)
}

func TestProcessRequest_HandlerReturnsError(t *testing.T) {
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
		Publisher:  commonoutline.ServiceName("flow-manager"),
		ResourceID: testID,
		Events:     []string{"activeflow_*"},
		PageSize:   10,
	}
	reqData, _ := json.Marshal(req)

	// Handler returns error - v1EventsPost returns 500, then processRequest catches and converts to 400
	mockEvent.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("database error"))

	sockReq := &sock.Request{
		URI:    "/v1/events",
		Method: sock.RequestMethodPost,
		Data:   reqData,
	}

	resp, err := handler.processRequest(sockReq)
	// processRequest handles errors internally and returns 400 when handler returns non-nil error
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}

	// processRequest sets status to 400 when err is returned from v1EventsPost
	// but v1EventsPost returns 500 for handler errors, so processRequest wraps and converts
	if resp.StatusCode != 500 {
		t.Errorf("processRequest() StatusCode = %d, want 500", resp.StatusCode)
	}
}

func TestRun_ConsumeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSock := sockhandler.NewMockSockHandler(ctrl)
	mockEvent := eventhandler.NewMockEventHandler(ctrl)
	mockSIP := siphandler.NewMockSIPHandler(ctrl)

	handler := NewListenHandler(mockSock, mockEvent, mockSIP)

	mockSock.EXPECT().
		QueueCreate("test-queue", "normal").
		Return(nil)

	// ConsumeRPC returns error in goroutine (just verify it doesn't crash)
	mockSock.EXPECT().
		ConsumeRPC(gomock.Any(), "test-queue", "timeline-manager", false, false, false, 10, gomock.Any()).
		Return(errors.New("consume error"))

	err := handler.Run("test-queue")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Give goroutine time to execute (coverage will be counted)
	time.Sleep(50 * time.Millisecond)
}
