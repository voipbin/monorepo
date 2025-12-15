package requesthandler

import (
	"context"
	"reflect"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_AIV1AIGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *sock.Response

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectRes     []amai.AI
	}{
		{
			name: "normal",

			pageToken: "2020-09-20 03:23:20.995000",
			pageSize:  10,
			filters: map[string]string{
				"customer_id": "83fec56f-8e28-4356-a50c-7641e39ed2df",
				"deleted":     "false",
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"db662396-4449-456c-a6ee-39aa2ec30b55"},{"id":"0ea936d3-c74f-4744-8ca6-44e47178d88a"}]`),
			},

			expectURL:    "/v1/ais?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/ais?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&filter_customer_id=83fec56f-8e28-4356-a50c-7641e39ed2df&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			expectRes: []amai.AI{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("db662396-4449-456c-a6ee-39aa2ec30b55"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("0ea936d3-c74f-4744-8ca6-44e47178d88a"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.AIV1AIGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIGet(t *testing.T) {

	type test struct {
		name string
		aiID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *amai.AI
	}

	tests := []test{
		{
			name: "normal",
			aiID: uuid.FromStringOrNil("d628f462-cf28-47d9-ae37-c604c0ea2863"),

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/ais/d628f462-cf28-47d9-ae37-c604c0ea2863",
				Method: sock.RequestMethodGet,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"d628f462-cf28-47d9-ae37-c604c0ea2863"}`),
			},
			expectRes: &amai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d628f462-cf28-47d9-ae37-c604c0ea2863"),
				},
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

			res, err := reqHandler.AIV1AIGet(ctx, tt.aiID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AICreate(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		aiName      string
		detail      string
		engineType  amai.EngineType
		engineModel amai.EngineModel
		engineData  map[string]any
		engineKey   string
		initPrompt  string
		ttsType     amai.TTSType
		ttsVoiceID  string
		sttType     amai.STTType

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amai.AI
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("eeaf1e90-237a-4da5-a978-a8fc0eb691d0"),
			aiName:      "test name",
			detail:      "test detail",
			engineType:  amai.EngineTypeNone,
			engineModel: amai.EngineModelOpenaiGPT4,
			engineData: map[string]any{
				"key1": "value1",
				"key2": 2,
			},
			engineKey:  "test engine key",
			initPrompt: "test init prompt",
			ttsType:    amai.TTSTypeElevenLabs,
			ttsVoiceID: "test tts voice id",
			sttType:    amai.STTTypeDeepgram,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e6248322-de4f-4313-bd89-f9de1c6466a8"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/ais",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"eeaf1e90-237a-4da5-a978-a8fc0eb691d0","name":"test name","detail":"test detail","engine_model":"openai.gpt-4","engine_data":{"key1":"value1","key2":2},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test tts voice id","stt_type":"deepgram"}`),
			},
			expectRes: &amai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e6248322-de4f-4313-bd89-f9de1c6466a8"),
				},
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

			cf, err := reqHandler.AIV1AICreate(ctx, tt.customerID, tt.aiName, tt.detail, tt.engineType, tt.engineModel, tt.engineData, tt.engineKey, tt.initPrompt, tt.ttsType, tt.ttsVoiceID, tt.sttType)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_AIV1AIDelete(t *testing.T) {

	tests := []struct {
		name string

		aiID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amai.AI
	}{
		{
			name: "normal",

			aiID: uuid.FromStringOrNil("5d4c38bf-6cd5-4255-950a-9abf52704472"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"5d4c38bf-6cd5-4255-950a-9abf52704472"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/ais/5d4c38bf-6cd5-4255-950a-9abf52704472",
				Method: sock.RequestMethodDelete,
			},
			expectRes: &amai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5d4c38bf-6cd5-4255-950a-9abf52704472"),
				},
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

			res, err := reqHandler.AIV1AIDelete(ctx, tt.aiID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIUpdate(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		aiName      string
		detail      string
		engineType  amai.EngineType
		engineModel amai.EngineModel
		engineData  map[string]any
		engineKey   string
		initPrompt  string
		ttsType     amai.TTSType
		ttsVoiceID  string
		sttType     amai.STTType

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amai.AI
	}{
		{
			name: "normal",

			id:          uuid.FromStringOrNil("76380ede-f84a-11ed-a288-2bf54d8b92e6"),
			aiName:      "test name",
			detail:      "test detail",
			engineType:  amai.EngineTypeNone,
			engineModel: amai.EngineModelOpenaiGPT4,
			engineData: map[string]any{
				"key1": "value1",
				"key2": 2,
			},
			engineKey:  "test engine key",
			initPrompt: "test init prompt",
			ttsType:    amai.TTSTypeElevenLabs,
			ttsVoiceID: "test tts voice id",
			sttType:    amai.STTTypeDeepgram,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"76380ede-f84a-11ed-a288-2bf54d8b92e6"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/ais/76380ede-f84a-11ed-a288-2bf54d8b92e6",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-4","engine_data":{"key1":"value1","key2":2},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test tts voice id","stt_type":"deepgram"}`),
			},
			expectRes: &amai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("76380ede-f84a-11ed-a288-2bf54d8b92e6"),
				},
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

			cf, err := reqHandler.AIV1AIUpdate(ctx, tt.id, tt.aiName, tt.detail, tt.engineType, tt.engineModel, tt.engineData, tt.engineKey, tt.initPrompt, tt.ttsType, tt.ttsVoiceID, tt.sttType)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}
