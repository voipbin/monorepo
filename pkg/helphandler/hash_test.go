package helphandler

import "testing"

func Test_HashGenerate(t *testing.T) {
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

			h := helpHandler{}

			res, err := h.HashGenerate(tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !h.HashCheck(tt.password, res) {
				t.Errorf("Wrong match. expect: true, got: false")
			}
		})
	}
}
