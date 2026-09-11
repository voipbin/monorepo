package listenhandler

import (
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/participant"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/participanthandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

func Test_processV1AIsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAIs []*ai.AI

		expectPageSize  uint64
		expectPageToken string
		expectFilters   map[ai.Field]any
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/ais?page_size=10&page_token=2020-05-03T21:35:02.809Z&filter_customer_id=24676972-7f49-11ec-bc89-b7d33e9d3ea8&filter_deleted=false",
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
			expectPageToken: "2020-05-03T21:35:02.809Z",
			expectFilters: map[ai.Field]any{
				ai.FieldDeleted:    false,
				ai.FieldCustomerID: uuid.FromStringOrNil("24676972-7f49-11ec-bc89-b7d33e9d3ea8"),
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0b61dcbe-a770-11ed-bab4-2fc1dac66672","customer_id":"00000000-0000-0000-0000-000000000000","is_insight_active":false,"rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","direct_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null},{"id":"0bbe1dee-a770-11ed-b455-cbb60d5dd90b","customer_id":"00000000-0000-0000-0000-000000000000","is_insight_active":false,"rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","direct_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}]`),
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

			mockAI.EXPECT().List(gomock.Any(), tt.expectPageSize, tt.expectPageToken, gomock.Any()).Return(tt.responseAIs, nil)
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

func Test_processV1AIsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAI *ai.AI

		expectCustomerID  uuid.UUID
		expectName        string
		expectDetail      string
		expectEngineModel ai.EngineModel
		expectParameter   map[string]any
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
				Data:     []byte(`{"customer_id": "58e7502c-a770-11ed-9b86-7fabe2dba847", "name": "test name", "detail": "test detail", "engine_model": "openai.gpt-5", "parameter": {"key1": "val1"}, "engine_key": "test engine key", "init_prompt": "test init prompt", "tts_type": "elevenlabs", "tts_voice_id": "test-voice-id", "stt_type": "deepgram"}`),
			},

			responseAI: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("59230ca2-a770-11ed-b5dd-2783587ed477"),
				},
			},

			expectCustomerID:  uuid.FromStringOrNil("58e7502c-a770-11ed-9b86-7fabe2dba847"),
			expectName:        "test name",
			expectDetail:      "test detail",
			expectEngineModel: ai.EngineModelOpenaiGPT5,
			expectParameter: map[string]any{
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
				Data:       []byte(`{"id":"59230ca2-a770-11ed-b5dd-2783587ed477","customer_id":"00000000-0000-0000-0000-000000000000","is_insight_active":false,"rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","direct_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
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
				gomock.Any(), // aiType
				tt.expectEngineModel,
				tt.expectParameter,
				tt.expectEngineKey,
				gomock.Any(), // ragID
				tt.expectInitPrompt,
				tt.expectTTSType,
				tt.expectTTSVoiceID,
				tt.expectSTTType,
				gomock.Any(), // sttLanguage
				gomock.Any(), // toolNames
				gomock.Any(), // vadConfig
				gomock.Any(), // smartTurnEnabled
				gomock.Any(), // autoAICallAuditEnabled
			).Return(tt.responseAI, nil)
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
				Data:       []byte(`{"id":"de740384-a770-11ed-afab-5f9c8a447889","customer_id":"00000000-0000-0000-0000-000000000000","is_insight_active":false,"rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","direct_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
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
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
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
				Data:       []byte(`{"id":"de99e522-a770-11ed-a0ab-5b39ee2db203","customer_id":"00000000-0000-0000-0000-000000000000","is_insight_active":false,"rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","direct_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
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
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
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
		expectEngineModel ai.EngineModel
		expectParameter   map[string]any
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
				Data:     []byte(`{"name":"new name","detail":"new detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"new prompt","tts_type":"cartesia","tts_voice_id":"new-voice-id","stt_type":"deepgram"}`),
			},

			responseAI: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
				},
			},

			expectID:          uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
			expectName:        "new name",
			expectDetail:      "new detail",
			expectEngineModel: ai.EngineModelOpenaiGPT5,
			expectParameter: map[string]any{
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
				Data:       []byte(`{"id":"fa4d3b6a-f82f-11ed-9176-d32f5705e10c","customer_id":"00000000-0000-0000-0000-000000000000","is_insight_active":false,"rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","direct_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
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
				gomock.Any(), // aiType
				tt.expectEngineModel,
				tt.expectParameter,
				tt.expectEngineKey,
				gomock.Any(), // ragID
				tt.expectInitPrompt,
				tt.expectTTSType,
				tt.expectTTSVoiceID,
				tt.expectSTTType,
				gomock.Any(), // sttLanguage
				gomock.Any(), // toolNames
				gomock.Any(), // vadConfig
				gomock.Any(), // smartTurnEnabled
				gomock.Any(), // autoAICallAuditEnabled
			).Return(tt.responseAI, nil)
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

func Test_processV1AIsIDParticipantsGet(t *testing.T) {
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	now := time.Now().UTC()

	tests := []struct {
		name      string
		request   *sock.Request
		mockSetup func(mockParticipant *participanthandler.MockParticipantHandler)
		expectRes *sock.Response
	}{
		{
			name: "returns participants list",
			request: &sock.Request{
				URI:    "/v1/ais/22222222-2222-2222-2222-222222222222/participants?page_size=5",
				Method: sock.RequestMethodGet,
			},
			mockSetup: func(mockParticipant *participanthandler.MockParticipantHandler) {
				mockParticipant.EXPECT().ListByAIID(gomock.Any(), aiID, uint64(5), "").Return([]*participant.Participant{
					{AIID: aiID, AIcallID: aicallID, TMCreate: &now},
				}, nil).Times(1)
			},
			expectRes: &sock.Response{StatusCode: 200, DataType: "application/json"},
		},
		{
			name: "returns 404 on invalid UUID (no route match)",
			request: &sock.Request{
				URI:    "/v1/ais/bad-uuid/participants",
				Method: sock.RequestMethodGet,
			},
			mockSetup: func(mockParticipant *participanthandler.MockParticipantHandler) {},
			expectRes: &sock.Response{StatusCode: 404},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockParticipant := participanthandler.NewMockParticipantHandler(mc)
			tt.mockSetup(mockParticipant)

			h := &listenHandler{
				sockHandler:        mockSock,
				participantHandler: mockParticipant,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if res.StatusCode != tt.expectRes.StatusCode {
				t.Fatalf("expected status %d, got %d", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}

func Test_processV1AIsIDDirectHashRegenerate_errorMapping(t *testing.T) {
	tests := []struct {
		name         string
		request      *sock.Request
		handlerErr   error
		expectStatus int
	}{
		{
			name: "dbhandler.ErrNotFound maps to 404",
			request: &sock.Request{
				URI:    "/v1/ais/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/direct-hash-regenerate",
				Method: sock.RequestMethodPost,
			},
			handlerErr:   dbhandler.ErrNotFound,
			expectStatus: 404,
		},
		{
			name: "generic error maps to 500",
			request: &sock.Request{
				URI:    "/v1/ais/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/direct-hash-regenerate",
				Method: sock.RequestMethodPost,
			},
			handlerErr:   fmt.Errorf("boom"),
			expectStatus: 500,
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

			mockAI.EXPECT().DirectHashRegenerate(gomock.Any(), gomock.Any()).Return(nil, tt.handlerErr)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

// Test_processV1AIsIDPut_McpServerIDsEmptyArrayClears pins the fix for a
// bug found during PR review: a PUT body with "mcp_server_ids": []
// unmarshals into a non-nil *[]uuid.UUID pointing at an empty slice --
// distinct from the field being omitted (nil pointer, leave untouched) --
// and must reach ValidateMcpServerIDs/UpdateMcpServerIDs with an empty
// (not nil) []uuid.UUID, actually clearing the AI's whitelist.
func Test_processV1AIsIDPut_McpServerIDsEmptyArrayClears(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)

	h := &listenHandler{
		sockHandler: mockSock,
		aiHandler:   mockAI,
	}

	id := uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203")
	customerID := uuid.FromStringOrNil("24676972-7f49-11ec-bc89-b7d33e9d3ea8")

	req := &sock.Request{
		URI:    "/v1/ais/" + id.String(),
		Method: sock.RequestMethodPut,
		Data:   []byte(`{"mcp_server_ids":[]}`),
	}

	preUpdate := &ai.AI{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
	}

	mockAI.EXPECT().Update(
		gomock.Any(), id, "", "", ai.Type(""), ai.EngineModel(""), map[string]any(nil), "",
		uuid.Nil, "", ai.TTSType(""), "", ai.STTType(""), "", []tool.ToolName(nil), (*ai.VADConfig)(nil), false, false,
	).Return(preUpdate, nil)

	mockAI.EXPECT().ValidateMcpServerIDs(gomock.Any(), customerID, []uuid.UUID{}).Return(nil)
	mockAI.EXPECT().UpdateMcpServerIDs(gomock.Any(), id, []uuid.UUID{}).Return(preUpdate, nil)

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d (body: %s)", res.StatusCode, res.Data)
	}
}

// Test_processV1AIsIDPut_McpServerIDsOmittedLeavesUntouched is the
// counterpart: omitting mcp_server_ids entirely must not call
// ValidateMcpServerIDs/UpdateMcpServerIDs at all (gomock's strict
// controller fails the test if either is called with no EXPECT() set).
func Test_processV1AIsIDPut_McpServerIDsOmittedLeavesUntouched(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)

	h := &listenHandler{
		sockHandler: mockSock,
		aiHandler:   mockAI,
	}

	id := uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203")

	req := &sock.Request{
		URI:    "/v1/ais/" + id.String(),
		Method: sock.RequestMethodPut,
		Data:   []byte(`{"name":"renamed"}`),
	}

	mockAI.EXPECT().Update(
		gomock.Any(), id, "renamed", "", ai.Type(""), ai.EngineModel(""), map[string]any(nil), "",
		uuid.Nil, "", ai.TTSType(""), "", ai.STTType(""), "", []tool.ToolName(nil), (*ai.VADConfig)(nil), false, false,
	).Return(&ai.AI{Identity: identity.Identity{ID: id}}, nil)

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d (body: %s)", res.StatusCode, res.Data)
	}
}

// Test_processV1AIsIDPut_McpServerIDsInvalidReturns400 pins the fix for a
// bug found during CPO review: ValidateMcpServerIDs used to return a plain
// fmt.Errorf, which errorResponse() (only maps *cerrors.VoipbinError to a
// non-500 status) turned into an opaque 500 for what is a client-input
// mistake (an invalid or cross-customer mcp_server_id). It must now surface
// as 400.
func Test_processV1AIsIDPut_McpServerIDsInvalidReturns400(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)

	h := &listenHandler{
		sockHandler: mockSock,
		aiHandler:   mockAI,
	}

	id := uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203")
	customerID := uuid.FromStringOrNil("24676972-7f49-11ec-bc89-b7d33e9d3ea8")
	invalidServerID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	req := &sock.Request{
		URI:    "/v1/ais/" + id.String(),
		Method: sock.RequestMethodPut,
		Data:   []byte(`{"mcp_server_ids":["` + invalidServerID.String() + `"]}`),
	}

	preUpdate := &ai.AI{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
	}

	mockAI.EXPECT().Update(
		gomock.Any(), id, "", "", ai.Type(""), ai.EngineModel(""), map[string]any(nil), "",
		uuid.Nil, "", ai.TTSType(""), "", ai.STTType(""), "", []tool.ToolName(nil), (*ai.VADConfig)(nil), false, false,
	).Return(preUpdate, nil)

	mockAI.EXPECT().ValidateMcpServerIDs(gomock.Any(), customerID, []uuid.UUID{invalidServerID}).Return(
		cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, "INVALID_MCP_SERVER_ID", "mcp_server_id "+invalidServerID.String()+" is not accessible"),
	)

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.StatusCode != 400 {
		t.Fatalf("expected status 400, got %d (body: %s)", res.StatusCode, res.Data)
	}
}
