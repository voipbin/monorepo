package mcpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func testTable() CarrierTable {
	return CarrierTable{
		"/fake":  {Carrier: "fake", Credential: CredentialClientIDSecret},
		"/fake2": {Carrier: "fake2", Credential: CredentialAPIKey},
	}
}

func Test_IsReservedHost(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		expect bool
	}{
		{"plain", "https://carrier.invalid/fake", true},
		{"upper case", "https://CARRIER.INVALID/fake", true},
		{"trailing dots", "https://carrier.invalid../fake", true},
		{"ideographic full stop", "https://carrier\u3002invalid/fake", true},
		{"fullwidth full stop", "https://carrier\uff0einvalid/fake", true},
		{"halfwidth ideographic full stop", "https://carrier\uff61invalid/fake", true},
		{"no path", "https://carrier.invalid", true},
		{"with port", "https://carrier.invalid:443/fake", true},
		{"turkish dotted capital i folds to the reserved host and stays off the network", "https://carr\u0130er.invalid/fake", true},
		{"other invalid host", "https://example.invalid/fake", false},
		{"sub domain", "https://x.carrier.invalid/fake", false},
		{"ordinary host", "https://mcp.example.com/fake", false},
		{"not a url", "://%%", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReservedHost(tt.url); got != tt.expect {
				t.Errorf("IsReservedHost(%q) = %v, expected %v", tt.url, got, tt.expect)
			}
		})
	}
}

func Test_InternalCarrier(t *testing.T) {
	table := testTable()

	tests := []struct {
		name          string
		table         CarrierTable
		url           string
		expectOK      bool
		expectCarrier string
	}{
		{"canonical", table, "https://carrier.invalid/fake", true, "fake"},
		{"second entry", table, "https://carrier.invalid/fake2", true, "fake2"},
		{"upper case host", table, "https://CARRIER.invalid/fake", true, "fake"},
		{"trailing dot host", table, "https://carrier.invalid./fake", true, "fake"},
		{"ideographic full stop host", table, "https://carrier\u3002invalid/fake", true, "fake"},
		{"trailing slash", table, "https://carrier.invalid/fake/", false, ""},
		{"query", table, "https://carrier.invalid/fake?x=1", false, ""},
		{"empty query marker", table, "https://carrier.invalid/fake?", false, ""},
		{"fragment", table, "https://carrier.invalid/fake#a", false, ""},
		{"user info", table, "https://user@carrier.invalid/fake", false, ""},
		{"port", table, "https://carrier.invalid:443/fake", false, ""},
		{"empty port", table, "https://carrier.invalid:/fake", false, ""},
		{"encoded path", table, "https://carrier.invalid/%66ake", false, ""},
		{"upper case path", table, "https://carrier.invalid/FAKE", false, ""},
		{"unknown path", table, "https://carrier.invalid/unknown", false, ""},
		{"no path", table, "https://carrier.invalid", false, ""},
		{"http scheme", table, "http://carrier.invalid/fake", false, ""},
		{"ordinary host", table, "https://mcp.example.com/fake", false, ""},
		{"empty fragment marker", table, "https://carrier.invalid/fake#", false, ""},
		{"turkish dotted capital i is not folded to i", table, "https://carr\u0130er.invalid/fake", false, ""},
		{"empty table", CarrierTable{}, "https://carrier.invalid/fake", false, ""},
		{"nil table", nil, "https://carrier.invalid/fake", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := tt.table.InternalCarrier(tt.url)
			if ok != tt.expectOK || entry.Carrier != tt.expectCarrier {
				t.Errorf("InternalCarrier(%q) = (%q, %v), expected (%q, %v)", tt.url, entry.Carrier, ok, tt.expectCarrier, tt.expectOK)
			}
		})
	}
}

func Test_DefaultCarrierTable(t *testing.T) {
	table := DefaultCarrierTable()
	if len(table) != 3 {
		t.Fatalf("the production table holds the open carriers only, got %d entries", len(table))
	}
	for path, carrier := range map[string]string{"/fedex": "fedex", "/ups": "ups"} {
		if entry := table[path]; entry.Carrier != carrier || entry.Credential != CredentialClientIDSecret {
			t.Errorf("unexpected line for %s: %+v", path, entry)
		}
		if entry, ok := table.InternalCarrier("https://carrier.invalid" + path); !ok || entry.Carrier != carrier {
			t.Errorf("the %s url must be recognized, got %+v %v", carrier, entry, ok)
		}
	}
	for _, closed := range []string{"https://carrier.invalid/unlisted"} {
		if _, ok := table.InternalCarrier(closed); ok {
			t.Errorf("%s must not be recognized while it is not listed", closed)
		}
	}
	if entry := table["/dhl"]; entry.Carrier != "dhl" || entry.Credential != CredentialAPIKey {
		t.Errorf("unexpected dhl line %+v", entry)
	}
	if entry, ok := table.InternalCarrier("https://carrier.invalid/dhl"); !ok || entry.Carrier != "dhl" {
		t.Errorf("the dhl url must be recognized, got %+v %v", entry, ok)
	}
	if got := table.AllowedURLs(); fmt.Sprint(got) != fmt.Sprint([]string{"https://carrier.invalid/dhl", "https://carrier.invalid/fedex", "https://carrier.invalid/ups"}) {
		t.Errorf("unexpected allowed urls %v", got)
	}

	// each call returns a new map, so a caller cannot change another caller's table.
	table["/x"] = CarrierEntry{Carrier: "x"}
	delete(table, "/ups")
	delete(table, "/dhl")
	if again := DefaultCarrierTable(); len(again) != 3 || again["/ups"].Carrier != "ups" || again["/dhl"].Carrier != "dhl" {
		t.Errorf("DefaultCarrierTable must return a new map on every call")
	}
	if _, ok := DefaultCarrierTable()["/x"]; ok {
		t.Errorf("a line added by a caller must not show in the next table")
	}
}

func Test_AllowedURLs(t *testing.T) {
	got := testTable().AllowedURLs()
	expect := []string{"https://carrier.invalid/fake", "https://carrier.invalid/fake2"}
	if fmt.Sprint(got) != fmt.Sprint(expect) {
		t.Errorf("AllowedURLs() = %v, expected %v", got, expect)
	}
}

func Test_NormalizeHost(t *testing.T) {
	tests := []struct {
		in     string
		expect string
	}{
		{"CARRIER.Invalid", "carrier.invalid"},
		{"CARR\u0130ER.Invalid", "carrier.invalid"},
		{"carrier.invalid...", "carrier.invalid"},
		{"carrier\u3002invalid\uff0e", "carrier.invalid"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeHost(tt.in); got != tt.expect {
			t.Errorf("NormalizeHost(%q) = %q, expected %q", tt.in, got, tt.expect)
		}
	}
}

func Test_ParseCredential(t *testing.T) {
	tests := []struct {
		name         string
		format       CredentialFormat
		raw          string
		expectErr    bool
		expectID     string
		expectSecret string
	}{
		{"client id and secret", CredentialClientIDSecret, "id1:sec1", false, "id1", "sec1"},
		{"trims outer space", CredentialClientIDSecret, "  id1:sec1\n", false, "id1", "sec1"},
		{"splits on the first colon only", CredentialClientIDSecret, "id1:sec:1", false, "id1", "sec:1"},
		{"no colon", CredentialClientIDSecret, "id1sec1", true, "", ""},
		{"empty id", CredentialClientIDSecret, ":sec1", true, "", ""},
		{"empty secret", CredentialClientIDSecret, "id1:", true, "", ""},
		{"both empty", CredentialClientIDSecret, ":", true, "", ""},
		{"inner space", CredentialClientIDSecret, "id 1:sec1", true, "", ""},
		{"control character", CredentialClientIDSecret, "id1:se\x00c1", true, "", ""},
		{"non breaking space", CredentialClientIDSecret, "id1:sec\u00a01", true, "", ""},
		{"zero width space", CredentialClientIDSecret, "id1:sec\u200b1", true, "", ""},
		{"non ascii", CredentialClientIDSecret, "id1:sec\u00e91", true, "", ""},
		{"empty", CredentialClientIDSecret, "   ", true, "", ""},
		{"too long", CredentialClientIDSecret, "a:" + strings.Repeat("b", 600), true, "", ""},
		{"api key", CredentialAPIKey, " key1 ", false, "", "key1"},
		{"api key keeps colon", CredentialAPIKey, "k:1", false, "", "k:1"},
		{"api key inner space", CredentialAPIKey, "k 1", true, "", ""},
		{"api key empty", CredentialAPIKey, "", true, "", ""},
		{"unknown format", CredentialFormat("x"), "abc", true, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred, err := ParseCredential(tt.format, tt.raw)
			if tt.expectErr {
				if !errors.Is(err, ErrCredentialMalformed) {
					t.Fatalf("expected ErrCredentialMalformed, got %v", err)
				}
				if err != nil && strings.Contains(err.Error(), "sec") {
					t.Errorf("the error must not quote the input: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cred.ID != tt.expectID || cred.Secret != tt.expectSecret {
				t.Errorf("got (%q, %q), expected (%q, %q)", cred.ID, cred.Secret, tt.expectID, tt.expectSecret)
			}
		})
	}
}

func Test_Credential_isRedactedEverywhere(t *testing.T) {
	sentinel := "SENTINEL-SECRET-VALUE"
	cred := Credential{ID: "SENTINEL-ID-VALUE", Secret: sentinel}

	b, err := json.Marshal(cred)
	if err != nil {
		t.Fatal(err)
	}
	text, err := cred.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	outputs := []string{
		cred.String(), cred.GoString(), string(b), string(text),
		fmt.Sprintf("%v", cred), fmt.Sprintf("%+v", cred), fmt.Sprintf("%#v", cred), fmt.Sprintf("%q", cred),
		fmt.Sprintf("%v", &cred), fmt.Sprintf("%+v", map[string]any{"c": cred}),
		cred.LogValue().String(),
	}
	for _, out := range outputs {
		if strings.Contains(out, "SENTINEL") {
			t.Errorf("a credential leaked into %q", out)
		}
	}

	var _ slog.LogValuer = cred
}
