package summaryhandler

import (
	"fmt"
	"strings"
	"testing"
)

// Test_canonPrimarySubtag covers the BCP47 primary-subtag canonicalization,
// including the strings.ToLower step: without case-folding, "KO-KR"/"EN" would
// not canonicalize to "ko"/"en" and shouldVerify would misclassify them (this is
// the R4-3 regression the shared harness fixtures do not catch, since they only
// use already-lowercase primary subtags "ko"/"en").
func Test_canonPrimarySubtag(t *testing.T) {
	tests := []struct {
		name string

		lang string

		expectRes string
	}{
		{"lowercase region", "ko-KR", "ko"},
		{"uppercase primary and region", "KO-KR", "ko"},
		{"uppercase primary only", "EN", "en"},
		{"lowercase primary with region", "en-US", "en"},
		{"empty", "", ""},
		{"mixed case latin", "Fr-FR", "fr"},
		{"primary only lowercase", "ja", "ja"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := canonPrimarySubtag(tt.lang)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectRes, res)
			}
		})
	}
}

// Test_shouldVerify covers the shouldVerify decision across case variants:
// English targets (any case) are skipped, every non-English target is verified,
// and empty is skipped. The uppercase cases ("KO-KR", "EN") are what pin the
// strings.ToLower canonicalization: removing it would flip these and this test
// would fail (the shared harness fixtures would still pass).
func Test_shouldVerify(t *testing.T) {
	tests := []struct {
		name string

		outputLanguage string

		expectRes bool
	}{
		{"ko lower region", "ko-KR", true},
		{"ko upper region", "KO-KR", true},
		{"en lower region", "en-US", false},
		{"en upper primary", "EN", false},
		{"ja primary only", "ja", true},
		{"fr upper region", "fr-FR", true},
		{"empty skipped", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &summaryHandler{}
			res := h.shouldVerify(tt.outputLanguage)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_proseLen covers the prose-length accounting used by the min-prose
// verification guard: header lines (ending in ':') and "- None" items are
// excluded, normal hyphen items count their post-hyphen rune length, and the
// exactly-20-rune boundary (languageVerifyMinProse) is pinned so the guard's
// comparison (< languageVerifyMinProse) is exercised at its edge.
func Test_proseLen(t *testing.T) {
	tests := []struct {
		name string

		content string

		expectRes int
	}{
		{
			name:      "empty",
			content:   "",
			expectRes: 0,
		},
		{
			name:      "headers only",
			content:   "Call Type:\nKey Discussion Points:",
			expectRes: 0,
		},
		{
			name:      "none items excluded",
			content:   "Call Type:\n- None\n\nKey Discussion Points:\n- None",
			expectRes: 0,
		},
		{
			name:      "none is case insensitive",
			content:   "Key Discussion Points:\n- none\n- NONE",
			expectRes: 0,
		},
		{
			name:      "empty hyphen item excluded",
			content:   "Key Discussion Points:\n- ",
			expectRes: 0,
		},
		{
			// "hello world" = 11 runes (only the text after "- " counts).
			name:      "single prose item",
			content:   "Key Discussion Points:\n- hello world",
			expectRes: 11,
		},
		{
			// Exactly 20 runes: the languageVerifyMinProse boundary. proseLen must
			// return exactly 20 so the guard (proseLen < 20) does NOT skip.
			name:      "exactly twenty runes boundary",
			content:   "Points:\n- 12345678901234567890",
			expectRes: 20,
		},
		{
			// 19 runes: just below the boundary; guard skips.
			name:      "nineteen runes below boundary",
			content:   "Points:\n- 1234567890123456789",
			expectRes: 19,
		},
		{
			// Multibyte prose counts runes, not bytes. Korean "고객이 환불을 요청했습니다" is
			// counted per rune.
			name:      "multibyte counts runes",
			content:   "Key Discussion Points:\n- 안녕하세요",
			expectRes: 5,
		},
		{
			// Translated "None" (Korean "없음") is NOT excluded: only literal English
			// "None" is filtered (see proseLen's tradeoff doc). "없음" = 2 runes.
			name:      "translated none counts as prose",
			content:   "Key Discussion Points:\n- 없음",
			expectRes: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := proseLen(tt.content)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectRes, res)
			}
		})
	}
}

// Test_languageVerifyPrompt_shape asserts the verifier prompt template embeds
// both the BCP47 code and the sample when formatted, matching the payload the
// harness assertions expect (defense against silently reordering the fmt verbs).
func Test_languageVerifyPrompt_shape(t *testing.T) {
	got := fmt.Sprintf(languageVerifyPrompt, "ko-KR", "sample text")
	if !strings.Contains(got, "ko-KR") {
		t.Errorf("expect prompt to contain BCP47 code, got %q", got)
	}
	if !strings.Contains(got, "sample text") {
		t.Errorf("expect prompt to contain sample, got %q", got)
	}
	if !strings.Contains(got, "language detector") {
		t.Errorf("expect prompt to contain language detector fragment, got %q", got)
	}
}
