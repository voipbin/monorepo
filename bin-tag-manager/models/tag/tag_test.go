package tag

import (
	"testing"
)

func TestTag(t *testing.T) {
	tests := []struct {
		name string

		tagName string
		detail  string
	}{
		{
			name: "creates_tag_with_all_fields",

			tagName: "VIP Customer",
			detail:  "High value customer tag",
		},
		{
			name: "creates_tag_with_empty_fields",

			tagName: "",
			detail:  "",
		},
		{
			name: "creates_tag_with_special_characters",

			tagName: "Customer-Tag_123",
			detail:  "Tag with special chars: !@#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := &Tag{
				Name:   tt.tagName,
				Detail: tt.detail,
			}

			if tag.Name != tt.tagName {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.tagName, tag.Name)
			}
			if tag.Detail != tt.detail {
				t.Errorf("Wrong Detail. expect: %s, got: %s", tt.detail, tag.Detail)
			}
		})
	}
}
