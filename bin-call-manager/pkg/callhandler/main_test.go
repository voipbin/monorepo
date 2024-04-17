package callhandler

import "testing"

func TestGetContextType(t *testing.T) {

	type test struct {
		name              string
		context           interface{}
		expectContextType contextType
	}

	tests := []test{
		{
			"nil",
			nil,
			contextTypeCall,
		},
		{
			"empty",
			"",
			contextTypeCall,
		},
		{
			"conference echo",
			"conf-echo",
			contextTypeConference,
		},
		{
			"call-incoming",
			"call-in",
			contextTypeCall,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getContextType(tt.context)
			if res != tt.expectContextType {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectContextType, res)
			}
		})
	}
}
