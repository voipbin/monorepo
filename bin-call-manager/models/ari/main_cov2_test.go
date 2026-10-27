package ari

import (
	"reflect"
	"testing"
)

func Test_ArgsMap_UnmarshalJSON_cov2(t *testing.T) {
	tests := []struct {
		name string

		data string

		expectErr bool
		expect    ArgsMap
	}{
		{name: "pairs and bare key", data: `["a=1","b"]`, expect: ArgsMap{"a": "1", "b": ""}},
		{name: "not an array", data: `{"a":1}`, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res ArgsMap
			err := res.UnmarshalJSON([]byte(tt.data))
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && !reflect.DeepEqual(res, tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}

func Test_Timestamp_UnmarshalJSON_cov2(t *testing.T) {
	tests := []struct {
		name string

		data string

		expectErr bool
		expect    Timestamp
	}{
		{name: "strips the zone suffix", data: `"2020-02-10T13:08:18.888+0000"`, expect: Timestamp("2020-02-10T13:08:18.888")},
		{name: "not a string", data: `123`, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res Timestamp
			err := res.UnmarshalJSON([]byte(tt.data))
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && res != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}

func Test_Parse_cov2_invalidJSON(t *testing.T) {
	bad := []byte(`{"id": 123`)

	if res, err := ParseBridge(bad); err == nil || res != nil {
		t.Errorf("ParseBridge: expect error, got: %v, %v", res, err)
	}
	if res, err := ParseChannel(bad); err == nil || res != nil {
		t.Errorf("ParseChannel: expect error, got: %v, %v", res, err)
	}
	if res, err := ParsePlayback(bad); err == nil || res != nil {
		t.Errorf("ParsePlayback: expect error, got: %v, %v", res, err)
	}
}
