package action

import (
	"reflect"
	"testing"
)

func Test_ConvertOption_error(t *testing.T) {
	tests := []struct {
		name string
		opt  any
	}{
		{"value that cannot be marshaled", map[string]any{"ch": make(chan int)}},
		{"value that is not an object", "plain text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := ConvertOption(tt.opt); res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_ConvertOption_roundTrip(t *testing.T) {
	res := ConvertOption(OptionFetch{EventURL: "http://example.invalid/x", EventMethod: "POST"})
	expected := map[string]any{"event_url": "http://example.invalid/x", "event_method": "POST"}
	if !reflect.DeepEqual(res, expected) {
		t.Errorf("Wrong result. expect: %v, got: %v", expected, res)
	}
}

func Test_ParseOption_error(t *testing.T) {
	tests := []struct {
		name string
		opt  map[string]any
	}{
		{"option that cannot be marshaled", map[string]any{"ch": make(chan int)}},
		{"option with a wrong field type", map[string]any{"anonymous": 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target OptionCall
			if err := ParseOption(tt.opt, &target); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if target.Anonymous != "" {
				t.Errorf("The target is touched. got: %q", target.Anonymous)
			}
		})
	}
}
