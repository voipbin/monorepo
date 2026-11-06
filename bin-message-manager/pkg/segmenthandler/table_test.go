package segmenthandler

import "testing"

// defaultAlphabet is the GSM 7 bit default alphabet of 3GPP TS 23.038 section 6.2.1 written as code points by septet
// value, so that it does not depend on the string of the implementation. 0 at 0x1b is the escape code.
var defaultAlphabet = [128]rune{
	0x40, 0xA3, 0x24, 0xA5, 0xE8, 0xE9, 0xF9, 0xEC, 0xF2, 0xC7, 0x0A, 0xD8, 0xF8, 0x0D, 0xC5, 0xE5,
	0x394, 0x5F, 0x3A6, 0x393, 0x39B, 0x3A9, 0x3A0, 0x3A8, 0x3A3, 0x398, 0x39E, 0, 0xC6, 0xE6, 0xDF, 0xC9,
	0x20, 0x21, 0x22, 0x23, 0xA4, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F,
	0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D, 0x3E, 0x3F,
	0xA1, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F,
	0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5A, 0xC4, 0xD6, 0xD1, 0xDC, 0xA7,
	0xBF, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6A, 0x6B, 0x6C, 0x6D, 0x6E, 0x6F,
	0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7A, 0xE4, 0xF6, 0xF1, 0xFC, 0xE0,
}

// extensionTable is the extension table of section 6.2.1.1: form feed, ^, {, }, \, [, ~, ], | and the euro sign.
var extensionTable = []rune{0x0C, 0x5E, 0x7B, 0x7D, 0x5C, 0x5B, 0x7E, 0x5D, 0x7C, 0x20AC}

// Test_Count_alphabetTables checks every character of both tables one by one, so that a character that is missing from
// or added to a table of the implementation changes a result.
func Test_Count_alphabetTables(t *testing.T) {
	known := map[rune]bool{}

	for septet, r := range defaultAlphabet {
		if septet == 0x1b {
			continue
		}
		known[r] = true

		res := Count(string(r))
		if res.Encoding != EncodingGSM7 || res.Units != 1 || res.Segments != 1 {
			t.Errorf("Wrong default alphabet character. septet: 0x%02x, rune: U+%04X, got: %v", septet, r, res)
		}
	}

	for _, r := range extensionTable {
		known[r] = true

		res := Count(string(r))
		if res.Encoding != EncodingGSM7 || res.Units != 2 || res.Segments != 1 {
			t.Errorf("Wrong extension character. rune: U+%04X, got: %v", r, res)
		}
	}

	if len(known) != 127+len(extensionTable) {
		t.Fatalf("The test tables are wrong. known: %d", len(known))
	}

	// every other character of the Latin-1 range, the C0 controls and the surrounding blocks is outside the alphabet.
	for r := rune(0); r < 0x500; r++ {
		if known[r] {
			continue
		}

		if res := Count(string(r)); res.Encoding != EncodingUCS2 || res.Units != 1 {
			t.Errorf("A character outside the alphabet was accepted. rune: U+%04X, got: %v", r, res)
		}
	}
}

// Test_Count_surrogateBoundary checks the last code point of the basic plane (1 unit) and the first one beyond it
// (a surrogate pair, 2 units).
func Test_Count_surrogateBoundary(t *testing.T) {
	if res := Count("\uffff"); res.Units != 1 {
		t.Errorf("Wrong units for U+FFFF. got: %v", res)
	}
	if res := Count("\U00010000"); res.Units != 2 {
		t.Errorf("Wrong units for U+10000. got: %v", res)
	}
}
