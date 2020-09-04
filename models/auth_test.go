package models

import "testing"

func TestAuthGenerateHash(t *testing.T) {

	type test struct {
		name string
		auth Auth
	}

	tests := []test{
		{
			"normal",
			Auth{
				Username: "test",
				Password: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := GenerateHash(tt.auth.Password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if tt.auth.checkHash(res) != true {
				t.Error("Wrong match. expect: true, got: false")
			}
		})
	}
}
