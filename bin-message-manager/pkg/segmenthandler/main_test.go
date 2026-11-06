package segmenthandler

import (
	"strings"
	"testing"
)

func Test_Count(t *testing.T) {
	ext := "{" // an extension character, 2 septets

	tests := []struct {
		name string
		text string

		expectEncoding Encoding
		expectUnits    int
		expectSegments int
	}{
		{"empty", "", EncodingGSM7, 0, 1},
		{"gsm7 160", strings.Repeat("a", 160), EncodingGSM7, 160, 1},
		{"gsm7 161", strings.Repeat("a", 161), EncodingGSM7, 161, 2},
		{"gsm7 306", strings.Repeat("a", 306), EncodingGSM7, 306, 2},
		{"gsm7 307", strings.Repeat("a", 307), EncodingGSM7, 307, 3},
		{"gsm7 1530", strings.Repeat("a", 1530), EncodingGSM7, 1530, 10},
		{"gsm7 1531", strings.Repeat("a", 1531), EncodingGSM7, 1531, 11},
		{"ucs2 hangul 70", strings.Repeat("가", 70), EncodingUCS2, 70, 1},
		{"ucs2 hangul 71", strings.Repeat("가", 71), EncodingUCS2, 71, 2},
		{"ucs2 134", strings.Repeat("가", 134), EncodingUCS2, 134, 2},
		{"ucs2 135", strings.Repeat("가", 135), EncodingUCS2, 135, 3},
		{"ucs2 670", strings.Repeat("가", 670), EncodingUCS2, 670, 10},
		{"ucs2 671", strings.Repeat("가", 671), EncodingUCS2, 671, 11},
		{"one non gsm7 char switches the whole text", strings.Repeat("a", 69) + "가", EncodingUCS2, 70, 1},
		{"one non gsm7 char at 71", strings.Repeat("a", 70) + "가", EncodingUCS2, 71, 2},
		{"backtick is outside gsm7", "`", EncodingUCS2, 1, 1},
		{"typographic quote is outside gsm7", "it\u2019s", EncodingUCS2, 4, 1},
		{"extension char starting at septet 159 totals 160", strings.Repeat("a", 158) + ext, EncodingGSM7, 160, 1},
		{"extension char starting at septet 160 totals 161", strings.Repeat("a", 159) + ext, EncodingGSM7, 161, 2},
		{"form feed is an extension char", strings.Repeat("a", 158) + "\f", EncodingGSM7, 160, 1},
		{"euro is an extension char", "\u20ac", EncodingGSM7, 2, 1},
		{"80 extension chars", strings.Repeat(ext, 80), EncodingGSM7, 160, 1},
		{"81 extension chars", strings.Repeat(ext, 81), EncodingGSM7, 162, 2},
		{"emoji 35", strings.Repeat("\U0001F600", 35), EncodingUCS2, 70, 1},
		{"emoji 36", strings.Repeat("\U0001F600", 36), EncodingUCS2, 72, 2},
		{"emoji starting at unit 69 totals 70", strings.Repeat("가", 68) + "\U0001F600", EncodingUCS2, 70, 1},
		{"emoji starting at unit 70 totals 71", strings.Repeat("가", 69) + "\U0001F600", EncodingUCS2, 71, 2},
		{"invalid utf8 is replaced", "a\xffb", EncodingUCS2, 3, 1},
		// greedy packing: 152 basic septets, an extension char that does not fit the 153rd septet, 150 basic, one more
		// extension char that does not fit segment 2. 306 septets, a plain division gives 2, the packing gives 3.
		{"greedy packing", strings.Repeat("a", 152) + ext + strings.Repeat("a", 150) + ext, EncodingGSM7, 306, 3},
		// emoji at the segment boundary of UCS-2: 66 units, then an emoji (2 units) does not fit the 67th unit.
		{"surrogate pair is not split", strings.Repeat("가", 66) + "\U0001F600" + strings.Repeat("가", 66), EncodingUCS2, 134, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Count(tt.text)
			if res.Encoding != tt.expectEncoding {
				t.Errorf("Wrong encoding. expect: %s, got: %s", tt.expectEncoding, res.Encoding)
			}
			if res.Units != tt.expectUnits {
				t.Errorf("Wrong units. expect: %d, got: %d", tt.expectUnits, res.Units)
			}
			if res.Segments != tt.expectSegments {
				t.Errorf("Wrong segments. expect: %d, got: %d", tt.expectSegments, res.Segments)
			}
		})
	}
}
