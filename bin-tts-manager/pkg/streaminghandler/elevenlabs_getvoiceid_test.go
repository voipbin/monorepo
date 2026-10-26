package streaminghandler

import (
	"context"
	"errors"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// Test_getVoiceID_explicitVoiceIDTakesPriority verifies the first branch of
// getVoiceID: an explicit st.VoiceID short-circuits the variable and
// language/gender lookups.
func Test_getVoiceID_explicitVoiceIDTakesPriority(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// No EXPECT() on FlowV1VariableGet: a call here would mean the explicit
	// VoiceID branch did not short-circuit.
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &elevenlabsHandler{reqHandler: mockReq}

	st := &streaming.Streaming{
		VoiceID:      "explicit-voice-id",
		ActiveflowID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
		Language:     "english",
	}

	result := h.getVoiceID(context.Background(), st)
	if result != "explicit-voice-id" {
		t.Errorf("Wrong match. expected: explicit-voice-id, got: %s", result)
	}
}

// Test_getVoiceIDByVariable_errorAndMissingCases covers getVoiceIDByVariable's
// remaining branches: FlowV1VariableGet error, missing key, and a value of
// the wrong length.
func Test_getVoiceIDByVariable_errorAndMissingCases(t *testing.T) {
	activeflowID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002")

	tests := []struct {
		name string

		responseVariable *fmvariable.Variable
		responseErr      error
	}{
		{
			name:        "FlowV1VariableGet returns an error",
			responseErr: errors.New("boom"),
		},
		{
			name: "the voice id variable key is missing",
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{"some_other_key": "value"},
			},
		},
		{
			name: "the voice id variable has the wrong length",
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{variableElevenlabsVoiceID: "too-short"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockReq.EXPECT().FlowV1VariableGet(gomock.Any(), activeflowID).Return(tt.responseVariable, tt.responseErr)

			h := &elevenlabsHandler{reqHandler: mockReq}

			result := h.getVoiceIDByVariable(context.Background(), activeflowID)
			if result != "" {
				t.Errorf("Wrong match. expected: empty string, got: %s", result)
			}
		})
	}
}

// Test_getVoiceIDByLangGender_fallsBackToNeutralWhenGenderKeyMissing proves
// the second map lookup (the neutral fallback) is reached and succeeds when
// the exact "<language>_<gender>" key does not exist but a
// "<language>_neutral" key does. Every language currently in the map has an
// explicit entry for every known streaming.Gender value, so an unknown
// gender string is used to force the first lookup to miss.
func Test_getVoiceIDByLangGender_fallsBackToNeutralWhenGenderKeyMissing(t *testing.T) {
	h := &elevenlabsHandler{}

	result := h.getVoiceIDByLangGender(context.Background(), "french", streaming.Gender("unknown-gender"))
	if result != "SmWACbi37pETyxxMhSpc" {
		t.Errorf("Wrong match. expected the french neutral voice id, got: %s", result)
	}
}
