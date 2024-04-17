package agenthandler

import "testing"

func Test_generateHash(t *testing.T) {

	tests := []struct {
		name string

		password string
	}{
		{
			name: "normal",

			password: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := generateHash(tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !checkHash(tt.password, res) {
				t.Errorf("Wrong match. expect: ok, got: false")
			}
		})
	}
}
