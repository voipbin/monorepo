package asteriskaddress

import (
	"testing"
	"time"
)

func Test_Key(t *testing.T) {
	tests := []struct {
		name string

		id string

		expect string
	}{
		{
			name: "mac_style_id",

			id: "3e:50:6b:43:bb:32",

			expect: "asterisk.3e:50:6b:43:bb:32.address-internal",
		},
		{
			name: "empty_id",

			id: "",

			expect: "asterisk..address-internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := Key(tt.id); res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

func Test_ParseKey(t *testing.T) {
	tests := []struct {
		name string

		key string

		expectID string
		expectOK bool
	}{
		{
			name: "normal",

			key: "asterisk.3e:50:6b:43:bb:32.address-internal",

			expectID: "3e:50:6b:43:bb:32",
			expectOK: true,
		},
		{
			name: "id_containing_dots",

			key: "asterisk.172.24.0.101.address-internal",

			expectID: "172.24.0.101",
			expectOK: true,
		},
		{
			name: "empty_id_is_rejected",

			key: "asterisk..address-internal",

			expectID: "",
			expectOK: false,
		},
		{
			name: "wrong_prefix",

			key: "kamailio.3e:50:6b:43:bb:32.address-internal",

			expectID: "",
			expectOK: false,
		},
		{
			name: "wrong_suffix",

			key: "asterisk.3e:50:6b:43:bb:32.address-external",

			expectID: "",
			expectOK: false,
		},
		{
			name: "empty_key",

			key: "",

			expectID: "",
			expectOK: false,
		},
		{
			name: "prefix_and_suffix_overlap",

			key: "asterisk.address-internal",

			expectID: "",
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resID, resOK := ParseKey(tt.key)

			if resOK != tt.expectOK {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectOK, resOK)
			}
			if resID != tt.expectID {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectID, resID)
			}
		})
	}
}

func Test_KeyParseKeyRoundTrip(t *testing.T) {
	ids := []string{"3e:50:6b:43:bb:32", "72:ce:24:e6:51:2f", "a"}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			res, ok := ParseKey(Key(id))
			if !ok {
				t.Fatalf("Wrong match. expect: ok, got: not ok")
			}
			if res != id {
				t.Errorf("Wrong match. expect: %s, got: %s", id, res)
			}
		})
	}
}

// Test_IsFresh pins the freshness boundary EXACTLY at `TTL - FreshnessMargin` (24h - 12min),
// including the just-above / just-below / exactly-on cases. This threshold is the single knob
// separating "this pass learned the current occupant of the IP" from "this pass learned nothing",
// so an off-by-one in the comparison direction is the failure this test exists to catch.
func Test_IsFresh(t *testing.T) {
	threshold := TTL - FreshnessMargin // 23h48m

	tests := []struct {
		name string

		ttl time.Duration

		expect bool
	}{
		{
			name: "full_ttl_just_set",

			ttl: TTL,

			expect: true,
		},
		{
			name: "exactly_on_the_threshold_is_fresh",

			ttl: threshold,

			expect: true,
		},
		{
			name: "one_nanosecond_above_the_threshold",

			ttl: threshold + time.Nanosecond,

			expect: true,
		},
		{
			name: "one_nanosecond_below_the_threshold",

			ttl: threshold - time.Nanosecond,

			expect: false,
		},
		{
			name: "two_missed_refreshes_still_fresh",

			ttl: TTL - 2*RefreshInterval,

			expect: true,
		},
		{
			name: "one_second_after_two_missed_refreshes_still_fresh",

			ttl: TTL - 2*RefreshInterval - time.Second,

			expect: true,
		},
		{
			name: "three_missed_refreshes_is_stale",

			ttl: TTL - 3*RefreshInterval,

			expect: false,
		},
		{
			name: "dead_generation_half_a_day_old",

			ttl: 12 * time.Hour,

			expect: false,
		},
		{
			name: "zero_ttl",

			ttl: 0,

			expect: false,
		},
		{
			name: "negative_ttl_redis_no_expire_sentinel",

			ttl: -1,

			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AsteriskAddress{ID: "3e:50:6b:43:bb:32", Address: "172.24.0.101", TTL: tt.ttl}

			if res := a.IsFresh(); res != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v (ttl: %v, threshold: %v)", tt.expect, res, tt.ttl, threshold)
			}
		})
	}
}

// Test_FreshnessMarginSpansMultipleRefreshIntervals pins the design rationale itself: the margin
// must tolerate at least two consecutive missed proxy refreshes. Shrinking it back toward one
// interval is exactly the regression design review rejected.
func Test_FreshnessMarginSpansMultipleRefreshIntervals(t *testing.T) {
	if FreshnessMargin < 2*RefreshInterval {
		t.Errorf("Wrong match. expect: FreshnessMargin >= 2 refresh intervals (%v), got: %v", 2*RefreshInterval, FreshnessMargin)
	}
	if FreshnessMargin >= TTL {
		t.Errorf("Wrong match. expect: FreshnessMargin < TTL (%v), got: %v", TTL, FreshnessMargin)
	}
}

func Test_KeyPattern(t *testing.T) {
	if KeyPattern != "asterisk.*.address-internal" {
		t.Errorf("Wrong match. expect: asterisk.*.address-internal, got: %s", KeyPattern)
	}
}
