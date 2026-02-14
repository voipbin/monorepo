package utilhandler

import (
	"net/url"
	"strings"
	"testing"
)

func TestURLMergeFilters(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		filters map[string]string
		check   func(t *testing.T, result string)
	}{
		{
			name: "merge single filter",
			uri:  "http://example.com?existing=param",
			filters: map[string]string{
				"name": "value",
			},
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "filter_name=value") {
					t.Errorf("URLMergeFilters() missing filter_name=value in: %s", result)
				}
			},
		},
		{
			name:    "empty filters returns original URI",
			uri:     "http://example.com",
			filters: map[string]string{},
			check: func(t *testing.T, result string) {
				if result != "http://example.com" {
					t.Errorf("URLMergeFilters() with empty filters should return original URI, got: %s", result)
				}
			},
		},
		{
			name: "merge multiple filters",
			uri:  "http://example.com",
			filters: map[string]string{
				"status": "active",
				"type":   "user",
			},
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "filter_status=active") {
					t.Errorf("URLMergeFilters() missing filter_status in: %s", result)
				}
				if !strings.Contains(result, "filter_type=user") {
					t.Errorf("URLMergeFilters() missing filter_type in: %s", result)
				}
			},
		},
		{
			name: "URL encodes special characters",
			uri:  "http://example.com",
			filters: map[string]string{
				"email": "test@example.com",
			},
			check: func(t *testing.T, result string) {
				// @ should be encoded as %40
				if !strings.Contains(result, "test%40example.com") {
					t.Errorf("URLMergeFilters() should URL encode special characters, got: %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLMergeFilters(tt.uri, tt.filters)
			tt.check(t, result)
		})
	}
}

func TestURLParseFilters_EmptyQueryParams(t *testing.T) {
	u, err := url.Parse("http://example.com?filter_empty=")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	result := URLParseFilters(u)

	// Should still parse filter even with empty value
	if val, ok := result["empty"]; !ok {
		t.Error("URLParseFilters() should parse filter_empty")
	} else if val != "" {
		t.Errorf("URLParseFilters() filter_empty value = %q, want empty string", val)
	}
}
