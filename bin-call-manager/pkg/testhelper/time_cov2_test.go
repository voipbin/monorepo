package testhelper

import (
	"reflect"
	"testing"
	"time"
)

func Test_TimePtr_cov2_formats(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect *time.Time
	}{
		{"empty", "", nil},
		{"invalid", "not-a-time", nil},
		{"micro z", "2020-04-18T03:22:17.995000Z", ptr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC))},
		{"micro no z", "2020-04-18T03:22:17.995000", ptr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC))},
		{"milli z", "2020-04-18T03:22:17.995Z", ptr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC))},
		{"sec z", "2020-04-18T03:22:17Z", ptr(time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC))},
		{"sec", "2020-04-18T03:22:17", ptr(time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC))},
		{"rfc3339 offset", "2020-04-18T03:22:17+09:00", ptr(time.Date(2020, 4, 17, 18, 22, 17, 0, time.UTC))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := TimePtr(tt.input)
			if tt.expect == nil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}
			if res == nil || !res.Equal(*tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}

func ptr(v time.Time) *time.Time { return &v }

func Test_TimeMust_cov2(t *testing.T) {
	res := TimeMust("2020-04-18T03:22:17Z")
	expect := time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC)
	if !reflect.DeepEqual(*res, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, *res)
	}
}

func Test_TimeMust_cov2_panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	TimeMust("bad")
}
