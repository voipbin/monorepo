package listenhandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_processV1PipecatcallsPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responsePipecatcall *pipecatcall.Pipecatcall

		expectID            uuid.UUID
		expectCustomerID    uuid.UUID
		expectActiveflowID  uuid.UUID
		expectReferenceType pipecatcall.ReferenceType
		expectReferenceID   uuid.UUID
		expectLLM           pipecatcall.LLM
		expectSTT           pipecatcall.STT
		expectTTS           pipecatcall.TTS
		expectVoiceID       string
		expectMessages      []map[string]any

		expectRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/pipecatcalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"ffa2ac7a-b3a4-11f0-aeda-5b3b3498e619","customer_id":"cd1d344c-aa43-11f0-a6b9-fb100dc5e57c","activeflow_id":"cd65b1b8-aa43-11f0-8c1e-bfc7dc74bbd9","reference_type":"call","reference_id":"cd97ff42-aa43-11f0-9042-0f14ff740ec1","llm":"openai.gpt-3.5-turbo","stt":"deepgram","tts":"elevenlabs","voice_id":"c41bacee-aadd-11f0-a5a5-8bedee791598","messages":[{"role":"system","content":"Say hello world after user"},{"role":"user","content":"Hello!"}]}`),
			},

			responsePipecatcall: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b973a0dc-aa45-11f0-a03f-071efa103755"),
				},
			},

			expectID:            uuid.FromStringOrNil("ffa2ac7a-b3a4-11f0-aeda-5b3b3498e619"),
			expectCustomerID:    uuid.FromStringOrNil("cd1d344c-aa43-11f0-a6b9-fb100dc5e57c"),
			expectActiveflowID:  uuid.FromStringOrNil("cd65b1b8-aa43-11f0-8c1e-bfc7dc74bbd9"),
			expectReferenceType: pipecatcall.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("cd97ff42-aa43-11f0-9042-0f14ff740ec1"),
			expectLLM:           pipecatcall.LLM("openai.gpt-3.5-turbo"),
			expectSTT:           pipecatcall.STTDeepgram,
			expectTTS:           pipecatcall.TTSElevenLabs,
			expectVoiceID:       "c41bacee-aadd-11f0-a5a5-8bedee791598",
			expectMessages: []map[string]any{
				{"role": "system", "content": "Say hello world after user"},
				{"role": "user", "content": "Hello!"},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b973a0dc-aa45-11f0-a03f-071efa103755","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				pipecatcallHandler: mockPipecatcall,
			}
			ctx := context.Background()

			mockPipecatcall.EXPECT().Start(ctx,
				tt.expectID,
				tt.expectCustomerID,
				tt.expectActiveflowID,
				tt.expectReferenceType,
				tt.expectReferenceID,
				tt.expectLLM,
				tt.expectSTT,
				tt.expectTTS,
				tt.expectVoiceID,
				tt.expectMessages,
			).Return(tt.responsePipecatcall, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1PipecatcallsIDGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responsePipecatcall *pipecatcall.Pipecatcall

		expectPipecatcallID uuid.UUID
		expectRes           *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/v1/pipecatcalls/7aa63386-ab0a-11f0-9db2-a34eeb9c9133",
				Method: sock.RequestMethodGet,
			},

			responsePipecatcall: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7aa63386-ab0a-11f0-9db2-a34eeb9c9133"),
				},
			},

			expectPipecatcallID: uuid.FromStringOrNil("7aa63386-ab0a-11f0-9db2-a34eeb9c9133"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7aa63386-ab0a-11f0-9db2-a34eeb9c9133","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				pipecatcallHandler: mockPipecatcall,
			}
			ctx := context.Background()

			mockPipecatcall.EXPECT().Get(ctx, tt.expectPipecatcallID).Return(tt.responsePipecatcall, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1PipecatcallsIDStopPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responsePipecatcall *pipecatcall.Pipecatcall

		expectPipecatcallID uuid.UUID
		expectRes           *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/v1/pipecatcalls/e594fff6-ab0a-11f0-8220-1fe5a6807315/stop",
				Method: sock.RequestMethodPost,
			},

			responsePipecatcall: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e594fff6-ab0a-11f0-8220-1fe5a6807315"),
				},
			},

			expectPipecatcallID: uuid.FromStringOrNil("e594fff6-ab0a-11f0-8220-1fe5a6807315"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e594fff6-ab0a-11f0-8220-1fe5a6807315","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockPipecatcall := pipecatcallhandler.NewMockPipecatcallHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				pipecatcallHandler: mockPipecatcall,
			}
			ctx := context.Background()

			mockPipecatcall.EXPECT().Stop(ctx, tt.expectPipecatcallID).Return(tt.responsePipecatcall, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
