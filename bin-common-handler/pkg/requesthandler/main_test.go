package requesthandler

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	initPrometheus("test")

	os.Exit(m.Run())
}

func Test_parseFilters(t *testing.T) {

	tests := []struct {
		name string

		uri     string
		filters map[string]string

		expectRes string
	}{
		{
			"normal",

			"/v1/calls/5bfcdcd6-4c6e-11ec-bed9-8fe4c0fdf5ba",
			map[string]string{
				"customer_id": "c222417c-f0b8-11ee-ba42-d3d4b7a5fd72",
				"deleted":     "false",
			},

			"/v1/calls/5bfcdcd6-4c6e-11ec-bed9-8fe4c0fdf5ba&filter_customer_id=c222417c-f0b8-11ee-ba42-d3d4b7a5fd72&filter_deleted=false",
		},
		{
			"empty uri",

			"",
			map[string]string{
				"customer_id": "0438f358-f0b9-11ee-b276-873b0a3bf9bd",
				"deleted":     "false",
			},

			"&filter_customer_id=0438f358-f0b9-11ee-b276-873b0a3bf9bd&filter_deleted=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := parseFilters(tt.uri, tt.filters)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}
