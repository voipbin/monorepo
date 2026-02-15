package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-rag-manager/pkg/chunker"
	"monorepo/bin-rag-manager/pkg/raghandler"
)

// mockSockHandler for testing
type mockSockHandler struct {
	queueCreateFunc func(name string, queueType string) error
	consumeRPCFunc  func(ctx context.Context, queueName, consumerName string, skipVerify, enableDelay, enableDelayHourly bool, delayExpiredMin int, handler sock.CbMsgRPC) error
}

func (m *mockSockHandler) QueueCreate(name string, queueType string) error {
	if m.queueCreateFunc != nil {
		return m.queueCreateFunc(name, queueType)
	}
	return nil
}

func (m *mockSockHandler) ConsumeRPC(ctx context.Context, queueName, consumerName string, skipVerify, enableDelay, enableDelayHourly bool, delayExpiredMin int, handler sock.CbMsgRPC) error {
	if m.consumeRPCFunc != nil {
		return m.consumeRPCFunc(ctx, queueName, consumerName, skipVerify, enableDelay, enableDelayHourly, delayExpiredMin, handler)
	}
	return nil
}

func (m *mockSockHandler) Close() {}

func (m *mockSockHandler) Connect() {}

func (m *mockSockHandler) ConsumeMessage(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error {
	return nil
}

func (m *mockSockHandler) TopicCreate(name string) error {
	return nil
}

func (m *mockSockHandler) EventPublish(topic string, key string, evt *sock.Event) error {
	return nil
}

func (m *mockSockHandler) EventPublishWithDelay(topic string, key string, evt *sock.Event, delay int) error {
	return nil
}

func (m *mockSockHandler) RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error) {
	return nil, nil
}

func (m *mockSockHandler) RequestPublishWithDelay(queueName string, req *sock.Request, delay int) error {
	return nil
}

func (m *mockSockHandler) QueueSubscribe(name string, topic string) error {
	return nil
}

// mockRagHandler for testing
type mockRagHandlerForListen struct {
	queryFunc         func(ctx context.Context, req *raghandler.QueryRequest) (*raghandler.QueryResponse, error)
	indexFullFunc     func(ctx context.Context) error
	indexIncrFunc     func(ctx context.Context, files []string) error
	indexStatusFunc   func(ctx context.Context) (*raghandler.IndexStatusResponse, error)
}

func (m *mockRagHandlerForListen) Query(ctx context.Context, req *raghandler.QueryRequest) (*raghandler.QueryResponse, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, req)
	}
	return &raghandler.QueryResponse{
		Answer: "test answer",
		Sources: []raghandler.Source{
			{
				SourceFile:     "test.md",
				SectionTitle:   "Test",
				DocType:        chunker.DocTypeDesign,
				RelevanceScore: 0.95,
			},
		},
	}, nil
}

func (m *mockRagHandlerForListen) IndexFull(ctx context.Context) error {
	if m.indexFullFunc != nil {
		return m.indexFullFunc(ctx)
	}
	return nil
}

func (m *mockRagHandlerForListen) IndexIncremental(ctx context.Context, files []string) error {
	if m.indexIncrFunc != nil {
		return m.indexIncrFunc(ctx, files)
	}
	return nil
}

func (m *mockRagHandlerForListen) IndexStatus(ctx context.Context) (*raghandler.IndexStatusResponse, error) {
	if m.indexStatusFunc != nil {
		return m.indexStatusFunc(ctx)
	}
	return &raghandler.IndexStatusResponse{
		LastRun:    time.Now(),
		ChunkCount: 100,
		Errors:     []string{},
	}, nil
}

func TestListenHandler_Interface(t *testing.T) {
	var _ ListenHandler = &listenHandler{}
}

func TestNewListenHandler(t *testing.T) {
	sockHandler := &mockSockHandler{}
	ragHandler := &mockRagHandlerForListen{}

	h := NewListenHandler(sockHandler, ragHandler)
	if h == nil {
		t.Error("expected non-nil handler")
	}
}

func TestListenHandler_Run(t *testing.T) {
	queueCreated := false
	consumeCalled := false

	sockHandler := &mockSockHandler{
		queueCreateFunc: func(name string, queueType string) error {
			queueCreated = true
			if queueType != "normal" {
				t.Errorf("expected queueType 'normal', got %q", queueType)
			}
			return nil
		},
		consumeRPCFunc: func(ctx context.Context, queueName, consumerName string, skipVerify, enableDelay, enableDelayHourly bool, delayExpiredMin int, handler sock.CbMsgRPC) error {
			consumeCalled = true
			return nil
		},
	}
	ragHandler := &mockRagHandlerForListen{}

	h := NewListenHandler(sockHandler, ragHandler)
	err := h.Run("test-queue", "test-exchange")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !queueCreated {
		t.Error("expected queue to be created")
	}

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

	if !consumeCalled {
		t.Error("expected ConsumeRPC to be called")
	}
}

func TestListenHandler_Run_QueueCreateError(t *testing.T) {
	expectedErr := errors.New("queue create failed")
	sockHandler := &mockSockHandler{
		queueCreateFunc: func(name string, queueType string) error {
			return expectedErr
		},
	}
	ragHandler := &mockRagHandlerForListen{}

	h := NewListenHandler(sockHandler, ragHandler)
	err := h.Run("test-queue", "test-exchange")
	if err == nil {
		t.Error("expected error from queue creation")
	}
}

func TestSimpleResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"success", 200},
		{"created", 201},
		{"bad request", 400},
		{"not found", 404},
		{"server error", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := simpleResponse(tt.statusCode)
			if resp.StatusCode != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, resp.StatusCode)
			}
		})
	}
}

func TestJsonResponse(t *testing.T) {
	data := map[string]string{"key": "value"}
	resp := jsonResponse(200, data)

	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}
	if resp.DataType != "application/json" {
		t.Errorf("expected DataType 'application/json', got %q", resp.DataType)
	}

	var decoded map[string]string
	if err := json.Unmarshal(resp.Data, &decoded); err != nil {
		t.Errorf("failed to unmarshal response data: %v", err)
	}
	if decoded["key"] != "value" {
		t.Error("response data not marshalled correctly")
	}
}

func TestJsonResponse_MarshalError(t *testing.T) {
	// Create data that cannot be marshalled (channels, functions, etc.)
	data := make(chan int)
	resp := jsonResponse(200, data)

	if resp.StatusCode != 500 {
		t.Errorf("expected status code 500 for marshal error, got %d", resp.StatusCode)
	}
}

func TestProcessRequest_QueryPost(t *testing.T) {
	ragHandler := &mockRagHandlerForListen{
		queryFunc: func(ctx context.Context, req *raghandler.QueryRequest) (*raghandler.QueryResponse, error) {
			if req.Query != "test query" {
				t.Errorf("expected query 'test query', got %q", req.Query)
			}
			return &raghandler.QueryResponse{
				Answer:  "test answer",
				Sources: []raghandler.Source{},
			}, nil
		},
	}

	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  ragHandler,
	}

	reqData := raghandler.QueryRequest{
		Query: "test query",
		TopK:  5,
	}
	data, _ := json.Marshal(reqData)

	req := &sock.Request{
		URI:    "/v1/rags/query",
		Method: sock.RequestMethodPost,
		Data:   data,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}
}

func TestProcessRequest_QueryPost_EmptyQuery(t *testing.T) {
	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  &mockRagHandlerForListen{},
	}

	reqData := raghandler.QueryRequest{
		Query: "",
	}
	data, _ := json.Marshal(reqData)

	req := &sock.Request{
		URI:    "/v1/rags/query",
		Method: sock.RequestMethodPost,
		Data:   data,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected status code 400, got %d", resp.StatusCode)
	}
}

func TestProcessRequest_QueryPost_InvalidJSON(t *testing.T) {
	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  &mockRagHandlerForListen{},
	}

	req := &sock.Request{
		URI:    "/v1/rags/query",
		Method: sock.RequestMethodPost,
		Data:   []byte("invalid json"),
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected status code 400, got %d", resp.StatusCode)
	}
}

func TestProcessRequest_IndexPost(t *testing.T) {
	indexCalled := false
	ragHandler := &mockRagHandlerForListen{
		indexFullFunc: func(ctx context.Context) error {
			indexCalled = true
			return nil
		},
	}

	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  ragHandler,
	}

	req := &sock.Request{
		URI:    "/v1/rags/index",
		Method: sock.RequestMethodPost,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 202 {
		t.Errorf("expected status code 202, got %d", resp.StatusCode)
	}

	// Give goroutine time to run
	time.Sleep(50 * time.Millisecond)

	if !indexCalled {
		t.Error("expected IndexFull to be called")
	}
}

func TestProcessRequest_IndexIncrementalPost(t *testing.T) {
	filesReceived := []string{}
	ragHandler := &mockRagHandlerForListen{
		indexIncrFunc: func(ctx context.Context, files []string) error {
			filesReceived = files
			return nil
		},
	}

	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  ragHandler,
	}

	reqData := indexIncrementalRequest{
		Files: []string{"file1.md", "file2.md"},
	}
	data, _ := json.Marshal(reqData)

	req := &sock.Request{
		URI:    "/v1/rags/index/incremental",
		Method: sock.RequestMethodPost,
		Data:   data,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 202 {
		t.Errorf("expected status code 202, got %d", resp.StatusCode)
	}

	// Give goroutine time to run
	time.Sleep(50 * time.Millisecond)

	if len(filesReceived) != 2 {
		t.Errorf("expected 2 files, got %d", len(filesReceived))
	}
}

func TestProcessRequest_IndexIncrementalPost_NoFiles(t *testing.T) {
	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  &mockRagHandlerForListen{},
	}

	reqData := indexIncrementalRequest{
		Files: []string{},
	}
	data, _ := json.Marshal(reqData)

	req := &sock.Request{
		URI:    "/v1/rags/index/incremental",
		Method: sock.RequestMethodPost,
		Data:   data,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected status code 400, got %d", resp.StatusCode)
	}
}

func TestProcessRequest_IndexStatusGet(t *testing.T) {
	ragHandler := &mockRagHandlerForListen{
		indexStatusFunc: func(ctx context.Context) (*raghandler.IndexStatusResponse, error) {
			return &raghandler.IndexStatusResponse{
				LastRun:    time.Now(),
				ChunkCount: 150,
				Errors:     []string{},
			}, nil
		},
	}

	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  ragHandler,
	}

	req := &sock.Request{
		URI:    "/v1/rags/index/status",
		Method: sock.RequestMethodGet,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}

	var status raghandler.IndexStatusResponse
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
	}
	if status.ChunkCount != 150 {
		t.Errorf("expected chunk count 150, got %d", status.ChunkCount)
	}
}

func TestProcessRequest_NotFound(t *testing.T) {
	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  &mockRagHandlerForListen{},
	}

	req := &sock.Request{
		URI:    "/v1/unknown/endpoint",
		Method: sock.RequestMethodGet,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", resp.StatusCode)
	}
}

func TestProcessRequest_QueryError(t *testing.T) {
	ragHandler := &mockRagHandlerForListen{
		queryFunc: func(ctx context.Context, req *raghandler.QueryRequest) (*raghandler.QueryResponse, error) {
			return nil, errors.New("query failed")
		},
	}

	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  ragHandler,
	}

	reqData := raghandler.QueryRequest{
		Query: "test",
	}
	data, _ := json.Marshal(reqData)

	req := &sock.Request{
		URI:    "/v1/rags/query",
		Method: sock.RequestMethodPost,
		Data:   data,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected status code 500, got %d", resp.StatusCode)
	}
}

func TestProcessRequest_IndexStatusError(t *testing.T) {
	ragHandler := &mockRagHandlerForListen{
		indexStatusFunc: func(ctx context.Context) (*raghandler.IndexStatusResponse, error) {
			return nil, errors.New("status failed")
		},
	}

	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  ragHandler,
	}

	req := &sock.Request{
		URI:    "/v1/rags/index/status",
		Method: sock.RequestMethodGet,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected status code 500, got %d", resp.StatusCode)
	}
}

func TestRegexpPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern *regexp.Regexp
		uri     string
		matches bool
	}{
		{"query endpoint", regV1RagQuery, "/v1/rags/query", true},
		{"index endpoint", regV1RagIndex, "/v1/rags/index", true},
		{"index incremental", regV1RagIndexIncr, "/v1/rags/index/incremental", true},
		{"index status", regV1RagIndexStatus, "/v1/rags/index/status", true},
		{"query no match", regV1RagQuery, "/v1/rags/query/other", false},
		{"index no match", regV1RagIndex, "/v1/rags/index/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pattern.MatchString(tt.uri)
			if result != tt.matches {
				t.Errorf("expected match=%v for pattern and URI %q, got %v", tt.matches, tt.uri, result)
			}
		})
	}
}

func TestProcessRequest_IndexIncrementalPost_InvalidJSON(t *testing.T) {
	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  &mockRagHandlerForListen{},
	}

	req := &sock.Request{
		URI:    "/v1/rags/index/incremental",
		Method: sock.RequestMethodPost,
		Data:   []byte("invalid json"),
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected status code 400, got %d", resp.StatusCode)
	}
}

func TestProcessRequest_WrongMethod(t *testing.T) {
	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  &mockRagHandlerForListen{},
	}

	tests := []struct {
		name   string
		uri    string
		method sock.RequestMethod
	}{
		{"query with GET", "/v1/rags/query", sock.RequestMethodGet},
		{"index with GET", "/v1/rags/index", sock.RequestMethodGet},
		{"incremental with GET", "/v1/rags/index/incremental", sock.RequestMethodGet},
		{"status with POST", "/v1/rags/index/status", sock.RequestMethodPost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &sock.Request{
				URI:    tt.uri,
				Method: tt.method,
			}

			resp, err := h.processRequest(req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if resp.StatusCode != 404 {
				t.Errorf("expected status code 404, got %d", resp.StatusCode)
			}
		})
	}
}

func TestProcessRequest_QueryWithDocTypes(t *testing.T) {
	docTypesReceived := []chunker.DocType{}
	ragHandler := &mockRagHandlerForListen{
		queryFunc: func(ctx context.Context, req *raghandler.QueryRequest) (*raghandler.QueryResponse, error) {
			docTypesReceived = req.DocTypes
			return &raghandler.QueryResponse{
				Answer:  "test answer",
				Sources: []raghandler.Source{},
			}, nil
		},
	}

	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  ragHandler,
	}

	reqData := raghandler.QueryRequest{
		Query:    "test query",
		TopK:     10,
		DocTypes: []chunker.DocType{chunker.DocTypeOpenAPI, chunker.DocTypeDesign},
	}
	data, _ := json.Marshal(reqData)

	req := &sock.Request{
		URI:    "/v1/rags/query",
		Method: sock.RequestMethodPost,
		Data:   data,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}
	if len(docTypesReceived) != 2 {
		t.Errorf("expected 2 doc types, got %d", len(docTypesReceived))
	}
}

func TestListenHandler_Run_ConsumeRPCError(t *testing.T) {
	errorReturned := false
	sockHandler := &mockSockHandler{
		queueCreateFunc: func(name string, queueType string) error {
			return nil
		},
		consumeRPCFunc: func(ctx context.Context, queueName, consumerName string, skipVerify, enableDelay, enableDelayHourly bool, delayExpiredMin int, handler sock.CbMsgRPC) error {
			errorReturned = true
			return errors.New("consume error")
		},
	}
	ragHandler := &mockRagHandlerForListen{}

	h := NewListenHandler(sockHandler, ragHandler)
	err := h.Run("test-queue", "test-exchange")
	if err != nil {
		t.Errorf("Run should not return error, got: %v", err)
	}

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)

	if !errorReturned {
		t.Error("expected ConsumeRPC to be called and return error")
	}
}

func TestProcessRequest_AllMethods(t *testing.T) {
	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  &mockRagHandlerForListen{},
	}

	tests := []struct {
		name           string
		method         sock.RequestMethod
		uri            string
		expectedStatus int
	}{
		{"PUT not found", sock.RequestMethodPut, "/v1/rags/query", 404},
		{"DELETE not found", sock.RequestMethodDelete, "/v1/rags/query", 404},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &sock.Request{
				URI:    tt.uri,
				Method: tt.method,
			}

			resp, err := h.processRequest(req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestProcessRequest_MultipleRequests(t *testing.T) {
	callCount := 0
	ragHandler := &mockRagHandlerForListen{
		queryFunc: func(ctx context.Context, req *raghandler.QueryRequest) (*raghandler.QueryResponse, error) {
			callCount++
			return &raghandler.QueryResponse{Answer: "answer", Sources: []raghandler.Source{}}, nil
		},
	}

	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  ragHandler,
	}

	reqData := raghandler.QueryRequest{Query: "test"}
	data, _ := json.Marshal(reqData)

	// Make multiple requests
	for i := 0; i < 3; i++ {
		req := &sock.Request{
			URI:    "/v1/rags/query",
			Method: sock.RequestMethodPost,
			Data:   data,
		}

		resp, err := h.processRequest(req)
		if err != nil {
			t.Errorf("unexpected error on request %d: %v", i, err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected status 200, got %d", i, resp.StatusCode)
		}
	}

	if callCount != 3 {
		t.Errorf("expected 3 query calls, got %d", callCount)
	}
}
