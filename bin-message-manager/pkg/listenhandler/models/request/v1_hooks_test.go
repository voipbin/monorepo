package request

import (
	"encoding/json"
	"reflect"
	"testing"

	hmhook "monorepo/bin-hook-manager/models/hook"
)

func Test_V1DataHooksPostUnmarshal(t *testing.T) {
	tests := []struct {
		name string

		data      []byte
		expectRes *V1DataHooksPost
	}{
		{
			"normal",

			[]byte(`{"received_uri":"hook.voipbin.net/v1.0/hooks/telnyx","received_data":"eyJrZXkxIjoidmFsMSJ9"}`),
			&V1DataHooksPost{
				Hook: hmhook.Hook{
					ReceviedURI:  "hook.voipbin.net/v1.0/hooks/telnyx",
					ReceivedData: []byte(`{"key1":"val1"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := V1DataHooksPost{}
			if err := json.Unmarshal(tt.data, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, res)
			}
		})
	}
}
