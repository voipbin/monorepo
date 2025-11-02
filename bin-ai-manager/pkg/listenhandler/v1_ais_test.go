package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/aihandler"
)

func Test_processV1AIsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIs []*ai.AI

		expectPageSize  uint64
		expectPageToken string
		expectFilters   map[string]string
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/ais?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=24676972-7f49-11ec-bc89-b7d33e9d3ea8&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			responseAIs: []*ai.AI{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("0b61dcbe-a770-11ed-bab4-2fc1dac66672"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("0bbe1dee-a770-11ed-b455-cbb60d5dd90b"),
					},
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-05-03 21:35:02.809",
			expectFilters: map[string]string{
				"deleted":     "false",
				"customer_id": "24676972-7f49-11ec-bc89-b7d33e9d3ea8",
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0b61dcbe-a770-11ed-bab4-2fc1dac66672","customer_id":"00000000-0000-0000-0000-000000000000"},{"id":"0bbe1dee-a770-11ed-b455-cbb60d5dd90b","customer_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				aiHandler:   mockAI,
			}

			mockAI.EXPECT().Gets(gomock.Any(), tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseAIs, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AIsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAI *ai.AI

		expectCustomerID  uuid.UUID
		expectName        string
		expectDetail      string
		expectEngineType  ai.EngineType
		expectEngineModel ai.EngineModel
		expectEngineData  map[string]any
		expectEngineKey   string
		expectInitPrompt  string
		expectTTSType     ai.TTSType
		expectTTSVoiceID  string
		expectSTTType     ai.STTType
		expectRes         *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/ais",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "58e7502c-a770-11ed-9b86-7fabe2dba847", "name": "test name", "detail": "test detail", "engine_type":"", "engine_model": "openai.gpt-4", "engine_data": {"key1": "val1"}, "engine_key": "test engine key", "init_prompt": "test init prompt", "tts_type": "elevenlabs", "tts_voice_id": "test-voice-id", "stt_type": "deepgram"}`),
			},

			responseAI: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("59230ca2-a770-11ed-b5dd-2783587ed477"),
				},
			},

			expectCustomerID:  uuid.FromStringOrNil("58e7502c-a770-11ed-9b86-7fabe2dba847"),
			expectName:        "test name",
			expectDetail:      "test detail",
			expectEngineType:  ai.EngineTypeNone,
			expectEngineModel: ai.EngineModelOpenaiGPT4,
			expectEngineData: map[string]any{
				"key1": "val1",
			},
			expectEngineKey:  "test engine key",
			expectInitPrompt: "test init prompt",
			expectTTSType:    ai.TTSTypeElevenLabs,
			expectTTSVoiceID: "test-voice-id",
			expectSTTType:    ai.STTTypeDeepgram,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"59230ca2-a770-11ed-b5dd-2783587ed477","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				aiHandler:   mockAI,
			}

			mockAI.EXPECT().Create(
				gomock.Any(),
				tt.expectCustomerID,
				tt.expectName,
				tt.expectDetail,
				tt.expectEngineType,
				tt.expectEngineModel,
				tt.expectEngineData,
				tt.expectEngineKey,
				tt.expectInitPrompt,
				tt.expectTTSType,
				tt.expectTTSVoiceID,
				tt.expectSTTType,
			).Return(tt.responseAI, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AIsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAI *ai.AI

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/ais/de740384-a770-11ed-afab-5f9c8a447889",
				Method: sock.RequestMethodGet,
			},

			&ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("de740384-a770-11ed-afab-5f9c8a447889"),
				},
			},

			uuid.FromStringOrNil("de740384-a770-11ed-afab-5f9c8a447889"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"de740384-a770-11ed-afab-5f9c8a447889","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				aiHandler:   mockAI,
			}

			mockAI.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseAI, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AIsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAI *ai.AI

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/ais/de99e522-a770-11ed-a0ab-5b39ee2db203",
				Method: sock.RequestMethodDelete,
			},

			&ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203"),
				},
			},

			uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"de99e522-a770-11ed-a0ab-5b39ee2db203","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				aiHandler:   mockAI,
			}

			mockAI.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseAI, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AIsIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAI *ai.AI

		expectID          uuid.UUID
		expectName        string
		expectDetail      string
		expectEngineType  ai.EngineType
		expectEngineModel ai.EngineModel
		expectEngineData  map[string]any
		expectEngineKey   string
		expectInitPrompt  string
		expectTTSType     ai.TTSType
		expectTTSVoiceID  string
		expectSTTType     ai.STTType
		expectRes         *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/ais/fa4d3b6a-f82f-11ed-9176-d32f5705e10c",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"new name","detail":"new detail","engine_type":"","engine_model":"openai.gpt-4","engine_data":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"new prompt","tts_type":"cartesia","tts_voice_id":"new-voice-id","stt_type":"deepgram"}`),
			},

			responseAI: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
				},
			},

			expectID:          uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
			expectName:        "new name",
			expectDetail:      "new detail",
			expectEngineType:  ai.EngineTypeNone,
			expectEngineModel: ai.EngineModelOpenaiGPT4,
			expectEngineData: map[string]any{
				"key1": "val1",
			},
			expectEngineKey:  "test engine key",
			expectInitPrompt: "new prompt",
			expectTTSType:    ai.TTSTypeCartesia,
			expectTTSVoiceID: "new-voice-id",
			expectSTTType:    ai.STTTypeDeepgram,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fa4d3b6a-f82f-11ed-9176-d32f5705e10c","customer_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				aiHandler:   mockAI,
			}

			mockAI.EXPECT().Update(
				gomock.Any(),
				tt.expectID,
				tt.expectName,
				tt.expectDetail,
				tt.expectEngineType,
				tt.expectEngineModel,
				tt.expectEngineData,
				tt.expectEngineKey,
				tt.expectInitPrompt,
				tt.expectTTSType,
				tt.expectTTSVoiceID,
				tt.expectSTTType,
			).Return(tt.responseAI, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
