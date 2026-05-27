package geminiaudithandler

import (
	"testing"
)

func TestStripMarkdownFence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain json unchanged",
			input: `{"overall_score":4}`,
			want:  `{"overall_score":4}`,
		},
		{
			name:  "json fence stripped",
			input: "```json\n{\"overall_score\":4}\n```",
			want:  `{"overall_score":4}`,
		},
		{
			name:  "plain fence stripped",
			input: "```\n{\"overall_score\":4}\n```",
			want:  `{"overall_score":4}`,
		},
		{
			name:  "leading and trailing whitespace trimmed",
			input: "  ```json\n{\"overall_score\":4}\n```  ",
			want:  `{"overall_score":4}`,
		},
		{
			name:  "no closing fence handled",
			input: "```json\n{\"overall_score\":4}",
			want:  `{"overall_score":4}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(stripMarkdownFence([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("stripMarkdownFence(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.want)
			}
		})
	}
}
