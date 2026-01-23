package ai

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
			name:     "field_name",
			constant: FieldName,
			expected: "name",
		},
		{
			name:     "field_detail",
			constant: FieldDetail,
			expected: "detail",
		},
		{
			name:     "field_engine_type",
			constant: FieldEngineType,
			expected: "engine_type",
		},
		{
			name:     "field_engine_model",
			constant: FieldEngineModel,
			expected: "engine_model",
		},
		{
			name:     "field_engine_data",
			constant: FieldEngineData,
			expected: "engine_data",
		},
		{
			name:     "field_engine_key",
			constant: FieldEngineKey,
			expected: "engine_key",
		},
		{
			name:     "field_init_prompt",
			constant: FieldInitPrompt,
			expected: "init_prompt",
		},
		{
			name:     "field_tts_type",
			constant: FieldTTSType,
			expected: "tts_type",
		},
		{
			name:     "field_tts_voice_id",
			constant: FieldTTSVoiceID,
			expected: "tts_voice_id",
		},
		{
			name:     "field_stt_type",
			constant: FieldSTTType,
			expected: "stt_type",
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
