package middleware

import (
	"reflect"
	"testing"
)

func Test_GenerateTokenWithData(t *testing.T) {

	tests := []struct {
		name string

		data map[string]interface{}
	}{
		{
			name: "normal",

			data: map[string]interface{}{
				"key1": "val1",
				"key2": "val2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := GenerateTokenWithData(tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tmp, err := ValidateToken(res)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			delete(tmp, "expire")
			if !reflect.DeepEqual(tmp, tt.data) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.data, tmp)
			}
		})
	}
}
