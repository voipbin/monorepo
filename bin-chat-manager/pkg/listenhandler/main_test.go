package listenhandler

import (
	"net/url"
	"reflect"
	"testing"
)

func Test_getFilters(t *testing.T) {
	tests := []struct {
		name string

		url       string
		expectRes map[string]string
	}{
		{
			"normal",

			"/v1/chatrooms?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_owner_id=5dae1058-3503-11ed-a7d3-df338985d478&filter_deleted=false",
			map[string]string{
				"owner_id": "5dae1058-3503-11ed-a7d3-df338985d478",
				"deleted":  "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			u, err := url.Parse(tt.url)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res := getFilters(u)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
