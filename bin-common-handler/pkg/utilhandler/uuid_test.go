package utilhandler

import "testing"

func Test_UUIDCreate(t *testing.T) {

	type test struct {
		name string
	}

	tests := []test{
		{
			name: "normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := UUIDCreate()

			if res.String() == "" {
				t.Errorf("Wrong match. expected: not empty, got: empty")
			}
		})
	}
}
