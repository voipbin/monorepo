package databasehandler

import (
	"reflect"
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

func TestConvertToBool(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		want      bool
		wantError bool
	}{
		{
			name:      "bool true passthrough",
			input:     true,
			want:      true,
			wantError: false,
		},
		{
			name:      "bool false passthrough",
			input:     false,
			want:      false,
			wantError: false,
		},
		{
			name:      "string true",
			input:     "true",
			want:      true,
			wantError: false,
		},
		{
			name:      "string false",
			input:     "false",
			want:      false,
			wantError: false,
		},
		{
			name:      "string empty returns false",
			input:     "",
			want:      false,
			wantError: false,
		},
		{
			name:      "string other returns false",
			input:     "anything",
			want:      false,
			wantError: false,
		},
		{
			name:      "int unsupported",
			input:     1,
			want:      false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToBool(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("convertToBool() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if got != tt.want {
				t.Errorf("convertToBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToString(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		want      string
		wantError bool
	}{
		{
			name:      "string passthrough",
			input:     "hello",
			want:      "hello",
			wantError: false,
		},
		{
			name:      "empty string",
			input:     "",
			want:      "",
			wantError: false,
		},
		{
			name:      "unsupported type returns error",
			input:     123,
			want:      "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToString(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("convertToString() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if got != tt.want {
				t.Errorf("convertToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToInt(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		want      int
		wantError bool
	}{
		{
			name:      "int passthrough",
			input:     42,
			want:      42,
			wantError: false,
		},
		{
			name:      "float64 to int",
			input:     42.0,
			want:      42,
			wantError: false,
		},
		{
			name:      "float64 with decimal truncated",
			input:     42.7,
			want:      42,
			wantError: false,
		},
		{
			name:      "int64 to int",
			input:     int64(100),
			want:      100,
			wantError: false,
		},
		{
			name:      "negative int",
			input:     -10,
			want:      -10,
			wantError: false,
		},
		{
			name:      "unsupported type returns error",
			input:     "not-a-number",
			want:      0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToInt(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("convertToInt() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if got != tt.want {
				t.Errorf("convertToInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToFloat64(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		want      float64
		wantError bool
	}{
		{
			name:      "float64 passthrough",
			input:     3.14,
			want:      3.14,
			wantError: false,
		},
		{
			name:      "int to float64",
			input:     42,
			want:      42.0,
			wantError: false,
		},
		{
			name:      "int64 to float64",
			input:     int64(100),
			want:      100.0,
			wantError: false,
		},
		{
			name:      "unsupported type returns error",
			input:     "not-a-number",
			want:      0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToFloat64(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("convertToFloat64() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if got != tt.want {
				t.Errorf("convertToFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertValue(t *testing.T) {
	testUUID := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	tests := []struct {
		name       string
		value      any
		targetType interface{}
		wantError  bool
	}{
		{
			name:       "UUID string to UUID",
			value:      "550e8400-e29b-41d4-a716-446655440000",
			targetType: uuid.UUID{},
			wantError:  false,
		},
		{
			name:       "bool conversion",
			value:      "true",
			targetType: true,
			wantError:  false,
		},
		{
			name:       "string conversion",
			value:      "test",
			targetType: "",
			wantError:  false,
		},
		{
			name:       "int conversion",
			value:      42.0,
			targetType: 0,
			wantError:  false,
		},
		{
			name:       "float64 conversion",
			value:      42,
			targetType: 0.0,
			wantError:  false,
		},
		{
			name:       "same type passthrough",
			value:      testUUID,
			targetType: uuid.UUID{},
			wantError:  false,
		},
		{
			name:       "unsupported conversion",
			value:      123,
			targetType: struct{}{},
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetType := reflect.TypeOf(tt.targetType)
			_, err := convertValue(tt.value, targetType)

			if (err != nil) != tt.wantError {
				t.Errorf("convertValue() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
