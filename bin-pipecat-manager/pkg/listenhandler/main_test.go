package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/listenhandler/models/request"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func TestNewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := NewListenHandler(mockSock, mockPipecatcall)
	if h == nil {
		t.Errorf("NewListenHandler returned nil")
	}
}

func TestSimpleResponse(t *testing.T) {
	tests := []struct {
		name string
		code int
		want int
	}{
		{
			name: "status 200",
			code: 200,
			want: 200,
		},
		{
			name: "status 400",
			code: 400,
			want: 400,
		},
		{
			name: "status 404",
			code: 404,
			want: 404,
		},
		{
			name: "status 500",
			code: 500,
			want: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := simpleResponse(tt.code)
			if resp.StatusCode != tt.want {
				t.Errorf("simpleResponse() StatusCode = %v, want %v", resp.StatusCode, tt.want)
			}
		})
	}
}

func TestProcessRequest_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	req := &sock.Request{
		URI:    "/v1/nonexistent",
		Method: sock.RequestMethodGet,
		Data:   json.RawMessage([]byte("")),
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("processRequest() error = %v, want nil", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("processRequest() StatusCode = %v, want 404", resp.StatusCode)
	}
}

func TestProcessV1PipecatcallsPost_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	req := &sock.Request{
		URI:    "/v1/pipecatcalls",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage([]byte("invalid json")),
	}

	resp, err := h.processV1PipecatcallsPost(context.Background(), req)
	if err != nil {
		t.Errorf("processV1PipecatcallsPost() error = %v, want nil", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("processV1PipecatcallsPost() StatusCode = %v, want 400", resp.StatusCode)
	}
}

func TestProcessV1PipecatcallsPost_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	reqData := request.V1DataPipecatcallsPost{
		ID:            uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
		CustomerID:    uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
		ActiveflowID:  uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
	}

	jsonData, _ := json.Marshal(reqData)

	expectedPipecatcall := &pipecatcall.Pipecatcall{
		Identity: identity.Identity{
			ID:         reqData.ID,
			CustomerID: reqData.CustomerID,
		},
	}

	mockPipecatcall.EXPECT().Start(
		gomock.Any(),
		reqData.ID,
		reqData.CustomerID,
		reqData.ActiveflowID,
		reqData.ReferenceType,
		reqData.ReferenceID,
		reqData.LLMType,
		reqData.LLMMessages,
		reqData.STTType,
		reqData.STTLanguage,
		reqData.TTSType,
		reqData.TTSLanguage,
		reqData.TTSVoiceID,
	).Return(expectedPipecatcall, nil)

	req := &sock.Request{
		URI:    "/v1/pipecatcalls",
		Method: sock.RequestMethodPost,
		Data:   jsonData,
	}

	resp, err := h.processV1PipecatcallsPost(context.Background(), req)
	if err != nil {
		t.Errorf("processV1PipecatcallsPost() error = %v, want nil", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("processV1PipecatcallsPost() StatusCode = %v, want 200", resp.StatusCode)
	}
}

func TestProcessV1PipecatcallsIDGet_InvalidURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	req := &sock.Request{
		URI:    "/v1/pipecatcalls",
		Method: sock.RequestMethodGet,
		Data:   json.RawMessage([]byte("")),
	}

	resp, err := h.processV1PipecatcallsIDGet(context.Background(), req)
	if err != nil {
		t.Errorf("processV1PipecatcallsIDGet() error = %v, want nil", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("processV1PipecatcallsIDGet() StatusCode = %v, want 400", resp.StatusCode)
	}
}

func TestProcessV1PipecatcallsIDStopPost_InvalidURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	req := &sock.Request{
		URI:    "/v1/pipecatcalls",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage([]byte("")),
	}

	resp, err := h.processV1PipecatcallsIDStopPost(context.Background(), req)
	if err != nil {
		t.Errorf("processV1PipecatcallsIDStopPost() error = %v, want nil", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("processV1PipecatcallsIDStopPost() StatusCode = %v, want 400", resp.StatusCode)
	}
}

// TestProcessV1PipecatcallsPost_HandlerTypedError asserts that a typed
// cerrors.NotFound returned by pipecatcallHandler.Start surfaces as 404
// (not 500) after the errorResponse conversion.
func TestProcessV1PipecatcallsPost_HandlerTypedError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	reqData := request.V1DataPipecatcallsPost{
		ID:            uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
		CustomerID:    uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
		ActiveflowID:  uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
	}
	jsonData, _ := json.Marshal(reqData)

	typedErr := cerrors.NotFound(commonoutline.ServiceNamePipecatManager, "PIPECATCALL_NOT_FOUND", "not found")
	mockPipecatcall.EXPECT().Start(
		gomock.Any(),
		reqData.ID,
		reqData.CustomerID,
		reqData.ActiveflowID,
		reqData.ReferenceType,
		reqData.ReferenceID,
		reqData.LLMType,
		reqData.LLMMessages,
		reqData.STTType,
		reqData.STTLanguage,
		reqData.TTSType,
		reqData.TTSLanguage,
		reqData.TTSVoiceID,
	).Return(nil, typedErr)

	req := &sock.Request{
		URI:    "/v1/pipecatcalls",
		Method: sock.RequestMethodPost,
		Data:   jsonData,
	}

	resp, err := h.processV1PipecatcallsPost(context.Background(), req)
	if err != nil {
		t.Errorf("processV1PipecatcallsPost() error = %v, want nil", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("processV1PipecatcallsPost() StatusCode = %v, want 404 (typed error must propagate)", resp.StatusCode)
	}
}

// TestProcessV1PipecatcallsPost_HandlerPlainError asserts that a plain
// error returned by pipecatcallHandler.Start surfaces as 500.
func TestProcessV1PipecatcallsPost_HandlerPlainError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	reqData := request.V1DataPipecatcallsPost{
		ID:            uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
		CustomerID:    uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
		ActiveflowID:  uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
	}
	jsonData, _ := json.Marshal(reqData)

	mockPipecatcall.EXPECT().Start(
		gomock.Any(),
		reqData.ID,
		reqData.CustomerID,
		reqData.ActiveflowID,
		reqData.ReferenceType,
		reqData.ReferenceID,
		reqData.LLMType,
		reqData.LLMMessages,
		reqData.STTType,
		reqData.STTLanguage,
		reqData.TTSType,
		reqData.TTSLanguage,
		reqData.TTSVoiceID,
	).Return(nil, fmt.Errorf("internal failure"))

	req := &sock.Request{
		URI:    "/v1/pipecatcalls",
		Method: sock.RequestMethodPost,
		Data:   jsonData,
	}

	resp, err := h.processV1PipecatcallsPost(context.Background(), req)
	if err != nil {
		t.Errorf("processV1PipecatcallsPost() error = %v, want nil", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("processV1PipecatcallsPost() StatusCode = %v, want 500", resp.StatusCode)
	}
}

// TestProcessV1MessagesPost_HandlerTypedError asserts that a typed
// cerrors.NotFound returned by pipecatcallHandler.SendMessage surfaces as 404.
func TestProcessV1MessagesPost_HandlerTypedError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	pipecatcallID := uuid.FromStringOrNil("9bd7ed8e-b3ab-11f0-a12a-d3f1af50fa4a")
	jsonData := []byte(`{"pipecatcall_id":"9bd7ed8e-b3ab-11f0-a12a-d3f1af50fa4a","message_id":"msg1","message_text":"hello","run_immediately":false,"audio_response":false}`)

	typedErr := cerrors.NotFound(commonoutline.ServiceNamePipecatManager, "PIPECATCALL_NOT_FOUND", "not found")
	mockPipecatcall.EXPECT().SendMessage(gomock.Any(), pipecatcallID, "msg1", "hello", false, false).Return(nil, typedErr)

	req := &sock.Request{
		URI:    "/v1/messages",
		Method: sock.RequestMethodPost,
		Data:   jsonData,
	}

	resp, err := h.processV1MessagesPost(context.Background(), req)
	if err != nil {
		t.Errorf("processV1MessagesPost() error = %v, want nil", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("processV1MessagesPost() StatusCode = %v, want 404", resp.StatusCode)
	}
}

// TestProcessV1MessagesPost_HandlerPlainError asserts that a plain error
// returned by pipecatcallHandler.SendMessage surfaces as 500.
func TestProcessV1MessagesPost_HandlerPlainError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	pipecatcallID := uuid.FromStringOrNil("9bd7ed8e-b3ab-11f0-a12a-d3f1af50fa4a")
	jsonData := []byte(`{"pipecatcall_id":"9bd7ed8e-b3ab-11f0-a12a-d3f1af50fa4a","message_id":"msg1","message_text":"hello","run_immediately":false,"audio_response":false}`)

	mockPipecatcall.EXPECT().SendMessage(gomock.Any(), pipecatcallID, "msg1", "hello", false, false).Return(nil, fmt.Errorf("send failed"))

	req := &sock.Request{
		URI:    "/v1/messages",
		Method: sock.RequestMethodPost,
		Data:   jsonData,
	}

	resp, err := h.processV1MessagesPost(context.Background(), req)
	if err != nil {
		t.Errorf("processV1MessagesPost() error = %v, want nil", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("processV1MessagesPost() StatusCode = %v, want 500", resp.StatusCode)
	}
}

// TestProcessV1PingGet_HandlerTypedError asserts that a typed error
// returned by pipecatcallHandler.Ping surfaces as the typed status code.
func TestProcessV1PingGet_HandlerTypedError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	typedErr := cerrors.NotFound(commonoutline.ServiceNamePipecatManager, "PIPECATCALL_NOT_FOUND", "not found")
	mockPipecatcall.EXPECT().Ping(gomock.Any()).Return(nil, typedErr)

	req := &sock.Request{
		URI:    "/v1/ping",
		Method: sock.RequestMethodGet,
	}

	resp, err := h.processV1PingGet(context.Background(), req)
	if err != nil {
		t.Errorf("processV1PingGet() error = %v, want nil", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("processV1PingGet() StatusCode = %v, want 404", resp.StatusCode)
	}
}

// TestProcessV1PingGet_HandlerPlainError asserts that a plain error
// returned by pipecatcallHandler.Ping surfaces as 500.
func TestProcessV1PingGet_HandlerPlainError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	mockPipecatcall.EXPECT().Ping(gomock.Any()).Return(nil, fmt.Errorf("ping failed"))

	req := &sock.Request{
		URI:    "/v1/ping",
		Method: sock.RequestMethodGet,
	}

	resp, err := h.processV1PingGet(context.Background(), req)
	if err != nil {
		t.Errorf("processV1PingGet() error = %v, want nil", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("processV1PingGet() StatusCode = %v, want 500", resp.StatusCode)
	}
}

func TestProcessV1MessagesPost_InvalidJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

	h := &listenHandler{
		sockHandler:        mockSock,
		pipecatcallHandler: mockPipecatcall,
	}

	req := &sock.Request{
		URI:    "/v1/messages",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage([]byte("invalid json")),
	}

	resp, err := h.processV1MessagesPost(context.Background(), req)
	if err != nil {
		t.Errorf("processV1MessagesPost() error = %v, want nil", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("processV1MessagesPost() StatusCode = %v, want 400", resp.StatusCode)
	}
}
