package databasehandler

import (
	"testing"

	uuid "github.com/gofrs/uuid"
)

func TestConvertToUUID(t *testing.T) {
	validUUID := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	tests := []struct {
		name      string
		input     any
		want      uuid.UUID
		wantError bool
	}{
		{
			name:      "valid UUID string",
			input:     "550e8400-e29b-41d4-a716-446655440000",
			want:      validUUID,
			wantError: false,
		},
		{
			name:      "empty string returns Nil UUID without error",
			input:     "",
			want:      uuid.Nil,
			wantError: false,
		},
		{
			name:      "uuid.UUID passthrough",
			input:     validUUID,
			want:      validUUID,
			wantError: false,
		},
		{
			name:      "invalid UUID string returns error",
			input:     "not-a-valid-uuid",
			want:      uuid.Nil,
			wantError: true,
		},
		{
			name:      "malformed UUID string returns error",
			input:     "550e8400-e29b-41d4-a716",
			want:      uuid.Nil,
			wantError: true,
		},
		{
			name:      "unsupported type returns error",
			input:     12345,
			want:      uuid.Nil,
			wantError: true,
		},
		{
			name:      "nil input returns error",
			input:     nil,
			want:      uuid.Nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToUUID(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("convertToUUID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if got != tt.want {
				t.Errorf("convertToUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}
