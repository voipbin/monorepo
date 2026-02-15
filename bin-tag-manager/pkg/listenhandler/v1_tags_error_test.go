package listenhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/pkg/taghandler"
)

func TestProcessV1TagsGet_Error(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		listErr error
	}{
		{
			name: "handler_error",
			request: &sock.Request{
				URI:      "/v1/tags?customer_id=92883d56-7fe3-11ec-8931-37d08180a2b9&page_size=10&page_token=2021-11-23T17:55:39.712000Z",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			listErr: fmt.Errorf("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			mockTag.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, tt.listErr)

			res, err := h.processV1TagsGet(context.Background(), tt.request)
			if err != nil {
				t.Errorf("processV1TagsGet should not return error, got: %v", err)
			}

			if res.StatusCode == 200 {
				t.Errorf("Expected error status code, got 200")
			}
		})
	}
}

func TestProcessV1TagsPost_Error(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		createErr error
	}{
		{
			name: "handler_error",
			request: &sock.Request{
				URI:      "/v1/tags",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","name":"test","detail":"test"}`),
			},
			createErr: fmt.Errorf("create failed"),
		},
		{
			name: "invalid_json",
			request: &sock.Request{
				URI:      "/v1/tags",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`invalid json`),
			},
			createErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			if tt.createErr != nil {
				mockTag.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, tt.createErr)
			}

			res, err := h.processV1TagsPost(context.Background(), tt.request)
			if err != nil {
				t.Errorf("processV1TagsPost should not return error, got: %v", err)
			}

			if res.StatusCode == 200 {
				t.Errorf("Expected error status code, got 200")
			}
		})
	}
}

func TestProcessV1TagsIDGet_Error(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		getErr  error
	}{
		{
			name: "handler_error",
			request: &sock.Request{
				URI:      "/v1/tags/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			getErr: fmt.Errorf("not found"),
		},
		{
			name: "invalid_uri",
			request: &sock.Request{
				URI:      "/v1/tags",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			getErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			if tt.getErr != nil {
				mockTag.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, tt.getErr)
			}

			res, err := h.processV1TagsIDGet(context.Background(), tt.request)
			if err != nil {
				t.Errorf("processV1TagsIDGet should not return error, got: %v", err)
			}

			if tt.name == "invalid_uri" && res.StatusCode != 400 {
				t.Errorf("Expected 400 for invalid URI, got %d", res.StatusCode)
			}
		})
	}
}

func TestProcessV1TagsIDPut_Error(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		updateErr error
	}{
		{
			name: "handler_error",
			request: &sock.Request{
				URI:      "/v1/tags/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update","detail":"update"}`),
			},
			updateErr: fmt.Errorf("update failed"),
		},
		{
			name: "invalid_json",
			request: &sock.Request{
				URI:      "/v1/tags/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`invalid json`),
			},
			updateErr: nil,
		},
		{
			name: "invalid_uri",
			request: &sock.Request{
				URI:      "/v1/tags",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update","detail":"update"}`),
			},
			updateErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			if tt.updateErr != nil {
				mockTag.EXPECT().UpdateBasicInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, tt.updateErr)
			}

			res, err := h.processV1TagsIDPut(context.Background(), tt.request)
			if err != nil {
				t.Errorf("processV1TagsIDPut should not return error, got: %v", err)
			}

			if res.StatusCode == 200 {
				t.Errorf("Expected error status code, got 200")
			}
		})
	}
}

func TestProcessV1TagsIDDelete_Error(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		deleteErr error
	}{
		{
			name: "handler_error",
			request: &sock.Request{
				URI:      "/v1/tags/c31676f0-4e69-11ec-afe3-77ba49fae527",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},
			deleteErr: fmt.Errorf("delete failed"),
		},
		{
			name: "invalid_uri",
			request: &sock.Request{
				URI:      "/v1/tags",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},
			deleteErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			if tt.deleteErr != nil {
				mockTag.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, tt.deleteErr)
			}

			res, err := h.processV1TagsIDDelete(context.Background(), tt.request)
			if err != nil {
				t.Errorf("processV1TagsIDDelete should not return error, got: %v", err)
			}

			if res.StatusCode == 200 {
				t.Errorf("Expected error status code, got 200")
			}
		})
	}
}

func TestProcessRequest_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)

	h := &listenHandler{
		sockHandler: mockSock,
		tagHandler:  mockTag,
	}

	request := &sock.Request{
		URI:      "/v1/unknown",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Errorf("processRequest should not return error, got: %v", err)
	}

	if res.StatusCode != 404 {
		t.Errorf("Expected 404 for unknown route, got %d", res.StatusCode)
	}
}

func TestSimpleResponse(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		expectedCode   int
	}{
		{
			name:         "status_200",
			code:         200,
			expectedCode: 200,
		},
		{
			name:         "status_400",
			code:         400,
			expectedCode: 400,
		},
		{
			name:         "status_404",
			code:         404,
			expectedCode: 404,
		},
		{
			name:         "status_500",
			code:         500,
			expectedCode: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := simpleResponse(tt.code)
			if res.StatusCode != tt.expectedCode {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectedCode, res.StatusCode)
			}
		})
	}
}

func TestNewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)

	h := NewListenHandler(mockSock, mockTag)

	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name           string
		queue          string
		exchangeDelay  string
		queueCreateErr error
		expectError    bool
	}{
		{
			name:           "runs_successfully",
			queue:          "test-queue",
			exchangeDelay:  "test-exchange",
			queueCreateErr: nil,
			expectError:    false,
		},
		{
			name:           "queue_create_error",
			queue:          "test-queue",
			exchangeDelay:  "test-exchange",
			queueCreateErr: fmt.Errorf("queue create failed"),
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			mockSock.EXPECT().QueueCreate(tt.queue, "normal").Return(tt.queueCreateErr)

			if tt.queueCreateErr == nil {
				mockSock.EXPECT().ConsumeRPC(
					context.Background(),
					tt.queue,
					"tag-manager",
					false,
					false,
					false,
					10,
					gomock.Any(),
				).Return(nil).AnyTimes()
			}

			err := h.Run(tt.queue, tt.exchangeDelay)
			if (err != nil) != tt.expectError {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectError, err)
			}
		})
	}
}
