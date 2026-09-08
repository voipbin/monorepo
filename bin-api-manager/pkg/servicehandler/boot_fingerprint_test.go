package servicehandler

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// Test_directHashFingerprint_isKeyed is the regression guard for the keying
// itself.
//
// The other fingerprint assertions in this package derive their expected value
// by calling directHashFingerprint, so reverting it to a bare digest would
// leave them green. This one does not: it pins that the output depends on the
// signing key and is not the plain SHA-256 of the hash.
//
// Why it matters: the underlying direct hash is 6 random bytes in a known
// format (48 bits), and the whole *auth.AuthIdentity -- fingerprint included --
// is logged on the WebSocket paths. An unkeyed digest is brute-forceable
// offline from a log line, and recovering the hash that way outlives both the
// token's expiry and the boot session ceiling.
func Test_directHashFingerprint_isKeyed(t *testing.T) {
	const hash = "direct.a1b2c3d4e5f6"

	h1 := serviceHandler{jwtKey: []byte("key-one")}
	h2 := serviceHandler{jwtKey: []byte("key-two")}

	got1 := h1.directHashFingerprint(hash)
	got2 := h2.directHashFingerprint(hash)

	if got1 == got2 {
		t.Error("Fingerprint does not depend on the signing key; it is not keyed")
	}

	// Deterministic for a given key: the refresh check compares a value minted
	// on one replica against one computed on another.
	if again := h1.directHashFingerprint(hash); again != got1 {
		t.Errorf("Fingerprint is not deterministic. first: %s, second: %s", got1, again)
	}

	unkeyed := sha256.Sum256([]byte(hash))
	if got1 == hex.EncodeToString(unkeyed[:]) {
		t.Error("Fingerprint is a plain SHA-256 of the hash")
	}

	// Full width, no truncation: the input has only 48 bits of entropy, so a
	// shortened digest would narrow the search further.
	if len(got1) != sha256.Size*2 {
		t.Errorf("Wrong fingerprint length. expect: %d, got: %d", sha256.Size*2, len(got1))
	}
}

// Test_directHashFingerprint_goldenVector pins the exact derivation, including
// the domain-separation prefix.
//
// The prefix is otherwise only ever compared against itself, so removing it
// would leave every other test green -- and it is baked into live token claims.
// A silent change to it would 401 every in-flight refresh at deploy, because
// the stored fingerprint would no longer match the recomputed one.
//
// If this test fails after an intentional derivation change, that change is a
// breaking one: it invalidates outstanding tokens and needs a scope version
// bump alongside it.
func Test_directHashFingerprint_goldenVector(t *testing.T) {
	h := serviceHandler{jwtKey: []byte("golden-key")}

	const (
		hash = "direct.a1b2c3d4e5f6"
		want = "e6db509129fa40d00c82c270fc915a34af1cad3c5b4e65a033fb9d99787c5566"
	)

	if got := h.directHashFingerprint(hash); got != want {
		t.Errorf("Fingerprint derivation changed.\nexpect: %s\ngot:    %s", want, got)
	}
}
