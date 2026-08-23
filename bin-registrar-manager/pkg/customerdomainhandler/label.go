package customerdomainhandler

import (
	"crypto/rand"
	"fmt"
)

const (
	// labelCharset is the allowed label alphabet: base36 lowercase.
	labelCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

	// labelLength is the generated label length.
	labelLength = 4

	// labelGenerateMaxAttempts bounds the duplicate-key retry loop.
	// 36^4 = 1,679,616 combinations — collisions are rare; the bound only
	// guards against a pathological DB state.
	labelGenerateMaxAttempts = 10
)

// reservedLabels can never be assigned as a customer domain label. With
// exactly-4-char labels only "pstn" and "echo" can genuinely collide; the full
// list is kept as cheap future-proofing for any later length change.
var reservedLabels = map[string]struct{}{
	"pstn": {},
	"sip":  {},
	"echo": {},
	"reg":  {},
	"www":  {},
	"api":  {},
}

// isReservedLabel reports whether the given label is reserved.
func isReservedLabel(label string) bool {
	_, ok := reservedLabels[label]
	return ok
}

// generateLabel returns a random 4-char base36 lowercase label, re-drawing
// when the generated label is reserved.
func generateLabel() (string, error) {
	for {
		label, err := generateLabelOnce()
		if err != nil {
			return "", err
		}

		if isReservedLabel(label) {
			continue
		}

		return label, nil
	}
}

// generateLabelOnce draws labelLength crypto-random chars from labelCharset.
func generateLabelOnce() (string, error) {
	buf := make([]byte, labelLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("could not read random bytes. generateLabelOnce. err: %v", err)
	}

	for i, b := range buf {
		buf[i] = labelCharset[int(b)%len(labelCharset)]
	}

	return string(buf), nil
}
