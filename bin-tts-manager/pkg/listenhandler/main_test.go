package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-tts-manager/pkg/speakinghandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"
	"monorepo/bin-tts-manager/pkg/ttshandler"

	"go.uber.org/mock/gomock"
)

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTTS := ttshandler.NewMockTTSHandler(mc)
	mockStreaming := streaminghandler.NewMockStreamingHandler(mc)
	mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

	h := NewListenHandler(mockSock, mockTTS, mockStreaming, mockSpeaking)
	if h == nil {
		t.Fatal("expected handler, got nil")
	}

	lh, ok := h.(*listenHandler)
	if !ok {
		t.Fatal("handler is not listenHandler type")
	}

	if lh.sockHandler == nil {
		t.Error("sockHandler should not be nil")
	}
	if lh.ttsHandler == nil {
		t.Error("ttsHandler should not be nil")
	}
	if lh.streamingHandler == nil {
		t.Error("streamingHandler should not be nil")
	}
	if lh.speakingHandler == nil {
		t.Error("speakingHandler should not be nil")
	}
}

func Test_simpleResponse(t *testing.T) {
	tests := []struct {
		name string
		code int
	}{
		{name: "200 OK", code: 200},
		{name: "404 Not Found", code: 404},
		{name: "500 Internal Error", code: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := simpleResponse(tt.code)
			if res == nil {
				t.Fatal("expected response, got nil")
			}
			if res.StatusCode != tt.code {
				t.Errorf("expected status code %d, got %d", tt.code, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTTS := ttshandler.NewMockTTSHandler(mc)
	mockStreaming := streaminghandler.NewMockStreamingHandler(mc)
	mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

	h := &listenHandler{
		sockHandler:      mockSock,
		ttsHandler:       mockTTS,
		streamingHandler: mockStreaming,
		speakingHandler:  mockSpeaking,
	}

	tests := []struct {
		name   string
		req    *sock.Request
		expect int
	}{
		{
			name: "unknown URI",
			req: &sock.Request{
				URI:    "/v1/unknown",
				Method: sock.RequestMethodGet,
			},
			expect: 404,
		},
		{
			name: "unknown method",
			req: &sock.Request{
				URI:    "/v1/speeches",
				Method: sock.RequestMethodPut,
			},
			expect: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.processRequest(tt.req)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if res == nil {
				t.Fatal("expected response, got nil")
			}
			if res.StatusCode != tt.expect {
				t.Errorf("expected status code %d, got %d", tt.expect, res.StatusCode)
			}
		})
	}
}
