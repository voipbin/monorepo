package speaking

import (
	"testing"
)

func Test_Status(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{
			name:   "StatusInitiating",
			status: StatusInitiating,
		},
		{
			name:   "StatusActive",
			status: StatusActive,
		},
		{
			name:   "StatusStopped",
			status: StatusStopped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status == "" {
				t.Errorf("expected non-empty status")
			}
		})
	}
}

func Test_Field(t *testing.T) {
	tests := []struct {
		name  string
		field Field
	}{
		{name: "FieldID", field: FieldID},
		{name: "FieldCustomerID", field: FieldCustomerID},
		{name: "FieldReferenceType", field: FieldReferenceType},
		{name: "FieldReferenceID", field: FieldReferenceID},
		{name: "FieldLanguage", field: FieldLanguage},
		{name: "FieldProvider", field: FieldProvider},
		{name: "FieldVoiceID", field: FieldVoiceID},
		{name: "FieldDirection", field: FieldDirection},
		{name: "FieldStatus", field: FieldStatus},
		{name: "FieldPodID", field: FieldPodID},
		{name: "FieldTMCreate", field: FieldTMCreate},
		{name: "FieldTMUpdate", field: FieldTMUpdate},
		{name: "FieldTMDelete", field: FieldTMDelete},
		{name: "FieldDeleted", field: FieldDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field == "" {
				t.Errorf("expected non-empty field")
			}
		})
	}
}
