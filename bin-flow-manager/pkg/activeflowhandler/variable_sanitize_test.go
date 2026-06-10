package activeflowhandler

import (
	"strings"
	"testing"
)

func Test_sanitizeInitialVariables(t *testing.T) {
	tests := []struct {
		name string

		in map[string]string

		expectKeys    []string // keys expected present in the result
		expectDropped []string // keys expected absent from the result
		expectEmpty   bool     // whole injection rejected -> empty result
	}{
		{
			name:        "nil input",
			in:          nil,
			expectEmpty: true,
		},
		{
			name:       "plain keys pass through",
			in:         map[string]string{"campaign_id": "summer", "intent": "renewal"},
			expectKeys: []string{"campaign_id", "intent"},
		},
		{
			name:          "reserved voipbin prefix dropped, others kept",
			in:            map[string]string{"voipbin.activeflow.complete_count": "0", "campaign_id": "x"},
			expectKeys:    []string{"campaign_id"},
			expectDropped: []string{"voipbin.activeflow.complete_count"},
		},
		{
			name:          "reserved prefix case-insensitive and trimmed",
			in:            map[string]string{"  VOIPBIN.activeflow.id  ": "forged", "ok": "1"},
			expectKeys:    []string{"ok"},
			expectDropped: []string{"VOIPBIN.activeflow.id", "  VOIPBIN.activeflow.id  "},
		},
		{
			name:          "empty key dropped",
			in:            map[string]string{"   ": "v", "good": "1"},
			expectKeys:    []string{"good"},
			expectDropped: []string{"   "},
		},
		{
			name:          "key is trimmed and stored trimmed",
			in:            map[string]string{"  spaced  ": "v"},
			expectKeys:    []string{"spaced"},
			expectDropped: []string{"  spaced  "},
		},
		{
			name:        "oversize value rejects whole injection",
			in:          map[string]string{"big": strings.Repeat("a", initialVariablesMaxValueBytes+1), "small": "1"},
			expectEmpty: true,
		},
		{
			name:        "too many keys rejects whole injection",
			in:          makeNKeys(initialVariablesMaxKeyCount + 1),
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &activeflowHandler{}
			got := h.sanitizeInitialVariables(tt.in)

			if tt.expectEmpty {
				if len(got) != 0 {
					t.Errorf("expected empty result, got %v", got)
				}
				return
			}

			for _, k := range tt.expectKeys {
				if _, ok := got[k]; !ok {
					t.Errorf("expected key %q present, result: %v", k, got)
				}
			}
			for _, k := range tt.expectDropped {
				if _, ok := got[k]; ok {
					t.Errorf("expected key %q dropped, result: %v", k, got)
				}
			}
		})
	}
}

func makeNKeys(n int) map[string]string {
	m := make(map[string]string, n)
	for i := 0; i < n; i++ {
		m["k"+string(rune('a'+i%26))+string(rune('0'+i/26))] = "v"
	}
	return m
}
