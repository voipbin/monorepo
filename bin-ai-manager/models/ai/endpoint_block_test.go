package ai

import (
	"encoding/json"
	"net/netip"
	"os"
	"sort"
	"testing"
)

type blocklistFixture struct {
	V4    []string `json:"v4"`
	V6    []string `json:"v6"`
	NAT64 []string `json:"nat64"`
	Cases []struct {
		Addr    string `json:"addr"`
		Blocked bool   `json:"blocked"`
	} `json:"cases"`
}

func loadBlocklistFixture(t *testing.T) *blocklistFixture {
	t.Helper()

	raw, err := os.ReadFile("testdata/endpoint_blocklist.json")
	if err != nil {
		t.Fatalf("could not read the fixture. err: %v", err)
	}

	var res blocklistFixture
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("could not parse the fixture. err: %v", err)
	}

	return &res
}

func normalizePrefixes(t *testing.T, in []string) []string {
	t.Helper()

	res := make([]string, 0, len(in))
	for _, s := range in {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			t.Fatalf("bad prefix %q. err: %v", s, err)
		}
		res = append(res, p.Masked().String())
	}
	sort.Strings(res)

	return res
}

func prefixStrings(in []netip.Prefix) []string {
	res := make([]string, 0, len(in))
	for _, p := range in {
		res = append(res, p.Masked().String())
	}
	sort.Strings(res)

	return res
}

func Test_EndpointBlockTables_MatchFixture(t *testing.T) {
	f := loadBlocklistFixture(t)

	tests := []struct {
		name   string
		fixure []string
		code   []netip.Prefix
	}{
		{"v4", f.V4, blockedV4Prefixes},
		{"v6", f.V6, blockedV6Prefixes},
		{"nat64", f.NAT64, []netip.Prefix{nat64Prefix}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := normalizePrefixes(t, tt.fixure)
			got := prefixStrings(tt.code)

			if len(want) != len(got) {
				t.Fatalf("table size differs. expect: %v, got: %v", want, got)
			}
			for i := range want {
				if want[i] != got[i] {
					t.Fatalf("table differs. expect: %v, got: %v", want, got)
				}
			}
		})
	}
}

func Test_EndpointBlockTables_Corpus(t *testing.T) {
	f := loadBlocklistFixture(t)

	if len(f.Cases) < 100 {
		t.Fatalf("the corpus is too small: %d", len(f.Cases))
	}

	for _, c := range f.Cases {
		t.Run(c.Addr, func(t *testing.T) {
			if got := isBlockedAddrString(c.Addr); got != c.Blocked {
				t.Errorf("addr %q. expect blocked: %v, got: %v", c.Addr, c.Blocked, got)
			}
		})
	}
}

func Test_EndpointBlock_EmbeddedV4Decides(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		blocked bool
	}{
		{"mapped_public", "::ffff:8.8.8.8", false},
		{"mapped_private", "::ffff:10.1.2.3", true},
		{"mapped_metadata", "::ffff:169.254.169.254", true},
		{"nat64_public", "64:ff9b::808:808", false},
		{"nat64_private", "64:ff9b::a01:203", true},
		{"6to4_public", "2002:808:808::", false},
		{"6to4_private", "2002:a01:203::", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBlockedAddrString(tt.addr); got != tt.blocked {
				t.Errorf("expect: %v, got: %v", tt.blocked, got)
			}
		})
	}
}
