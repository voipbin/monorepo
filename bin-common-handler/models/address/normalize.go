package address

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors returned by NormalizeTarget. Callers may errors.Is on them.
var (
	// ErrUnknownType indicates addressType is not a known enum value. This is
	// almost always a programming error (a misconfigured or empty Type at a call
	// site). The original target is returned unchanged alongside this error.
	ErrUnknownType = errors.New("unknown address type")

	// ErrNotNormalizable indicates the type is known but the value cannot be
	// reduced to a canonical form for it. This is a legitimate domain case, e.g.
	// a tel carrier sentinel like "anonymous"/"Restricted" that contains no
	// digit. The ORIGINAL value is returned unchanged; the caller may store it
	// verbatim.
	ErrNotNormalizable = errors.New("value not normalizable for type")
)

// NormalizeTarget returns the canonical form of target for addressType.
// It is the single canonicalization authority for address targets across the
// monorepo.
//
// Loss-proof contract: the first return value is ALWAYS safe to store. If the
// input can be canonicalized, it is the canonical form with a nil error. If it
// cannot, the ORIGINAL input is returned unchanged with a non-nil sentinel
// error (ErrNotNormalizable for a known type, ErrUnknownType for an unknown
// type). NormalizeTarget never blanks a meaningful non-empty input, so a caller
// that discards the error never loses data.
//
// It is idempotent in both value and error class:
//
//	NormalizeTarget(t, NormalizeTarget(t, x)) returns the same value and error
//	class as NormalizeTarget(t, x).
//
// NormalizeTarget does NOT validate. Callers run ValidateTarget separately on
// the canonical form where validation is required (Normalize THEN Validate).
func NormalizeTarget(addressType Type, target string) (string, error) {
	switch addressType {
	case TypeTel, TypeWhatsApp:
		return normalizeTel(target)
	case TypeEmail:
		return normalizeEmail(target), nil
	case TypeSIP:
		return normalizeSIP(target), nil
	case TypeNone, TypeAgent, TypeAI, TypeAITeam, TypeConference, TypeExtension, TypeLine, TypeWebchat:
		// Opaque identifiers (UUIDs, names) with no sub-form to canonicalize.
		// Identity normalization keeps them unchanged and idempotent.
		return target, nil
	default:
		// Unknown type: return the original value (loss-proof) plus a sentinel.
		return target, fmt.Errorf("%w: %s", ErrUnknownType, addressType)
	}
}

// normalizeTel canonicalizes an E.164-style phone number (also used for
// whatsapp identifiers, which are E.164 phone numbers).
//
// Order is load-bearing and matches the legacy bin-contact-manager
// normalizeE164 loop: (1) trim whitespace FIRST, (2) keep a '+' only if it is at
// index 0 of the trimmed string, (3) keep ASCII digits [0-9] only, strip
// everything else. Trimming after the index-0 scan would drop a leading '+' on
// whitespace-prefixed input.
//
// ASCII-only digits (not unicode.IsDigit): the downstream tel validator is the
// ASCII regex ^\+[0-9]{7,15}$, so a retained non-ASCII digit would
// normalize-pass then validate-fail. Pinning ASCII keeps normalization output
// aligned with the validator's alphabet.
//
// Loss-proof: if the canonical form contains NO digit (e.g. a carrier sentinel
// like "anonymous"/"Restricted", an alphanumeric sender id, a bare "+", or an
// empty input), the ORIGINAL input is returned unchanged with ErrNotNormalizable
// so a legitimate non-numeric telecom value is never blanked.
func normalizeTel(target string) (string, error) {
	src := strings.TrimSpace(target)

	var b strings.Builder
	hasDigit := false
	for i, r := range src {
		switch {
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			hasDigit = true
		}
	}

	if !hasDigit {
		// No digit to canonicalize: preserve the original value verbatim.
		return target, fmt.Errorf("%w: tel %q", ErrNotNormalizable, target)
	}
	return b.String(), nil
}

// normalizeEmail canonicalizes an email address by trimming surrounding
// whitespace and lowercasing. This matches the existing contact-manager
// behavior (trim+lowercase of the raw string). Display-name forms such as
// "User <user@host>" are lowercased as-is; parsing them is a validation concern,
// not a normalization one. Lossless: always returns a nil error.
func normalizeEmail(target string) string {
	return strings.ToLower(strings.TrimSpace(target))
}

// normalizeSIP canonicalizes a sip target. Per RFC 3261 the host part is
// case-insensitive while the user part is case-sensitive, and param/header
// values are case-sensitive.
//
// Steps: (1) trim the whole input; (2) locate the params boundary = the first
// ';' or '?' (or end of string); (3) split the pre-boundary segment on its LAST
// '@' so userinfo "user:pass@host" keeps the password verbatim and a '@' inside
// a header/param value is not mistaken for the userinfo delimiter; (4) preserve
// the user part verbatim; (5) lowercase only the host token (port digits are
// safe); (6) preserve the ';params'/'?headers' tail verbatim. If there is no
// '@' before the params boundary, the trimmed input is returned unchanged.
// Lossless: always returns a nil error.
func normalizeSIP(target string) string {
	s := strings.TrimSpace(target)
	if s == "" {
		return ""
	}

	// Find the params/headers boundary.
	boundary := len(s)
	if i := strings.IndexAny(s, ";?"); i >= 0 {
		boundary = i
	}
	head := s[:boundary] // user@host:port
	tail := s[boundary:] // ;params / ?headers (preserved verbatim)

	at := strings.LastIndex(head, "@")
	if at < 0 {
		// No userinfo delimiter before the params boundary; nothing to
		// canonicalize safely, return the trimmed input unchanged.
		return s
	}

	user := head[:at+1] // includes the trailing '@'
	host := head[at+1:] // host[:port]
	return user + strings.ToLower(host) + tail
}
