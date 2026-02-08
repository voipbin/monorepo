package chunker

import (
	"os"
	"testing"
)

func TestRSTChunker_Chunk(t *testing.T) {
	content := `Introduction
============

This is the introduction section with some text.

Getting Started
===============

This section describes how to get started.

Step 1: Install the software.

Step 2: Configure the settings.

Advanced Usage
==============

This section covers advanced usage.
`

	tmpFile, err := os.CreateTemp("", "test_*.rst")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewRSTChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) < 3 {
		t.Errorf("expected at least 3 chunks, got %d", len(chunks))
	}

	// Verify doc type
	for _, chunk := range chunks {
		if chunk.DocType != DocTypeDevDoc {
			t.Errorf("expected doc type devdoc, got %s", chunk.DocType)
		}
	}

	// Verify section titles are extracted
	titles := make(map[string]bool)
	for _, chunk := range chunks {
		titles[chunk.SectionTitle] = true
	}
	if !titles["Introduction"] {
		t.Error("expected 'Introduction' section")
	}
	if !titles["Getting Started"] {
		t.Error("expected 'Getting Started' section")
	}
	if !titles["Advanced Usage"] {
		t.Error("expected 'Advanced Usage' section")
	}
}

func TestRSTChunker_NonExistentFile(t *testing.T) {
	c := NewRSTChunker()
	_, err := c.Chunk("/nonexistent/file.rst", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestSplitByTokenLimit(t *testing.T) {
	// Create text longer than 10 tokens (40 chars)
	text := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph with more content here."

	chunks := splitByTokenLimit(text, 10)
	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(chunks))
	}
}

func TestIsRSTUnderline(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"====", true},
		{"----", true},
		{"~~~~", true},
		{"===", true},
		{"==", false},     // too short
		{"abc", false},    // not underline chars
		{"==a=", false},   // mixed chars
		{"", false},       // empty
	}

	for _, tt := range tests {
		result := isRSTUnderline(tt.line, "=-~^`")
		if result != tt.expected {
			t.Errorf("isRSTUnderline(%q) = %v, want %v", tt.line, result, tt.expected)
		}
	}
}
