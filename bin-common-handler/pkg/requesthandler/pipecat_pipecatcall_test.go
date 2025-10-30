package requesthandler

import (
	"context"
	"testing"

	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_PipecatV1PipecatcallStart(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		cusotmerID    uuid.UUID
		activeflowID  uuid.UUID
		referenceType pipecatcall.ReferenceType
		referenceID   uuid.UUID
		llmType       pipecatcall.LLMType
		llmMessages   []map[string]any
		sttType       pipecatcall.STTType
		ttsType       pipecatcall.TTSType
		ttsVoiceID    string

		expectTarget  string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			name: "normal",

			id:            uuid.FromStringOrNil("775a5cb0-b45c-11f0-b77f-eb8a93884b92"),
			cusotmerID:    uuid.FromStringOrNil("087c5196-aba5-11f0-b874-67331df11790"),
			activeflowID:  uuid.FromStringOrNil("08b77244-aba5-11f0-867c-83627171cc5f"),
			referenceType: pipecatcall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("08ea1dac-aba5-11f0-98a0-075b9b4bcd29"),
			llmType:       "openai.gpt-3.5-turbo",
			sttType:       pipecatcall.STTTypeDeepgram,
			ttsType:       pipecatcall.TTSTypeElevenLabs,
			ttsVoiceID:    "09132436-aba5-11f0-835c-236dfc483b0e",
			llmMessages: []map[string]any{
				{"role": "system", "content": "Say hello world after user"},
				{"role": "user", "content": "Hello!"},
			},

			expectTarget: "bin-manager.pipecat-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/pipecatcalls",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"775a5cb0-b45c-11f0-b77f-eb8a93884b92","customer_id":"087c5196-aba5-11f0-b874-67331df11790","activeflow_id":"08b77244-aba5-11f0-867c-83627171cc5f","reference_type":"call","reference_id":"08ea1dac-aba5-11f0-98a0-075b9b4bcd29","llm_type":"openai.gpt-3.5-turbo","llm_messages":[{"content":"Say hello world after user","role":"system"},{"content":"Hello!","role":"user"}],"stt_type":"deepgram","tts_type":"elevenlabs","tts_voice_id":"09132436-aba5-11f0-835c-236dfc483b0e"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"093c46a4-aba5-11f0-a816-33a2e6a6d911"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			_, err := reqHandler.PipecatV1PipecatcallStart(ctx, tt.id, tt.cusotmerID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.llmType, tt.llmMessages, tt.sttType, tt.ttsType, tt.ttsVoiceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_PipecatV1PipecatcallGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("1c2231ec-aba6-11f0-8f2b-6fa1b3127622"),

			expectTarget: "bin-manager.pipecat-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/pipecatcalls/1c2231ec-aba6-11f0-8f2b-6fa1b3127622",
				Method: sock.RequestMethodGet,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1c2231ec-aba6-11f0-8f2b-6fa1b3127622"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			_, err := reqHandler.PipecatV1PipecatcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_PipecatV1PipecatcallTerminate(t *testing.T) {

	tests := []struct {
		name string

		hostID string
		id     uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			name: "normal",

			hostID: "8c342d88-b460-11f0-bc20-13fdcc8faeb3",
			id:     uuid.FromStringOrNil("1c506288-aba6-11f0-9faa-cfb11d9d5e47"),

			expectTarget: "bin-manager.pipecat-manager.request.8c342d88-b460-11f0-bc20-13fdcc8faeb3",
			expectRequest: &sock.Request{
				URI:    "/v1/pipecatcalls/1c506288-aba6-11f0-9faa-cfb11d9d5e47/stop",
				Method: sock.RequestMethodPost,
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1c506288-aba6-11f0-9faa-cfb11d9d5e47"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			_, err := reqHandler.PipecatV1PipecatcallTerminate(ctx, tt.hostID, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
