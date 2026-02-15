package chunker

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestDocType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		docType  DocType
		expected string
	}{
		{"devdoc", DocTypeDevDoc, "devdoc"},
		{"openapi", DocTypeOpenAPI, "openapi"},
		{"design", DocTypeDesign, "design"},
		{"guideline", DocTypeGuideline, "guideline"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.docType) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.docType))
			}
		})
	}
}

func TestChunk_Fields(t *testing.T) {
	chunk := Chunk{
		ID:           "test-id",
		Text:         "test text",
		SourceFile:   "test.md",
		DocType:      DocTypeDesign,
		SectionTitle: "Test Section",
	}

	if chunk.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", chunk.ID)
	}
	if chunk.Text != "test text" {
		t.Errorf("expected Text 'test text', got %q", chunk.Text)
	}
	if chunk.SourceFile != "test.md" {
		t.Errorf("expected SourceFile 'test.md', got %q", chunk.SourceFile)
	}
	if chunk.DocType != DocTypeDesign {
		t.Errorf("expected DocType 'design', got %q", chunk.DocType)
	}
	if chunk.SectionTitle != "Test Section" {
		t.Errorf("expected SectionTitle 'Test Section', got %q", chunk.SectionTitle)
	}
}

func TestGenerateChunkID(t *testing.T) {
	tests := []struct {
		name         string
		sourceFile   string
		sectionTitle string
	}{
		{
			name:         "simple",
			sourceFile:   "test.md",
			sectionTitle: "Section 1",
		},
		{
			name:         "with special chars",
			sourceFile:   "path/to/file.md",
			sectionTitle: "Section: Special!",
		},
		{
			name:         "empty section",
			sourceFile:   "test.md",
			sectionTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := generateChunkID(tt.sourceFile, tt.sectionTitle)

			// Check ID is not empty
			if id == "" {
				t.Error("expected non-empty ID")
			}

			// Check ID length (should be 16 chars from hex)
			if len(id) != 16 {
				t.Errorf("expected ID length 16, got %d", len(id))
			}

			// Verify consistency - same input produces same ID
			id2 := generateChunkID(tt.sourceFile, tt.sectionTitle)
			if id != id2 {
				t.Error("expected same ID for same input")
			}

			// Verify it matches expected hash
			h := sha256.New()
			h.Write([]byte(tt.sourceFile + "|" + tt.sectionTitle))
			expected := fmt.Sprintf("%x", h.Sum(nil))[:16]
			if id != expected {
				t.Errorf("expected ID %q, got %q", expected, id)
			}
		})
	}
}

func TestGenerateChunkID_Different(t *testing.T) {
	id1 := generateChunkID("file1.md", "Section 1")
	id2 := generateChunkID("file2.md", "Section 1")
	id3 := generateChunkID("file1.md", "Section 2")

	if id1 == id2 {
		t.Error("expected different IDs for different files")
	}
	if id1 == id3 {
		t.Error("expected different IDs for different sections")
	}
}
