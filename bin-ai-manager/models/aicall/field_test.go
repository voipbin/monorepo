package aicall

import (
	"testing"
)

func TestFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Field
		expected string
	}{
		{
			name:     "field_id",
			constant: FieldID,
			expected: "id",
		},
		{
			name:     "field_customer_id",
			constant: FieldCustomerID,
			expected: "customer_id",
		},
		{
			name:     "field_ai_id",
			constant: FieldAIID,
			expected: "ai_id",
		},
		{
			name:     "field_ai_engine_type",
			constant: FieldAIEngineType,
			expected: "ai_engine_type",
		},
		{
			name:     "field_ai_engine_model",
			constant: FieldAIEngineModel,
			expected: "ai_engine_model",
		},
		{
			name:     "field_ai_engine_data",
			constant: FieldAIEngineData,
			expected: "ai_engine_data",
		},
		{
			name:     "field_ai_tts_type",
			constant: FieldAITTSType,
			expected: "ai_tts_type",
		},
		{
			name:     "field_ai_tts_voice_id",
			constant: FieldAITTSVoiceID,
			expected: "ai_tts_voice_id",
		},
		{
			name:     "field_ai_stt_type",
			constant: FieldAISTTType,
			expected: "ai_stt_type",
		},
		{
			name:     "field_activeflow_id",
			constant: FieldActiveflowID,
			expected: "activeflow_id",
		},
		{
			name:     "field_reference_type",
			constant: FieldReferenceType,
			expected: "reference_type",
		},
		{
			name:     "field_reference_id",
			constant: FieldReferenceID,
			expected: "reference_id",
		},
		{
			name:     "field_confbridge_id",
			constant: FieldConfbridgeID,
			expected: "confbridge_id",
		},
		{
			name:     "field_pipecatcall_id",
			constant: FieldPipecatcallID,
			expected: "pipecatcall_id",
		},
		{
			name:     "field_status",
			constant: FieldStatus,
			expected: "status",
		},
		{
			name:     "field_gender",
			constant: FieldGender,
			expected: "gender",
		},
		{
			name:     "field_language",
			constant: FieldLanguage,
			expected: "language",
		},
		{
			name:     "field_tm_end",
			constant: FieldTMEnd,
			expected: "tm_end",
		},
		{
			name:     "field_tm_create",
			constant: FieldTMCreate,
			expected: "tm_create",
		},
		{
			name:     "field_tm_update",
			constant: FieldTMUpdate,
			expected: "tm_update",
		},
		{
			name:     "field_tm_delete",
			constant: FieldTMDelete,
			expected: "tm_delete",
		},
		{
			name:     "field_deleted",
			constant: FieldDeleted,
			expected: "deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
