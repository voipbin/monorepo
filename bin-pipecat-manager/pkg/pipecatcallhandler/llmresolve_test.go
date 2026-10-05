package pipecatcallhandler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	aitool "monorepo/bin-ai-manager/models/tool"
	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
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
		{"rejected raw openrouter", "openrouter.meta-llama/llama-3-70b", "customer-key", "", "", true},
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

// Test_Start_rejectedModelDoesNotCreate pins that a Rejected model fails Start
// before h.Create (no DB row): the DB mock has no expectations, so any call fails.
func Test_Start_rejectedModelDoesNotCreate(t *testing.T) {
	for _, model := range []pipecatcall.LLMType{"anthropic.claude-opus-4", "openrouter.meta-llama/llama-3-70b", ""} {
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
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineKey: "customer-key"}, nil)
			},
			expectRunnerType: "platform_openrouter.anthropic/claude-haiku-4.5",
			expectKey:        "",
		},
		{
			name:    "direct model keeps the ai key",
			llmType: "openai.gpt-5",
			prepare: func(m *requesthandler.MockRequestHandler) {
				m.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeAI, AssistanceID: aiID}, nil)
				m.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{EngineKey: "customer-key"}, nil)
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

// Test_startReferenceTypeCall_rejectedModel: rejected model errors before any
// session is created or any Asterisk resource is started.
func Test_startReferenceTypeCall_rejectedModel(t *testing.T) {
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
		LLMType:       "openrouter.meta-llama/llama-3-70b",
	}
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).Return(&cmcall.Call{}, nil)

	if err := h.startReferenceTypeCall(context.Background(), pc); err == nil {
		t.Fatal("expected error for rejected model")
	}
	if len(h.mapPipecatcallSession) != 0 {
		t.Errorf("no session may be created for a rejected model")
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
		{name: "stored raw openrouter.x with ai lookup failure errors", llmType: "openrouter.x", aiGetErr: fmt.Errorf("boom"), expectErr: true},
		{name: "rejected legacy model errors", llmType: "anthropic.claude-opus-4", expectErr: true},
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
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
					Identity:  commonidentity.Identity{ID: aiID},
					Type:      amai.TypeNormal,
					EngineKey: "customer-key",
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
