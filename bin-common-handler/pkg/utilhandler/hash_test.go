package utilhandler

import "testing"

func Test_HashGeneratePassword(t *testing.T) {

	type test struct {
		name string

		password string
		cost     int
	}
	tests := []test{
		{
			name:     "normal",
			password: "password",
			cost:     10,
		},
		{
			name:     "empty password",
			password: "",
			cost:     10,
		},
		{
			name:     "low cost",
			password: "password",
			cost:     4,
		},
		{
			name:     "high cost",
			password: "password",
			cost:     14,
		},
		{
			name:     "special characters",
			password: "p@$$w0rd!",
			cost:     10,
		},
		{
			name:     "unicode characters",
			password: "pässwörd",
			cost:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := HashGenerate(tt.password, tt.cost)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			if !HashCheckPassword(tt.password, res) {
				t.Errorf("Wrong match. expected: ok, got: false")
			}
		})
	}
}
