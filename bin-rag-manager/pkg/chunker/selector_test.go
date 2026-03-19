package chunker

import (
	"testing"
)

func TestGetChunkerByExtension(t *testing.T) {
	tests := []struct {
		ext     string
		wantNil bool
	}{
		{".rst", false},
		{".md", false},
		{".yaml", false},
		{".yml", false},
		{".txt", false},
		{".pdf", false},
		{".html", false},
		{".htm", false},
		{".csv", false},
		{".docx", false},
		{".json", false},
		{".unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			c := GetChunkerByExtension(tt.ext)
			if (c == nil) != tt.wantNil {
				t.Errorf("GetChunkerByExtension(%q) nil=%v, want nil=%v", tt.ext, c == nil, tt.wantNil)
			}
		})
	}
}

func TestDetectExtensionFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"document.pdf", ".pdf"},
		{"readme.md", ".md"},
		{"data.CSV", ".csv"},
		{"noext", ""},
		{"archive.tar.gz", ".gz"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := DetectExtensionFromFilename(tt.filename)
			if got != tt.want {
				t.Errorf("DetectExtensionFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestDetectExtensionFromContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        string
	}{
		{"application/pdf", ".pdf"},
		{"text/html", ".html"},
		{"text/html; charset=utf-8", ".html"},
		{"text/csv", ".csv"},
		{"application/json", ".json"},
		{"application/octet-stream", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			got := DetectExtensionFromContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("DetectExtensionFromContentType(%q) = %q, want %q", tt.contentType, got, tt.want)
			}
		})
	}
}
