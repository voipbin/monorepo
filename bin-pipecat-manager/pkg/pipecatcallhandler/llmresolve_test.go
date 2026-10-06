package pipecatcallhandler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amteam "monorepo/bin-ai-manager/models/team"
	aitool "monorepo/bin-ai-manager/models/tool"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_resolveSessionLLM(t *testing.T) {
	tests := []struct {
		name string

		llmType pipecatcall.LLMType
		aiKey   string

		expectRunnerType string
		expectKey        string
		expectErr        bool
	}{
		{"direct catalog keeps type and key", "openai.gpt-5", "customer-key", "openai.gpt-5", "customer-key", false},
		{"direct catalog without key", "gemini.gemini-2.5-flash", "", "gemini.gemini-2.5-flash", "", false},
		{"openrouter catalog rewrites type and blanks key", "anthropic.claude-haiku-4.5", "customer-key", "platform_openrouter.anthropic/claude-haiku-4.5", "", false},
		{"openrouter catalog meta slug", "meta.llama-3.3-70b-instruct", "customer-key", "platform_openrouter.meta-llama/llama-3.3-70b-instruct", "", false},
		{"passthrough keeps key", "openai.gpt-4o", "customer-key", "openai.gpt-4o", "customer-key", false},
		{"rejected legacy anthropic", "anthropic.claude-opus-4", "customer-key", "", "", true},
		{"custom openrouter forwards the key", "openrouter.meta-llama/llama-3-70b", "dummy-new-key", "openrouter.meta-llama/llama-3-70b", "dummy-new-key", false},
		{"custom openrouter keeps dotted id", "openrouter.mistralai/mistral-medium-3.1", "dummy-new-key", "openrouter.mistralai/mistral-medium-3.1", "dummy-new-key", false},
		{"custom openrouter trims the key", "openrouter.vendor/model-a", "  dummy-new-key  ", "openrouter.vendor/model-a", "dummy-new-key", false},
		{"custom openrouter empty key fails", "openrouter.vendor/model-a", "", "", "", true},
		{"custom openrouter blank key fails", "openrouter.vendor/model-a", " \t\n ", "", "", true},
		{"rejected custom without slash", "openrouter.dummy-key-not-real", "dummy-new-key", "", "", true},
		{"rejected custom router id", "openrouter.openrouter/auto", "dummy-new-key", "", "", true},
		{"rejected custom variant suffix", "openrouter.vendor/model:free", "dummy-new-key", "", "", true},
		{"rejected custom catalog entry id", "custom.openrouter", "dummy-new-key", "", "", true},
		{"rejected internal prefix", "platform_openrouter.x", "customer-key", "", "", true},
		{"rejected empty", "", "customer-key", "", "", true},
		{"rejected unknown", "unknown.x", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runnerType, key, err := resolveSessionLLM(tt.llmType, tt.aiKey)
			if (err != nil) != tt.expectErr {
				t.Fatalf("err = %v, expectErr = %v", err, tt.expectErr)
			}
			if runnerType != tt.expectRunnerType {
				t.Errorf("runnerType = %q, want %q", runnerType, tt.expectRunnerType)
			}
			if key != tt.expectKey {
				t.Errorf("key = %q, want %q", key, tt.expectKey)
			}
		})
	}
}

func Test_resolveSessionLLM_errorTextDoesNotEchoCustomInput(t *testing.T) {
	tests := []struct {
		name string

		llmType pipecatcall.LLMType
		aiKey   string

		expectContains    string
		expectNotContains string
	}{
		{"malformed custom id is not echoed", "openrouter.dummy-key-not-real", "dummy-new-key", "engine model is not available", "dummy-key-not-real"},
		{"upper case custom prefix is not echoed", "OpenRouter.dummy-key-not-real", "dummy-new-key", "engine model is not available", "dummy-key-not-real"},
		{"leading space custom prefix is not echoed", " openrouter.dummy-key-not-real", "dummy-new-key", "engine model is not available", "dummy-key-not-real"},
		{"leading tab custom prefix is not echoed", "\topenrouter.dummy-key-not-real", "dummy-new-key", "engine model is not available", "dummy-key-not-real"},
		{"empty custom key error has no model id", "openrouter.vendor/model-a", "", "custom engine key is empty", "vendor/model-a"},
		{"other rejected input keeps the existing text", "unknown.x", "", "engine model is not available: unknown.x", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := resolveSessionLLM(tt.llmType, tt.aiKey)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("err = %q, want it to contain %q", err.Error(), tt.expectContains)
			}
			if tt.expectNotContains != "" && strings.Contains(err.Error(), tt.expectNotContains) {
				t.Errorf("err = %q must not contain %q", err.Error(), tt.expectNotContains)
			}
		})
	}
}

func Test_classifySessionLLM(t *testing.T) {
	tests := []struct {
		name string

		llmType pipecatcall.LLMType

		expectErr         bool
		expectNotContains string
	}{
		{"custom openrouter passes without a key", "openrouter.vendor/model-a", false, ""},
		{"direct catalog passes", "openai.gpt-5", false, ""},
		{"openrouter catalog passes", "anthropic.claude-haiku-4.5", false, ""},
		{"rejected legacy anthropic", "anthropic.claude-opus-4", true, ""},
		{"rejected custom router id", "openrouter.openrouter/auto", true, "openrouter/auto"},
		{"rejected custom without slash", "openrouter.dummy-key-not-real", true, "dummy-key-not-real"},
		{"rejected internal prefix", "platform_openrouter.x", true, ""},
		{"rejected empty", "", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifySessionLLM(tt.llmType)
			if (err != nil) != tt.expectErr {
				t.Fatalf("err = %v, expectErr = %v", err, tt.expectErr)
			}
			if err != nil && tt.expectNotContains != "" && strings.Contains(err.Error(), tt.expectNotContains) {
				t.Errorf("err = %q must not contain %q", err.Error(), tt.expectNotContains)
			}
		})
	}
}

// Test_Start_rejectedModelDoesNotCreate pins that a Rejected model fails Start
// before h.Create (no DB row): the DB mock has no expectations, so any call fails.
func Test_Start_rejectedModelDoesNotCreate(t *testing.T) {
	for _, model := range []pipecatcall.LLMType{"anthropic.claude-opus-4", "openrouter.openrouter/auto", "openrouter.a/b:free", ""} {
		t.Run(string(model), func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &pipecatcallHandler{db: mockDB}

			res, err := h.Start(context.Background(),
				uuid.FromStringOrNil("11110000-1111-2222-3333-444455556666"),
				uuid.FromStringOrNil("11110000-1111-2222-3333-444455557777"),
				uuid.Nil, pipecatcall.ReferenceTypeAICall, uuid.Nil,
				model, nil, pipecatcall.STTTypeNone, "", pipecatcall.TTSTypeNone, "", "")
			if err == nil || res != nil {
				t.Fatalf("expected error and nil result, got res=%v err=%v", res, err)
			}
		})
	}
}

// Test_Start_customModelPassesClassification pins that the keyless pre-check in
// Start() accepts a custom OpenRouter model (the key only exists later, in the
// run path), so h.Create runs once. An unsupported reference type then ends Start
// before any runner is touched.
func Test_Start_customModelPassesClassification(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &pipecatcallHandler{db: mockDB, notifyHandler: mockNotify, hostID: "host-1"}

	id := uuid.FromStringOrNil("11110000-1111-2222-3333-444455556666")
	created := &pipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: id}}

	mockDB.EXPECT().PipecatcallCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().PipecatcallGet(gomock.Any(), id).Return(created, nil).Times(1)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), pipecatcall.EventTypeCreated, created).Times(1)

	res, err := h.Start(context.Background(),
		id, uuid.FromStringOrNil("11110000-1111-2222-3333-444455557777"),
		uuid.Nil, pipecatcall.ReferenceType("unsupported"), uuid.Nil,
		"openrouter.vendor/model-a", nil, pipecatcall.STTTypeNone, "", pipecatcall.TTSTypeNone, "", "")
	if err == nil || res != nil {
		t.Fatalf("expected the invalid reference type error and nil result, got res=%v err=%v", res, err)
	}
	if !strings.Contains(err.Error(), "invalid reference type") {
		t.Errorf("err = %q, want it to fail on the reference type, not on classification", err.Error())
	}
}

func Test_runGetLLMKey_resolves(t *testing.T) {
	aiID := uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111")
	referenceID := uuid.FromStringOrNil("f6f6f6f6-6666-6666-6666-666666666666")

	tests := []struct {
		name string

		llmType pipecatcall.LLMType
		prepare func(m *requesthandler.MockRequestHandler)

		expectRunnerType string
		expectKey        string
		expectErr        bool
	}{
		{
			name:    "openrouter model blanks the ai key",
			llmType: "anthropic.claude-haiku-4.5",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeAI, AssistanceID: aiID}, nil)
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineModel: "anthropic.claude-haiku-4.5", EngineKey: "customer-key"}, nil)
			},
			expectRunnerType: "platform_openrouter.anthropic/claude-haiku-4.5",
			expectKey:        "",
		},
		{
			name:    "direct model keeps the ai key",
			llmType: "openai.gpt-5",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeAI, AssistanceID: aiID}, nil)
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineModel: "openai.gpt-5", EngineKey: "customer-key"}, nil)
			},
			expectRunnerType: "openai.gpt-5",
			expectKey:        "customer-key",
		},
		{
			name:    "ai lookup failure with direct model proceeds without key",
			llmType: "openai.gpt-5",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("boom"))
			},
			expectRunnerType: "openai.gpt-5",
			expectKey:        "",
		},
		{
			name:    "custom model forwards the trimmed ai key",
			llmType: "openrouter.a/b",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeAI, AssistanceID: aiID}, nil)
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineModel: "openrouter.a/b", EngineKey: "  dummy-old-key  "}, nil)
			},
			expectRunnerType: "openrouter.a/b",
			expectKey:        "dummy-old-key",
		},
		{
			name:    "custom model with ai lookup failure has no key and errors",
			llmType: "openrouter.a/b",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("boom"))
			},
			expectErr: true,
		},
		{
			name:    "custom model with an empty stored key errors",
			llmType: "openrouter.a/b",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeAI, AssistanceID: aiID}, nil)
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineModel: "openrouter.a/b", EngineKey: ""}, nil)
			},
			expectErr: true,
		},
		{
			name:    "custom session with a live ai moved to openai rejects the new key",
			llmType: "openrouter.a/b",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeAI, AssistanceID: aiID}, nil)
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineModel: "openai.gpt-5", EngineKey: "dummy-new-key"}, nil)
			},
			expectErr: true,
		},
		{
			name:    "openai session with a live ai moved to custom openrouter rejects the new key",
			llmType: "openai.gpt-5",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeAI, AssistanceID: aiID}, nil)
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineModel: "openrouter.a/b", EngineKey: "dummy-new-key"}, nil)
			},
			expectErr: true,
		},
		{
			name:    "model change within the same vendor keeps working",
			llmType: "openrouter.a/b",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeAI, AssistanceID: aiID}, nil)
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineModel: "openrouter.c/d", EngineKey: "dummy-old-key"}, nil)
			},
			expectRunnerType: "openrouter.a/b",
			expectKey:        "dummy-old-key",
		},
		{
			name:    "routed session never forwards a key so a live vendor change is not blocked",
			llmType: "anthropic.claude-haiku-4.5",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeAI, AssistanceID: aiID}, nil)
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineModel: "openai.gpt-5", EngineKey: "dummy-new-key"}, nil)
			},
			expectRunnerType: "platform_openrouter.anthropic/claude-haiku-4.5",
			expectKey:        "",
		},
		{
			name:    "ai lookup failure cannot bypass rejection of stored openrouter.x",
			llmType: "openrouter.x",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("boom"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			tt.prepare(mockReq)
			h := &pipecatcallHandler{requestHandler: mockReq}

			pc := &pipecatcall.Pipecatcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("e5e5e5e5-5555-5555-5555-555555555555")},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
				LLMType:       tt.llmType,
			}

			runnerType, key, err := h.runGetLLMKey(context.Background(), pc)
			if (err != nil) != tt.expectErr {
				t.Fatalf("err = %v, expectErr = %v", err, tt.expectErr)
			}
			if runnerType != tt.expectRunnerType || key != tt.expectKey {
				t.Errorf("got (%q, %q), want (%q, %q)", runnerType, key, tt.expectRunnerType, tt.expectKey)
			}
		})
	}
}

// Test_startReferenceTypeCall_rejectedModel: a rejected model, or a custom model
// (which has no key on this path because the call reference never looks the AI
// up), errors before any session is created or any Asterisk resource is started.
func Test_startReferenceTypeCall_rejectedModel(t *testing.T) {
	tests := []struct {
		name string

		llmType pipecatcall.LLMType
	}{
		{"rejected legacy model", "anthropic.claude-opus-4"},
		{"custom model has no key on the call reference", "openrouter.vendor/model-a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &pipecatcallHandler{
				requestHandler:        mockReq,
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
			}
			referenceID := uuid.FromStringOrNil("b2c3d4e5-1111-2222-3333-444455556666")
			pc := &pipecatcall.Pipecatcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666")},
				ReferenceType: pipecatcall.ReferenceTypeCall,
				ReferenceID:   referenceID,
				LLMType:       tt.llmType,
			}
			mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).Return(&cmcall.Call{}, nil)

			if err := h.startReferenceTypeCall(context.Background(), pc); err == nil {
				t.Fatal("expected error")
			}
			if len(h.mapPipecatcallSession) != 0 {
				t.Errorf("no session may be created")
			}
		})
	}
}

// Test_startReferenceTypeAIcall_runnerHandoff drives the AI-call (non-call reference)
// start path through to pythonRunner.Start and asserts what the runner receives.
func Test_startReferenceTypeAIcall_runnerHandoff(t *testing.T) {
	aicallID := uuid.FromStringOrNil("11111111-aaaa-2222-3333-444455556666")
	aiID := uuid.FromStringOrNil("22222222-aaaa-2222-3333-444455556666")
	pcID := uuid.FromStringOrNil("33333333-aaaa-2222-3333-444455556666")

	tests := []struct {
		name string

		llmType   pipecatcall.LLMType
		aiModel   amai.EngineModel // live AI model; defaults to llmType
		aiGetErr  error
		expectErr bool

		expectRunnerType string
		expectKey        string
	}{
		{name: "openrouter model: slug type, blank key even though the ai has a key", llmType: "anthropic.claude-sonnet-4.5", expectRunnerType: "platform_openrouter.anthropic/claude-sonnet-4.5", expectKey: ""},
		{name: "direct model keeps the key", llmType: "openai.gpt-5", expectRunnerType: "openai.gpt-5", expectKey: "customer-key"},
		{name: "passthrough keeps the key", llmType: "openai.gpt-4o", expectRunnerType: "openai.gpt-4o", expectKey: "customer-key"},
		{name: "direct model with ai lookup failure still proceeds (warn and proceed)", llmType: "openai.gpt-5", aiGetErr: fmt.Errorf("boom"), expectRunnerType: "openai.gpt-5", expectKey: ""},
		{name: "openrouter model with ai lookup failure still resolves to the slug type", llmType: "meta.llama-4-maverick", aiGetErr: fmt.Errorf("boom"), expectRunnerType: "platform_openrouter.meta-llama/llama-4-maverick", expectKey: ""},
		{name: "custom model forwards the customer key", llmType: "openrouter.vendor/model-a", expectRunnerType: "openrouter.vendor/model-a", expectKey: "customer-key"},
		{name: "custom model with ai lookup failure errors", llmType: "openrouter.vendor/model-a", aiGetErr: fmt.Errorf("boom"), expectErr: true},
		{name: "stored raw openrouter.x with ai lookup failure errors", llmType: "openrouter.x", aiGetErr: fmt.Errorf("boom"), expectErr: true},
		{name: "rejected legacy model errors", llmType: "anthropic.claude-opus-4", expectErr: true},
		{name: "custom session with a live ai moved to openai errors", llmType: "openrouter.vendor/model-a", aiModel: "openai.gpt-5", expectErr: true},
		{name: "openai session with a live ai moved to custom errors", llmType: "openai.gpt-5", aiModel: "openrouter.vendor/model-a", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTool := toolhandler.NewMockToolHandler(mc)
			mockPython := NewMockPythonRunner(mc)
			h := &pipecatcallHandler{
				requestHandler:        mockReq,
				toolHandler:           mockTool,
				pythonRunner:          mockPython,
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
				muPipecatcallSession:  sync.Mutex{},
			}

			pc := &pipecatcall.Pipecatcall{
				Identity:      commonidentity.Identity{ID: pcID},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   aicallID,
				LLMType:       tt.llmType,
			}
			ac := &amaicall.AIcall{
				Identity:       commonidentity.Identity{ID: aicallID},
				ReferenceType:  amaicall.ReferenceTypeTask,
				AssistanceType: amaicall.AssistanceTypeAI,
				AssistanceID:   aiID,
			}

			mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).AnyTimes()
			if tt.aiGetErr != nil {
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(nil, tt.aiGetErr).AnyTimes()
			} else {
				aiModel := tt.aiModel
				if aiModel == "" {
					aiModel = amai.EngineModel(tt.llmType)
				}
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
					Identity:    commonidentity.Identity{ID: aiID},
					Type:        amai.TypeNormal,
					EngineModel: aiModel,
					EngineKey:   "customer-key",
				}, nil).AnyTimes()
			}
			mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return([]aitool.Tool{}).AnyTimes()
			mockReq.EXPECT().AIV1AIcallToolList(gomock.Any(), aicallID).Return(nil, nil).AnyTimes()

			type handoff struct{ runnerType, key string }
			started := make(chan handoff, 1)
			if !tt.expectErr {
				mockPython.EXPECT().Start(
					gomock.Any(), pcID, gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).DoAndReturn(func(
					_ context.Context, _ uuid.UUID, llmType string, llmKey string, _ any,
					_ string, _ string, _ string, _ string, _ string,
					_ []aitool.Tool, _ any, _ any, _ bool,
				) error {
					started <- handoff{llmType, llmKey}
					return nil
				}).Times(1)
			}

			err := h.startReferenceTypeAIcall(context.Background(), pc)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if len(h.mapPipecatcallSession) != 0 {
					t.Errorf("no session may be created for a rejected model")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			select {
			case got := <-started:
				if got.runnerType != tt.expectRunnerType || got.key != tt.expectKey {
					t.Errorf("runner got (%q, %q), want (%q, %q)", got.runnerType, got.key, tt.expectRunnerType, tt.expectKey)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("python runner was never started")
			}

			se, errSe := h.SessionGet(pcID)
			if errSe != nil {
				t.Fatalf("session missing: %v", errSe)
			}
			se.Cancel()
		})
	}
}

// Test_teamSession_skipsLiveVendorCheck: a team pipeline uses the per-member values from
// resolveTeamForPython. pc.LLMType is the start member's model while the key comes from
// the current member, so a vendor mismatch between members is a normal hand-off.
func Test_teamSession_skipsLiveVendorCheck(t *testing.T) {
	teamID := uuid.FromStringOrNil("aaaaaaaa-1111-0000-0000-000000000001")
	m1ID := uuid.FromStringOrNil("bbbbbbbb-1111-0000-0000-000000000001")
	m2ID := uuid.FromStringOrNil("bbbbbbbb-1111-0000-0000-000000000002")
	ai1ID := uuid.FromStringOrNil("dddddddd-1111-0000-0000-000000000001")
	ai2ID := uuid.FromStringOrNil("dddddddd-1111-0000-0000-000000000002")
	aicallID := uuid.FromStringOrNil("11111111-bbbb-2222-3333-444455556666")
	pcID := uuid.FromStringOrNil("33333333-bbbb-2222-3333-444455556666")

	tests := []struct {
		name string

		startModel   amai.EngineModel // member 1, the pipecatcall llm_type
		currentModel amai.EngineModel // member 2, the current member whose key is read

		expectRunnerType string
		expectKey        string
	}{
		{name: "members with different direct vendors", startModel: "openai.gpt-5", currentModel: "anthropic.claude-haiku-4.5", expectRunnerType: "openai.gpt-5", expectKey: "dummy-key"},
		{name: "start openai and current member custom openrouter", startModel: "openai.gpt-5", currentModel: "openrouter.vendor/model-a", expectRunnerType: "openai.gpt-5", expectKey: "dummy-key"},
		{name: "start custom openrouter and current member openai", startModel: "openrouter.vendor/model-a", currentModel: "openai.gpt-5", expectRunnerType: "openrouter.vendor/model-a", expectKey: "dummy-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTool := toolhandler.NewMockToolHandler(mc)
			mockPython := NewMockPythonRunner(mc)
			h := &pipecatcallHandler{
				requestHandler:        mockReq,
				toolHandler:           mockTool,
				pythonRunner:          mockPython,
				mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
			}

			ac := &amaicall.AIcall{
				Identity:        commonidentity.Identity{ID: aicallID},
				ReferenceType:   amaicall.ReferenceTypeTask,
				AssistanceType:  amaicall.AssistanceTypeTeam,
				AssistanceID:    teamID,
				CurrentMemberID: m2ID,
			}
			mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).AnyTimes()
			mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amteam.Team{
				Identity:      commonidentity.Identity{ID: teamID},
				StartMemberID: m1ID,
				Members: []amteam.Member{
					{ID: m1ID, Name: "one", AIID: ai1ID},
					{ID: m2ID, Name: "two", AIID: ai2ID},
				},
			}, nil).AnyTimes()
			mockReq.EXPECT().AIV1AIGet(gomock.Any(), ai1ID).Return(&amai.AI{EngineModel: tt.startModel, EngineKey: "dummy-key"}, nil).AnyTimes()
			mockReq.EXPECT().AIV1AIGet(gomock.Any(), ai2ID).Return(&amai.AI{EngineModel: tt.currentModel, EngineKey: "dummy-key"}, nil).AnyTimes()
			mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return([]aitool.Tool{}).AnyTimes()
			mockReq.EXPECT().AIV1AIcallToolList(gomock.Any(), aicallID).Return(nil, nil).AnyTimes()

			pc := &pipecatcall.Pipecatcall{
				Identity:      commonidentity.Identity{ID: pcID},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   aicallID,
				LLMType:       pipecatcall.LLMType(tt.startModel),
			}

			// runGetLLMKey path
			runnerType, key, err := h.runGetLLMKey(context.Background(), pc)
			if err != nil {
				t.Fatalf("runGetLLMKey: unexpected error: %v", err)
			}
			if runnerType != tt.expectRunnerType || key != tt.expectKey {
				t.Errorf("runGetLLMKey got (%q, %q), want (%q, %q)", runnerType, key, tt.expectRunnerType, tt.expectKey)
			}

			// start path
			started := make(chan struct{}, 1)
			mockPython.EXPECT().Start(
				gomock.Any(), pcID, gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			).DoAndReturn(func(
				_ context.Context, _ uuid.UUID, _ string, _ string, _ any,
				_ string, _ string, _ string, _ string, _ string,
				_ []aitool.Tool, _ any, _ any, _ bool,
			) error {
				started <- struct{}{}
				return nil
			}).Times(1)

			if errStart := h.startReferenceTypeAIcall(context.Background(), pc); errStart != nil {
				t.Fatalf("startReferenceTypeAIcall: unexpected error: %v", errStart)
			}
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("python runner was never started")
			}
			if se, errSe := h.SessionGet(pcID); errSe == nil {
				se.Cancel()
			}
		})
	}
}

func Test_engineVendor(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		expect string
	}{
		{"lower case", "openai.gpt-5", "openai"},
		{"upper case", "OpenAI.gpt-5", "openai"},
		{"leading and trailing whitespace", "  openai.gpt-5  ", "openai"},
		{"custom openrouter upper case with whitespace", " OpenRouter.vendor/model ", "openrouter"},
		{"no dot", "openai", "openai"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engineVendor(tt.model); got != tt.expect {
				t.Errorf("engineVendor(%q) = %q, want %q", tt.model, got, tt.expect)
			}
		})
	}
}

func Test_checkLiveEngineVendor_caseAndWhitespace(t *testing.T) {
	live := &amai.AI{EngineModel: "OpenAI.gpt-5"}
	if err := checkLiveEngineVendor(" openai.gpt-4o ", live, "dummy-key"); err != nil {
		t.Errorf("same vendor differing only in case and whitespace must pass, got %v", err)
	}
}
