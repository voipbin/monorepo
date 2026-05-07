package outboundconfig_test

import (
	"strings"
	"testing"

	"github.com/dongri/phonenumber"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

// TestISOMapDrift asserts that countries returned by the phonenumber library
// for a representative E.164 set are present in the local ISOCountryCodes map.
// If this test fails after a phonenumber library update, add the new codes to iso.go.
//
// Note: phonenumber.Parse(number, "") returns "" for numbers with a "+" prefix and
// empty country — this matches the library's documented behaviour. We supply the digits
// directly (no "+" prefix) to GetISO3166ByNumber, mirroring how the service normalises
// numbers internally (the "+" is stripped before database/cache storage).
func TestISOMapDrift(t *testing.T) {
	samples := []struct {
		// digits is the E.164 number without the leading "+".
		// Numbers must be mobile numbers recognised by the phonenumber library.
		// Landline numbers for several countries (DE, JP, KR, RU) are not in the
		// library's dataset, so we use mobile numbers throughout.
		digits      string
		wantCountry string
	}{
		{"12025550100", "us"},
		{"14165550100", "ca"},
		{"447911123456", "gb"},  // UK mobile (07x -> 447x)
		{"33612345678", "fr"},   // FR mobile (06x/07x -> 336x/337x)
		{"491511234567", "de"},  // DE mobile (015x)
		{"819012345678", "jp"},  // JP mobile (090x)
		{"821012345678", "kr"},  // KR mobile (010x)
		{"861012345678", "cn"},
		{"911234567890", "in"},
		{"5511987654321", "br"},
		{"61412345678", "au"},   // AU mobile (04x -> 614x)
		{"27721234567", "za"},   // ZA mobile (072x)
		{"971501234567", "ae"},
		{"79161234567", "ru"},   // RU mobile (9xx)
	}

	for _, s := range samples {
		iso := phonenumber.GetISO3166ByNumber(s.digits, true)
		got := strings.ToLower(iso.Alpha2)
		if got != s.wantCountry {
			t.Errorf("phonenumber returned %q for %s, expected %q — update test fixtures", got, s.digits, s.wantCountry)
			continue
		}
		if _, ok := outboundconfig.ISOCountryCodes[got]; !ok {
			t.Errorf("ISOCountryCodes is missing %q (returned for %s) — add it to iso.go", got, s.digits)
		}
	}
}
