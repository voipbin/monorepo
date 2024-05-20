package utilhandler

import (
	"net/url"
	reflect "reflect"
	"testing"
)

func Test_URLParseFilters(t *testing.T) {

	tests := []struct {
		name string
		url  string

		expectRes map[string]string
	}{
		{
			"normal",
			"/v1/agents?page_size=10&page_token=2021-11-23%2017:55:39.712000&filter_customer_id=5fd7f9b8-cb37-11ee-bd29-f30560a6ac86&filter_tag_ids=f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a,08789a66-b236-11ee-8a51-b31bbd98fe91&filter_deleted=false&filter_status=available&filter_test1=&filter_test2=test2_value&filter_number=%2B14703298699",

			map[string]string{
				"customer_id": "5fd7f9b8-cb37-11ee-bd29-f30560a6ac86",
				"deleted":     "false",
				"number":      "+14703298699",
				"status":      "available",
				"tag_ids":     "f768910c-4d8f-11ec-b5ec-ab5be5e8ef8a,08789a66-b236-11ee-8a51-b31bbd98fe91",
				"test1":       "",
				"test2":       "test2_value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			u, err := url.Parse(tt.url)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res := URLParseFilters(u)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_URLMergeFilters(t *testing.T) {

	tests := []struct {
		name string

		uri     string
		filters map[string]string

		expectRes string
	}{
		{
			"normal",

			"/v1/calls/5bfcdcd6-4c6e-11ec-bed9-8fe4c0fdf5ba?token=some_token",
			map[string]string{
				"customer_id": "c222417c-f0b8-11ee-ba42-d3d4b7a5fd72",
				// "deleted":     "false",
				// "number":      "+123456789",
			},

			// "/v1/calls/5bfcdcd6-4c6e-11ec-bed9-8fe4c0fdf5ba?token=some_token&filter_customer_id=c222417c-f0b8-11ee-ba42-d3d4b7a5fd72&filter_deleted=false&filter_number=%2B123456789",
			"/v1/calls/5bfcdcd6-4c6e-11ec-bed9-8fe4c0fdf5ba?token=some_token&filter_customer_id=c222417c-f0b8-11ee-ba42-d3d4b7a5fd72",
		},
		{
			"empty uri",

			"",
			map[string]string{
				"customer_id": "0438f358-f0b9-11ee-b276-873b0a3bf9bd",
				// "deleted":     "false",
				// "number":      "+123456789",
			},

			// "&filter_customer_id=0438f358-f0b9-11ee-b276-873b0a3bf9bd&filter_deleted=false&filter_number=%2B123456789",
			"&filter_customer_id=0438f358-f0b9-11ee-b276-873b0a3bf9bd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := URLMergeFilters(tt.uri, tt.filters)
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}
