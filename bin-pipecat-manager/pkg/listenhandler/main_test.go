package listenhandler

import (
	"context"
	"encoding/json"
	"testing"

	"monorepo/bin-common-handler/models/identity"
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
