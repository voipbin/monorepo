package summaryhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/mock/gomock"
)

// testModel/testReasoningEffort are the injected values used across summaryhandler
// tests. They stand in for the config-provided model and reasoning_effort so the
// request-building assertions do not depend on any package-level default.
const (
	testModel           = "gemini-3.8-flash"
	testReasoningEffort = "none"
)

func Test_contentGet(t *testing.T) {

	tests := []struct {
		name string

		activeflowID   uuid.UUID
		referenceType  summary.ReferenceType
		transcripts    []tmtranscript.Transcript
		outputLanguage string

		responseVariable *fmvariable.Variable
		responseOpenai   *openai.ChatCompletionResponse

		expectedRequestContent RequestContent
		expectedRes            string
	}{
		{
			name: "reference type call",

			activeflowID:  uuid.FromStringOrNil("77b6f188-0b96-11f0-8f7a-e3ffa3666724"),
			referenceType: summary.ReferenceTypeCall,
			transcripts: []tmtranscript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("77e95178-0b96-11f0-afe8-f7c1026e2d7c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("78171978-0b96-11f0-930b-c3391e420f82"),
					},
				},
			},
			outputLanguage: "en-US",

			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					"key1": "value1",
				},
			},
			responseOpenai: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "response content",
						},
					},
				},
			},

			expectedRequestContent: RequestContent{
				Prompt:         defaultSummaryGeneratePrompt,
				ReferenceType:  "call",
				OutputLanguage: "en-US",
				Transcripts: []tmtranscript.Transcript{
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("77e95178-0b96-11f0-afe8-f7c1026e2d7c"),
						},
					},
					{
						Identity: commonidentity.Identity{
							ID: uuid.FromStringOrNil("78171978-0b96-11f0-930b-c3391e420f82"),
						},
					},
				},
				Variables: map[string]string{
					"key1": "value1",
				},
			},
			expectedRes: "response content",
		},
		{
			name: "reference type conference",

			activeflowID:   uuid.FromStringOrNil("77b6f188-0b96-11f0-8f7a-e3ffa3666724"),
			referenceType:  summary.ReferenceTypeConference,
			transcripts:    []tmtranscript.Transcript{},
			outputLanguage: "ko-KR",

			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{},
			},
			responseOpenai: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "response content",
						},
					},
				},
			},

			expectedRequestContent: RequestContent{
				Prompt:         defaultSummaryGeneratePrompt,
				ReferenceType:  "conference",
				OutputLanguage: "ko-KR",
				Transcripts:    []tmtranscript.Transcript{},
				Variables:      map[string]string{},
			},
			expectedRes: "response content",
		},
		{
			name: "reference type recording",

			activeflowID:   uuid.FromStringOrNil("77b6f188-0b96-11f0-8f7a-e3ffa3666724"),
			referenceType:  summary.ReferenceTypeRecording,
			transcripts:    []tmtranscript.Transcript{},
			outputLanguage: "ko-KR",

			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{},
			},
			responseOpenai: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "response content",
						},
					},
				},
			},

			expectedRequestContent: RequestContent{
				Prompt:         defaultSummaryGeneratePrompt,
				ReferenceType:  "recording",
				OutputLanguage: "ko-KR",
				Transcripts:    []tmtranscript.Transcript{},
				Variables:      map[string]string{},
			},
			expectedRes: "response content",
		},
		{
			name: "reference type transcribe",

			activeflowID:   uuid.FromStringOrNil("77b6f188-0b96-11f0-8f7a-e3ffa3666724"),
			referenceType:  summary.ReferenceTypeTranscribe,
			transcripts:    []tmtranscript.Transcript{},
			outputLanguage: "en-US",

			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{},
			},
			responseOpenai: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "response content",
						},
					},
				},
			},

			expectedRequestContent: RequestContent{
				Prompt:         defaultSummaryGeneratePrompt,
				ReferenceType:  "transcribe",
				OutputLanguage: "en-US",
				Transcripts:    []tmtranscript.Transcript{},
				Variables:      map[string]string{},
			},
			expectedRes: "response content",
		},
		{
			name: "reference type none defaults to empty string",

			activeflowID:  uuid.FromStringOrNil("77b6f188-0b96-11f0-8f7a-e3ffa3666724"),
			referenceType: summary.ReferenceTypeNone,
			transcripts:   []tmtranscript.Transcript{},

			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{},
			},
			responseOpenai: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "response content",
						},
					},
				},
			},

			expectedRequestContent: RequestContent{
				Prompt:         defaultSummaryGeneratePrompt,
				ReferenceType:  "",
				OutputLanguage: "en-US",
				Transcripts:    []tmtranscript.Transcript{},
				Variables:      map[string]string{},
			},
			expectedRes: "response content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,

				engineOpenaiHandler: mockOpenai,
				model:               testModel,
				reasoningEffort:     testReasoningEffort,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVariable, nil)

			effectiveLang := tt.outputLanguage
			if effectiveLang == "" {
				effectiveLang = defaultOutputLanguage
			}

			tmpContent, err := json.Marshal(tt.expectedRequestContent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			tmpRequestContent := &openai.ChatCompletionRequest{
				Model:           testModel,
				Temperature:     0.2, // literal, not summaryTemperature: guards against the constant being set to 0 (VOIP-1536)
				ReasoningEffort: "none",
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: fmt.Sprintf(languageSystemPromptFmt, effectiveLang),
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: string(tmpContent),
					},
				},
			}
			mockOpenai.EXPECT().Send(ctx, tmpRequestContent).Return(tt.responseOpenai, nil)

			// Note: the shared fixtures use "response content" (16 runes of prose),
			// which is below languageVerifyMinProse, so the second-pass verifier is
			// skipped even for non-English targets. The verification harness itself
			// is covered by Test_contentGet_verificationHarness.

			res, err := h.contentGet(ctx, tt.activeflowID, tt.referenceType, tt.transcripts, tt.outputLanguage)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

// Test_contentGet_verificationHarness exercises the second-pass output-language
// verification harness (design §7). Generation uses Send, verification uses
// SendOnce; gomock.InOrder pins the call sequence so the response-to-call
// mapping is deterministic.
func Test_contentGet_verificationHarness(t *testing.T) {
	activeflowID := uuid.FromStringOrNil("77b6f188-0b96-11f0-8f7a-e3ffa3666724")
	koProse := "Call Type:\n- 통화\n\nKey Discussion Points:\n- 고객이 환불을 요청했고 상담원이 절차를 안내했습니다."
	enProse := "Call Type:\n- Call\n\nKey Discussion Points:\n- The customer asked for a refund and the agent explained the process."

	genResp := func(content string) *openai.ChatCompletionResponse {
		return &openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: content}}},
		}
	}
	verifyResp := func(answer string) *openai.ChatCompletionResponse {
		return &openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: answer}}},
		}
	}

	// genDo returns a DoAndReturn for a first-pass generation Send that asserts
	// the request uses the injected model and a [system, user] message pair whose
	// system message pins the requested output language by value, then returns content.
	genDo := func(lang, content string) func(context.Context, *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
		return func(_ context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
			if req.Model != testModel {
				t.Errorf("generate: expect model %q, got %q", testModel, req.Model)
			}
			// literal "none", not h.reasoningEffort: guards against the injected value being dropped (VOIP-1537)
			if req.ReasoningEffort != "none" {
				t.Errorf("generate: expect reasoning_effort none, got %q", req.ReasoningEffort)
			}
			// literal 0.2, not summaryTemperature: guards against the constant being set to 0 (VOIP-1536)
			if req.Temperature != 0.2 {
				t.Errorf("generate: expect temperature 0.2, got %v", req.Temperature)
			}
			if len(req.Messages) != 2 {
				t.Errorf("generate: expect 2 messages, got %d", len(req.Messages))
			} else {
				if req.Messages[0].Role != openai.ChatMessageRoleSystem {
					t.Errorf("generate: expect first message system, got %q", req.Messages[0].Role)
				}
				if !strings.Contains(req.Messages[0].Content, lang) {
					t.Errorf("generate: expect system message to contain %q, got %q", lang, req.Messages[0].Content)
				}
				if req.Messages[1].Role != openai.ChatMessageRoleUser {
					t.Errorf("generate: expect second message user, got %q", req.Messages[1].Role)
				}
			}
			return genResp(content), nil
		}
	}

	// verifyDo returns a DoAndReturn for a second-pass verification SendOnce that
	// asserts the verifier uses the injected model and a single user message built
	// from languageVerifyPrompt (containing the "language detector" prompt, the
	// requested BCP47 code, and the sampled prose), then returns answer. This is
	// what turns the harness assertions from "a SendOnce happened" into "the right
	// verify payload was sent" (catches R4-* mutations on model/prompt).
	verifyDo := func(lang, sample, answer string) func(context.Context, *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
		return func(_ context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
			if req.Model != testModel {
				t.Errorf("verify: expect model %q, got %q", testModel, req.Model)
			}
			// literal "none", not h.reasoningEffort: guards against the injected value being dropped (VOIP-1537)
			if req.ReasoningEffort != "none" {
				t.Errorf("verify: expect reasoning_effort none, got %q", req.ReasoningEffort)
			}
			// literal 0.2, not summaryTemperature: guards against the constant being set to 0 (VOIP-1536)
			if req.Temperature != 0.2 {
				t.Errorf("verify: expect temperature 0.2, got %v", req.Temperature)
			}
			if len(req.Messages) != 1 {
				t.Errorf("verify: expect 1 message, got %d", len(req.Messages))
			} else {
				if req.Messages[0].Role != openai.ChatMessageRoleUser {
					t.Errorf("verify: expect user role, got %q", req.Messages[0].Role)
				}
				want := fmt.Sprintf(languageVerifyPrompt, lang, sample)
				if req.Messages[0].Content != want {
					t.Errorf("verify: expect prompt %q, got %q", want, req.Messages[0].Content)
				}
				if !strings.Contains(req.Messages[0].Content, "language detector") {
					t.Errorf("verify: expect prompt to contain languageVerifyPrompt fragment, got %q", req.Messages[0].Content)
				}
			}
			return verifyResp(answer), nil
		}
	}

	t.Run("verification passes on first attempt", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
		h := summaryHandler{reqHandler: mockReq, engineOpenaiHandler: mockOpenai, model: testModel, reasoningEffort: testReasoningEffort}
		ctx := context.Background()

		mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{}}, nil)
		gomock.InOrder(
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", koProse)),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", koProse, "yes")),
		)

		res, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeCall, []tmtranscript.Transcript{}, "ko-KR")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != koProse {
			t.Errorf("expect: %q, got: %q", koProse, res)
		}
	})

	t.Run("mismatch then regenerate passes", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
		h := summaryHandler{reqHandler: mockReq, engineOpenaiHandler: mockOpenai, model: testModel, reasoningEffort: testReasoningEffort}
		ctx := context.Background()

		mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{}}, nil)
		gomock.InOrder(
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", enProse)),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", enProse, "no")),
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", koProse)),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", koProse, "yes")),
		)

		res, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeCall, []tmtranscript.Transcript{}, "ko-KR")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != koProse {
			t.Errorf("expect: %q, got: %q", koProse, res)
		}
	})

	t.Run("cap reached returns last non-empty", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
		h := summaryHandler{reqHandler: mockReq, engineOpenaiHandler: mockOpenai, model: testModel, reasoningEffort: testReasoningEffort}
		ctx := context.Background()

		last := enProse + " third attempt"
		mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{}}, nil)
		gomock.InOrder(
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", enProse)),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", enProse, "no")),
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", enProse+" second")),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", enProse+" second", "no")),
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", last)),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", last, "no")),
		)

		res, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeCall, []tmtranscript.Transcript{}, "ko-KR")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != last {
			t.Errorf("expect: %q, got: %q", last, res)
		}
	})

	t.Run("indeterminate verification treated as pass", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
		h := summaryHandler{reqHandler: mockReq, engineOpenaiHandler: mockOpenai, model: testModel, reasoningEffort: testReasoningEffort}
		ctx := context.Background()

		mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{}}, nil)
		gomock.InOrder(
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", koProse)),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", koProse, "")),
		)

		res, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeCall, []tmtranscript.Transcript{}, "ko-KR")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != koProse {
			t.Errorf("expect: %q, got: %q", koProse, res)
		}
	})

	t.Run("empty regeneration preserves earlier non-empty", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
		h := summaryHandler{reqHandler: mockReq, engineOpenaiHandler: mockOpenai, model: testModel, reasoningEffort: testReasoningEffort}
		ctx := context.Background()

		mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{}}, nil)
		gomock.InOrder(
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", enProse)),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", enProse, "no")),
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", "")),
		)

		res, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeCall, []tmtranscript.Transcript{}, "ko-KR")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != enProse {
			t.Errorf("expect: %q, got: %q", enProse, res)
		}
	})

	t.Run("regeneration send hard failure falls back", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
		h := summaryHandler{reqHandler: mockReq, engineOpenaiHandler: mockOpenai, model: testModel, reasoningEffort: testReasoningEffort}
		ctx := context.Background()

		mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{}}, nil)
		gomock.InOrder(
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", enProse)),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", enProse, "no")),
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(nil, fmt.Errorf("send failed")),
		)

		res, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeCall, []tmtranscript.Transcript{}, "ko-KR")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != enProse {
			t.Errorf("expect: %q, got: %q", enProse, res)
		}
	})

	t.Run("minimal prose skips verification", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
		h := summaryHandler{reqHandler: mockReq, engineOpenaiHandler: mockOpenai, model: testModel, reasoningEffort: testReasoningEffort}
		ctx := context.Background()

		headersOnly := "Call Type:\n- None\n\nKey Discussion Points:\n- None\n\nAdditional Notes:\n- None"
		mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{}}, nil)
		// Only one Send, no SendOnce (verification skipped by min-prose guard).
		mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(genResp(headersOnly), nil)

		res, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeCall, []tmtranscript.Transcript{}, "ko-KR")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != headersOnly {
			t.Errorf("expect: %q, got: %q", headersOnly, res)
		}
	})

	t.Run("english target skips verification", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
		h := summaryHandler{reqHandler: mockReq, engineOpenaiHandler: mockOpenai, model: testModel, reasoningEffort: testReasoningEffort}
		ctx := context.Background()

		mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{}}, nil)
		// English target: single Send, no SendOnce.
		mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(genResp(enProse), nil)

		res, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeCall, []tmtranscript.Transcript{}, "en-US")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != enProse {
			t.Errorf("expect: %q, got: %q", enProse, res)
		}
	})

	t.Run("first pass pins output language in system message", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
		h := summaryHandler{reqHandler: mockReq, engineOpenaiHandler: mockOpenai, model: testModel, reasoningEffort: testReasoningEffort}
		ctx := context.Background()

		mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{}}, nil)
		gomock.InOrder(
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(genDo("ko-KR", koProse)),
			mockOpenai.EXPECT().SendOnce(gomock.Any(), gomock.Any()).DoAndReturn(verifyDo("ko-KR", koProse, "yes")),
		)

		res, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeCall, []tmtranscript.Transcript{}, "ko-KR")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res != koProse {
			t.Errorf("expect: %q, got: %q", koProse, res)
		}
	})
}

func Test_contentProcessReferenceTypeConference(t *testing.T) {

	tests := []struct {
		name string

		conferenceID uuid.UUID

		responseSummaries   []*summary.Summary
		responseConference  *cfconference.Conference
		responseTranscribes []tmtranscribe.Transcribe
		responseTranscripts []tmtranscript.Transcript
		responseVariable    *fmvariable.Variable
		responseSend        *openai.ChatCompletionResponse

		expectedFilterSummary     map[summary.Field]any
		expectedReferenceID       uuid.UUID
		expectedFilterTranscribe  map[tmtranscribe.Field]any
		expectedFilterTranscripts map[tmtranscript.Field]any
		expectedActiveflowID      uuid.UUID
		expectedSummaryContent    string
	}{
		{
			name: "normal",

			conferenceID: uuid.FromStringOrNil("12793fb8-0d78-11f0-b745-5bd13769c11a"),

			responseSummaries: []*summary.Summary{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("12c0991c-0d78-11f0-956f-c7fd6e2a65cd"),
					},
					ActiveflowID: uuid.FromStringOrNil("4eb1732e-0d78-11f0-adc7-070b3fa7186b"),
					ReferenceID:  uuid.FromStringOrNil("4ddead0e-0d78-11f0-896a-930b66cfb72b"),
				},
			},
			responseConference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4ddead0e-0d78-11f0-896a-930b66cfb72b"),
				},
				ConfbridgeID: uuid.FromStringOrNil("4e0cb7d0-0d78-11f0-bba8-27fe297783c9"),
			},
			responseTranscribes: []tmtranscribe.Transcribe{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4e316e40-0d78-11f0-b125-a79d64ccad15"),
					},
				},
			},
			responseTranscripts: []tmtranscript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4e5f7218-0d78-11f0-97fa-ffa151b9b13c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4e880ac0-0d78-11f0-bacc-73f6c9abeebe"),
					},
				},
			},
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					"key1": "value1",
				},
			},
			responseSend: &openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "response content",
						},
					},
				},
			},

			expectedFilterSummary: map[summary.Field]any{
				summary.FieldDeleted:     false,
				summary.FieldReferenceID: uuid.FromStringOrNil("12793fb8-0d78-11f0-b745-5bd13769c11a"),
			},
			expectedReferenceID: uuid.FromStringOrNil("12793fb8-0d78-11f0-b745-5bd13769c11a"),
			expectedFilterTranscribe: map[tmtranscribe.Field]any{
				tmtranscribe.FieldDeleted:     false,
				tmtranscribe.FieldCustomerID:  cmcustomer.IDAIManager.String(),
				tmtranscribe.FieldReferenceID: "4e0cb7d0-0d78-11f0-bba8-27fe297783c9",
			},
			expectedFilterTranscripts: map[tmtranscript.Field]any{
				tmtranscript.FieldDeleted:      false,
				tmtranscript.FieldTranscribeID: "4e316e40-0d78-11f0-b125-a79d64ccad15",
			},
			expectedActiveflowID:   uuid.FromStringOrNil("4eb1732e-0d78-11f0-adc7-070b3fa7186b"),
			expectedSummaryContent: "response content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,

				engineOpenaiHandler: mockOpenai,
				model:               testModel,
				reasoningEffort:     testReasoningEffort,
			}
			ctx := context.Background()

			mockDB.EXPECT().SummaryList(ctx, uint64(1), "", gomock.Any()).Return(tt.responseSummaries, nil)
			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.expectedReferenceID).Return(tt.responseConference, nil)

			// contentGetTranscripts
			mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(1), tt.expectedFilterTranscribe).Return(tt.responseTranscribes, nil)
			mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), tt.expectedFilterTranscripts).Return(tt.responseTranscripts, nil)

			// contentGet
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.expectedActiveflowID).Return(tt.responseVariable, nil)
			mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(tt.responseSend, nil)

			// UpdateStatusDone
			mockDB.EXPECT().SummaryUpdateStatusDoneIfNotDone(ctx, tt.responseSummaries[0].ID, gomock.Any()).Return(int64(1), nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseSummaries[0].ID).Return(tt.responseSummaries[0], nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseSummaries[0].CustomerID, summary.EventTypeUpdated, tt.responseSummaries[0])

			if err := h.contentProcessReferenceTypeConference(ctx, tt.conferenceID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_contentProcessReferenceTypeConference_alreadyDone is a VOIP-1422 regression
// test: when UpdateStatusDone returns ErrSummaryAlreadyDone (the DB-level conditional
// update affected zero rows -- this is the losing side of bin-conference-manager's
// double conference_deleted delivery, see ErrSummaryAlreadyDone's doc comment),
// contentProcessReferenceTypeConference must return cleanly (nil error) WITHOUT calling
// startOnEndFlow or its downstream (FlowV1ActiveflowExecute) or the summary_updated
// webhook. No mock expectations are set up for any of those -- gomock's strict
// controller fails the test on any such unexpected call, so their absence is what
// proves this test actually exercises the guard rather than passing vacuously.
func Test_contentProcessReferenceTypeConference_alreadyDone(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

	h := summaryHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,

		engineOpenaiHandler: mockOpenai,
	}
	ctx := context.Background()

	conferenceID := uuid.FromStringOrNil("a1b2c3d4-8b8c-11f0-9d2e-4b7c8f2a5d70")
	confbridgeID := uuid.FromStringOrNil("a1e01234-8b8c-11f0-9d2e-4b7c8f2a5d70")
	activeflowID := uuid.FromStringOrNil("a2103456-8b8c-11f0-9d2e-4b7c8f2a5d70")

	sm := &summary.Summary{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("a2321234-8b8c-11f0-9d2e-4b7c8f2a5d70"),
		},
		ActiveflowID: activeflowID,
		ReferenceID:  conferenceID,
	}
	cf := &cfconference.Conference{
		Identity: commonidentity.Identity{
			ID: conferenceID,
		},
		ConfbridgeID: confbridgeID,
	}
	transcribes := []tmtranscribe.Transcribe{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a2541234-8b8c-11f0-9d2e-4b7c8f2a5d70")}},
	}
	transcripts := []tmtranscript.Transcript{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a2761234-8b8c-11f0-9d2e-4b7c8f2a5d70")}},
	}
	variable := &fmvariable.Variable{Variables: map[string]string{"key1": "value1"}}
	openaiRes := &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: "response content"}}},
	}

	mockDB.EXPECT().SummaryList(ctx, uint64(1), "", gomock.Any()).Return([]*summary.Summary{sm}, nil)
	mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, conferenceID).Return(cf, nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(1), gomock.Any()).Return(transcribes, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), gomock.Any()).Return(transcripts, nil)
	mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(variable, nil)
	mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(openaiRes, nil)

	// the losing delivery: DB-level guard rejects it. No SummaryGet, no
	// PublishWebhookEvent, no startOnEndFlow-related calls may follow.
	mockDB.EXPECT().SummaryUpdateStatusDoneIfNotDone(ctx, sm.ID, gomock.Any()).Return(int64(0), nil)

	if err := h.contentProcessReferenceTypeConference(ctx, conferenceID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_contentProcessReferenceTypeCall_alreadyDone mirrors
// Test_contentProcessReferenceTypeConference_alreadyDone for the call reference type,
// confirming contentProcessReferenceTypeCall's independent stderrors.Is(err,
// ErrSummaryAlreadyDone) branch (content.go) also skips startOnEndFlow and the webhook
// cleanly rather than erroring or re-finalizing.
func Test_contentProcessReferenceTypeCall_alreadyDone(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

	h := summaryHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,

		engineOpenaiHandler: mockOpenai,
	}
	ctx := context.Background()

	callID := uuid.FromStringOrNil("a3981234-8b8c-11f0-9d2e-4b7c8f2a5d70")
	activeflowID := uuid.FromStringOrNil("a3ba1234-8b8c-11f0-9d2e-4b7c8f2a5d70")

	sm := &summary.Summary{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("a3dc1234-8b8c-11f0-9d2e-4b7c8f2a5d70"),
		},
		ActiveflowID: activeflowID,
		ReferenceID:  callID,
	}
	transcribes := []tmtranscribe.Transcribe{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a3fe1234-8b8c-11f0-9d2e-4b7c8f2a5d70")}},
	}
	transcripts := []tmtranscript.Transcript{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a4201234-8b8c-11f0-9d2e-4b7c8f2a5d70")}},
	}
	variable := &fmvariable.Variable{Variables: map[string]string{"key1": "value1"}}
	openaiRes := &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: "response content"}}},
	}

	mockDB.EXPECT().SummaryList(ctx, uint64(1), "", gomock.Any()).Return([]*summary.Summary{sm}, nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(1), gomock.Any()).Return(transcribes, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(1000), gomock.Any()).Return(transcripts, nil)
	mockReq.EXPECT().FlowV1VariableGet(ctx, activeflowID).Return(variable, nil)
	mockOpenai.EXPECT().Send(ctx, gomock.Any()).Return(openaiRes, nil)

	// the losing delivery: DB-level guard rejects it. No SummaryGet, no
	// PublishWebhookEvent, no startOnEndFlow-related calls may follow.
	mockDB.EXPECT().SummaryUpdateStatusDoneIfNotDone(ctx, sm.ID, gomock.Any()).Return(int64(0), nil)

	if err := h.contentProcessReferenceTypeCall(ctx, callID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
