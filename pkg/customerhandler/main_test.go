package customerhandler

import "testing"

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name string

		password string
	}{
		{
			"normal",

			"admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := generateHash(tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !checkHash(tt.password, res) {
				t.Errorf("Wrong match. expect: true, got: false")
			}
		})
	}
}
