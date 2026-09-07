package aicallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_getDataAsJSON(t *testing.T) {
	tests := []struct {
		name string

		data         map[string]any
		activeflowID uuid.UUID

		responseSubstitutes map[string]string
		expectedRes         string
	}{
		{
			name: "nested structure with variable substitution",
			data: map[string]any{
				"key1": "value1",
				"key2": 2,
				"key3": true,
				"key4": "The culprit is {${lame_person}}.",
				"key5": map[string]any{
					"subkey1": "subvalue1",
					"subkey2": 3,
					"subkey3": "The ghost is {${ghost_person}}.",
					"subkey4": []string{
						"sub list val 1",
						"The secret is {${secret_info}}.",
					},
				},
				"key6": []string{
					"list val 1",
					"The answer is {${answer_info}}.",
				},
				"key7": 4.5,
				"key8": nil,
			},
			activeflowID: uuid.FromStringOrNil("d48b2510-c035-11f0-b454-83d837506895"),
			responseSubstitutes: map[string]string{
				"value1":                           "response 1",
				"The culprit is {${lame_person}}.": "response 2",
				"subvalue1":                        "response 3",
				"The ghost is {${ghost_person}}.":  "response 4",
				"sub list val 1":                   "response 5",
				"The secret is {${secret_info}}.":  "response 6",
				"list val 1":                       "response 7",
				"The answer is {${answer_info}}.":  "response 8",
			},
			expectedRes: `
{
    "key1": "response 1",
    "key2": 2,
    "key3": true,
    "key4": "response 2",
    "key5": {
        "subkey1": "response 3",
        "subkey2": 3,
        "subkey3": "response 4",
        "subkey4": [
            "response 5",
            "response 6"
        ]
    },
    "key6": [
        "response 7",
        "response 8"
    ],
    "key7": 4.5,
    "key8": null
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			for k, v := range tt.responseSubstitutes {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, k).Return(v, nil)
			}

			res := h.getDataAsJSON(ctx, tt.data, tt.activeflowID)

			var expected, actual map[string]any
			if err := json.Unmarshal([]byte(tt.expectedRes), &expected); err != nil {
				t.Fatalf("invalid expectedRes JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(res), &actual); err != nil {
				t.Fatalf("invalid result JSON: %v", err)
			}

			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", expected, actual)
			}
		})
	}
}

func Test_getParameterValue(t *testing.T) {
	tests := []struct {
		name string

		input          any
		activeflowID   uuid.UUID
		substitutes    map[string]string
		expectedResult any
	}{
		{
			name:         "simple string substitution",
			input:        "value1",
			activeflowID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			substitutes: map[string]string{
				"value1": "response 1",
			},
			expectedResult: "response 1",
		},
		{
			name: "nested map with substitutions",
			input: map[string]any{
				"key1": "value1",
				"key2": 2,
				"key3": "value2",
			},
			activeflowID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			substitutes: map[string]string{
				"value1": "response 1",
				"value2": "response 2",
			},
			expectedResult: map[string]any{
				"key1": "response 1",
				"key2": int64(2),
				"key3": "response 2",
			},
		},
		{
			name: "slice with substitutions",
			input: []any{
				"value1",
				"value2",
				3,
			},
			activeflowID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
			substitutes: map[string]string{
				"value1": "response 1",
				"value2": "response 2",
			},
			expectedResult: []any{
				"response 1",
				"response 2",
				int64(3),
			},
		},
		{
			name:           "nil value",
			input:          nil,
			activeflowID:   uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			substitutes:    map[string]string{},
			expectedResult: nil,
		},
		{
			name: "mixed nested structure",
			input: map[string]any{
				"key1": "value1",
				"key2": []any{
					"value2",
					map[string]any{
						"sub1": "value3",
						"sub2": 10,
					},
				},
			},
			activeflowID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			substitutes: map[string]string{
				"value1": "response 1",
				"value2": "response 2",
				"value3": "response 3",
			},
			expectedResult: map[string]any{
				"key1": "response 1",
				"key2": []any{
					"response 2",
					map[string]any{
						"sub1": "response 3",
						"sub2": int64(10),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			// string substitution mocking
			for k, v := range tt.substitutes {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, k).Return(v, nil)
			}

			actual := h.getParameterValue(ctx, tt.input, tt.activeflowID)

			if !reflect.DeepEqual(tt.expectedResult, actual) {
				t.Errorf("Wrong match.\nexpected: %#v\ngot: %#v", tt.expectedResult, actual)
			}
		})
	}
}

// Test_substituteText pins the extracted substitution core (VOIP-1484): it is a
// PLAIN wrapper over the RPC with no gating of its own, and it returns the
// error instead of swallowing it. Its two existing callers keep their lenient
// per-call fallbacks; the strict refresh path needs the error.
func Test_substituteText(t *testing.T) {
	tests := []struct {
		name string

		activeflowID uuid.UUID
		text         string

		responseText string
		responseErr  error

		expectRes string
		expectErr bool
	}{
		{
			name: "substituted",

			activeflowID: uuid.FromStringOrNil("11110000-0001-11f0-6666-000000000001"),
			text:         "Case ${voipbin.case.id}.",

			responseText: "Case c-1.",

			expectRes: "Case c-1.",
		},
		{
			name: "the rpc error is propagated, not swallowed",

			activeflowID: uuid.FromStringOrNil("11110000-0002-11f0-6666-000000000001"),
			text:         "Case ${voipbin.case.id}.",

			responseErr: fmt.Errorf("activeflow ended"),

			expectErr: true,
		},
		{
			// No "${" gate of its own: the caller decides. getParameterValue
			// relies on exactly this, since it substitutes EVERY string leaf.
			name: "a plain string still reaches the rpc",

			activeflowID: uuid.FromStringOrNil("11110000-0003-11f0-6666-000000000001"),
			text:         "plain",

			responseText: "plain",

			expectRes: "plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, tt.text).Return(tt.responseText, tt.responseErr)

			res, err := h.substituteText(ctx, tt.activeflowID, tt.text)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("wrong match. expect: %q, got: %q", tt.expectRes, res)
			}
		})
	}
}

// Test_getInitPrompt_SubstitutionErrorKeepsTheRawPrompt pins the LENIENT
// wrapper's behaviour across the extraction: a failed substitution must still
// produce the raw prompt for a live turn. Only the refresh path fails closed.
func Test_getInitPrompt_SubstitutionErrorKeepsTheRawPrompt(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	activeflowID := uuid.FromStringOrNil("11110000-0004-11f0-6666-000000000001")
	a := &ai.AI{InitPrompt: "Case ${voipbin.case.id}."}

	mockReq.EXPECT().FlowV1VariableSubstitute(ctx, activeflowID, a.InitPrompt).Return("", fmt.Errorf("activeflow ended"))

	if res := h.getInitPrompt(ctx, a, activeflowID); res != "Case ${voipbin.case.id}." {
		t.Errorf("the raw init prompt must survive a substitution failure. got: %q", res)
	}
}

// Test_getParameterValue_PerLeafFallback pins that the lenient walk is still
// PER LEAF after the extraction: one failing leaf must not discard the leaves
// that substituted successfully. That is exactly what separates it from
// substituteValue, which discards everything on the first error.
func Test_getParameterValue_PerLeafFallback(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	activeflowID := uuid.FromStringOrNil("11110000-0005-11f0-6666-000000000001")

	mockReq.EXPECT().FlowV1VariableSubstitute(ctx, activeflowID, "${bad}").Return("", fmt.Errorf("nope"))
	mockReq.EXPECT().FlowV1VariableSubstitute(ctx, activeflowID, "${good}").Return("resolved", nil)

	res := h.getParameterValue(ctx, map[string]any{"a": "${bad}", "b": "${good}"}, activeflowID)

	expect := map[string]any{"a": "${bad}", "b": "resolved"}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

// Test_substituteValue pins the STRICT walker used only by the session refresh:
// same recursive shape as getParameterValue, but the first error aborts and the
// partial result is discarded.
func Test_substituteValue(t *testing.T) {
	activeflowID := uuid.FromStringOrNil("11110000-0006-11f0-6666-000000000001")

	tests := []struct {
		name string

		input any

		substitutes map[string]string
		failOn      string

		expectRes any
		expectErr bool
	}{
		{
			name: "nested map and slice leaves are all substituted",

			input: map[string]any{
				"outer": map[string]any{"inner": "${a}"},
				"list":  []any{"${b}", 3, true},
			},

			substitutes: map[string]string{"${a}": "A", "${b}": "B"},

			expectRes: map[string]any{
				"outer": map[string]any{"inner": "A"},
				"list":  []any{"B", int64(3), true},
			},
		},
		{
			name: "nil is passed through without an rpc",

			input: nil,

			expectRes: nil,
		},
		{
			name: "the first leaf error discards the whole walk",

			input: map[string]any{"a": "${a}"},

			failOn: "${a}",

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}
			ctx := context.Background()

			for in, out := range tt.substitutes {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, activeflowID, in).Return(out, nil)
			}
			if tt.failOn != "" {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, activeflowID, tt.failOn).Return("", fmt.Errorf("nope"))
			}

			res, err := h.substituteValue(ctx, activeflowID, tt.input)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				if res != nil {
					t.Errorf("a failed strict walk must discard its partial result. got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("wrong match.\nexpect: %#v\ngot: %#v", tt.expectRes, res)
			}
		})
	}
}

// Test_refreshPrompt pins the Insight session refresh's own prompt rules
// (VOIP-1484):
//
//   - a prompt with no "${" never reaches the RPC, so a reused AIcall whose
//     original activeflow is long dead still gets a refreshed session;
//   - a prompt WITH "${" and no activeflow is an error, never a raw ${...}
//     persisted into a system row;
//   - an empty (or "{}") parameter block produces no row at all.
func Test_refreshPrompt(t *testing.T) {
	activeflowID := uuid.FromStringOrNil("11110000-0007-11f0-6666-000000000001")
	aicallID := uuid.FromStringOrNil("11110000-0008-11f0-6666-000000000001")

	tests := []struct {
		name string

		aicall *aicall.AIcall
		ai     *ai.AI

		substitutes map[string]string
		failOn      string

		expectInitPrompt string
		expectParamJSON  string
		expectErr        bool
	}{
		{
			name: "no variables anywhere means no rpc at all",

			aicall: &aicall.AIcall{
				Identity:     commonidentity.Identity{ID: aicallID},
				ActiveflowID: activeflowID,
				Parameter:    map[string]any{"case_id": "c-1"},
			},
			ai: &ai.AI{InitPrompt: "You are the case assistant."},

			expectInitPrompt: "You are the case assistant.",
			expectParamJSON:  `{"case_id":"c-1"}`,
		},
		{
			name: "a variable-free prompt refreshes even with no activeflow",

			aicall: &aicall.AIcall{
				Identity:     commonidentity.Identity{ID: aicallID},
				ActiveflowID: uuid.Nil,
			},
			ai: &ai.AI{InitPrompt: "You are the case assistant."},

			expectInitPrompt: "You are the case assistant.",
		},
		{
			name: "an init prompt variable is substituted",

			aicall: &aicall.AIcall{
				Identity:     commonidentity.Identity{ID: aicallID},
				ActiveflowID: activeflowID,
			},
			ai: &ai.AI{InitPrompt: "Case ${voipbin.case.id}."},

			substitutes: map[string]string{"Case ${voipbin.case.id}.": "Case c-1."},

			expectInitPrompt: "Case c-1.",
		},
		{
			name: "an init prompt variable with no activeflow fails closed",

			aicall: &aicall.AIcall{
				Identity:     commonidentity.Identity{ID: aicallID},
				ActiveflowID: uuid.Nil,
			},
			ai: &ai.AI{InitPrompt: "Case ${voipbin.case.id}."},

			expectErr: true,
		},
		{
			name: "an init prompt substitution failure fails closed",

			aicall: &aicall.AIcall{
				Identity:     commonidentity.Identity{ID: aicallID},
				ActiveflowID: activeflowID,
			},
			ai: &ai.AI{InitPrompt: "Case ${voipbin.case.id}."},

			failOn: "Case ${voipbin.case.id}.",

			expectErr: true,
		},
		{
			name: "nested parameter leaves are substituted",

			aicall: &aicall.AIcall{
				Identity:     commonidentity.Identity{ID: aicallID},
				ActiveflowID: activeflowID,
				Parameter: map[string]any{
					"nested": map[string]any{"case_id": "${voipbin.case.id}"},
					"list":   []any{"static"},
				},
			},
			ai: &ai.AI{InitPrompt: "You are the case assistant."},

			substitutes: map[string]string{
				"${voipbin.case.id}": "c-1",
				"static":             "static",
			},

			expectInitPrompt: "You are the case assistant.",
			expectParamJSON:  `{"list":["static"],"nested":{"case_id":"c-1"}}`,
		},
		{
			name: "a parameter variable with no activeflow fails closed",

			aicall: &aicall.AIcall{
				Identity:     commonidentity.Identity{ID: aicallID},
				ActiveflowID: uuid.Nil,
				Parameter:    map[string]any{"case_id": "${voipbin.case.id}"},
			},
			ai: &ai.AI{InitPrompt: "You are the case assistant."},

			expectErr: true,
		},
		{
			name: "an empty parameter map produces no parameter row",

			aicall: &aicall.AIcall{
				Identity:     commonidentity.Identity{ID: aicallID},
				ActiveflowID: activeflowID,
				Parameter:    map[string]any{},
			},
			ai: &ai.AI{InitPrompt: "You are the case assistant."},

			expectInitPrompt: "You are the case assistant.",
		},
		{
			name: "an empty init prompt stays empty and writes no row",

			aicall: &aicall.AIcall{
				Identity:     commonidentity.Identity{ID: aicallID},
				ActiveflowID: activeflowID,
			},
			ai: &ai.AI{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}
			ctx := context.Background()

			for in, out := range tt.substitutes {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, activeflowID, in).Return(out, nil)
			}
			if tt.failOn != "" {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, activeflowID, tt.failOn).Return("", fmt.Errorf("nope"))
			}

			initPrompt, paramJSON, err := h.refreshPrompt(ctx, tt.aicall, tt.ai)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if initPrompt != tt.expectInitPrompt {
				t.Errorf("wrong init prompt. expect: %q, got: %q", tt.expectInitPrompt, initPrompt)
			}
			if paramJSON != tt.expectParamJSON {
				t.Errorf("wrong parameter json. expect: %q, got: %q", tt.expectParamJSON, paramJSON)
			}
		})
	}
}
