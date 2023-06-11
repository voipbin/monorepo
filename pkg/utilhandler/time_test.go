package utilhandler

import (
	"testing"
)

func Test_TimeConvert(t *testing.T) {

	type test struct {
		name string

		timeString string

		expectRes string
	}

	tests := []test{
		{
			name: "normal",

			timeString: "2023-06-08 03:22:17.995001",
			expectRes:  "2023-06-08 03:22:17.995001 +0000 UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := TimeParse(tt.timeString)

			if tt.expectRes != res.String() {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
