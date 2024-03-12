package outline

import "testing"

func Test_GetMetricNameSpace(t *testing.T) {

	tests := []struct {
		name       string
		servieName ServiceName

		expectRes string
	}{
		{
			"call-manager",

			ServiceNameCallManager,
			"call_manager",
		},
		{
			"api-manager",

			ServiceNameAPIManager,
			"api_manager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := GetMetricNameSpace(tt.servieName)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", res, tt.expectRes)
			}
		})
	}
}
