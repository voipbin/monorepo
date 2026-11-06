// Package segmenthandler counts the SMS segments of a text from the international standard (3GPP TS 23.038
// and TS 23.040), whatever a provider reports.
package segmenthandler

// Encoding is the character set that a text needs.
type Encoding string

// list of Encoding
const (
	EncodingGSM7 Encoding = "gsm7"
	EncodingUCS2 Encoding = "ucs2"
)

// Limits of one segment (TS 23.038 section 4 and TS 23.040 section 9.2.3.24.1, 8 bit reference).
const (
	singleGSM7 = 160 // septets of a single message
	singleUCS2 = 70  // UTF-16 code units of a single message
	concatGSM7 = 153 // septets of a segment of a concatenated message (140 - 6 octets of header, 7 bit aligned)
	concatUCS2 = 67  // UTF-16 code units of a segment of a concatenated message
)

// Result is the count of a text.
type Result struct {
	Encoding Encoding // GSM-7 when every character is in the default alphabet or its extension table, UCS-2 otherwise
	Units    int      // septets for GSM-7 (an extension character is 2), UTF-16 code units for UCS-2
	Segments int      // number of segments, 1 for an empty text
}

// Count returns the encoding, the units and the segments of the text.
//
// One character outside the GSM 7 bit alphabet switches the whole text to UCS-2. An escape pair (an
// extension character) and a surrogate pair are never split between two segments, so the segments are
// packed greedily and the result can be larger than a plain division of the units. An invalid UTF-8 byte
// is read as U+FFFD, which is not in the alphabet.
func Count(text string) Result {
	costs := make([]int, 0, len(text))
	gsm7 := true
	septets := 0
	for _, r := range text {
		c, ok := gsm7Cost(r)
		if !ok {
			gsm7 = false
			break
		}
		costs = append(costs, c)
		septets += c
	}

	if gsm7 {
		return Result{Encoding: EncodingGSM7, Units: septets, Segments: segments(costs, septets, singleGSM7, concatGSM7)}
	}

	costs = costs[:0]
	units := 0
	for _, r := range text {
		c := 1
		if r > 0xFFFF {
			c = 2
		}
		costs = append(costs, c)
		units += c
	}

	return Result{Encoding: EncodingUCS2, Units: units, Segments: segments(costs, units, singleUCS2, concatUCS2)}
}

// segments packs the characters (each with its unit cost) into segments.
func segments(costs []int, total int, single int, concat int) int {
	if total <= single {
		return 1
	}

	res := 1
	used := 0
	for _, c := range costs {
		if used+c > concat {
			res++
			used = 0
		}
		used += c
	}

	return res
}

// gsm7Cost returns the septets of the rune: 1 for the default alphabet, 2 for the extension table and false
// for a rune outside both.
func gsm7Cost(r rune) (int, bool) {
	if _, ok := gsm7Extension[r]; ok {
		return 2, true
	}
	if _, ok := gsm7Basic[r]; ok {
		return 1, true
	}

	return 0, false
}

// gsm7Basic is the default alphabet of TS 23.038 section 6.2.1 (the escape code is not a character).
var gsm7Basic = func() map[rune]struct{} {
	m := map[rune]struct{}{}
	for _, r := range "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà" {
		m[r] = struct{}{}
	}
	return m
}()

// gsm7Extension is the extension table of TS 23.038 section 6.2.1.1: each character costs the escape and the character.
var gsm7Extension = func() map[rune]struct{} {
	m := map[rune]struct{}{}
	for _, r := range "\f^{}\\[~]|€" {
		m[r] = struct{}{}
	}
	return m
}()
