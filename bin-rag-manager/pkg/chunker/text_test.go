package chunker

import (
	"os"
	"strings"
	"testing"
)

func TestTextChunker_Chunk(t *testing.T) {
	content := "This is a plain text document.\n\nIt has multiple paragraphs.\n\nEach paragraph should be chunked properly."

	tmpFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewTextChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}

	var allText strings.Builder
	for _, chunk := range chunks {
		allText.WriteString(chunk.Text)
	}
	if !strings.Contains(allText.String(), "plain text document") {
		t.Error("expected content to be preserved")
	}
}

func TestTextChunker_NonExistentFile(t *testing.T) {
	c := NewTextChunker()
	_, err := c.Chunk("/nonexistent/file.txt", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestTextChunker_LargeFile(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("This is paragraph number ")
		sb.WriteString(strings.Repeat("word ", 50))
		sb.WriteString("\n\n")
	}

	tmpFile, err := os.CreateTemp("", "test_large_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(sb.String()); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewTextChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 100)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks for large file, got %d", len(chunks))
	}
}
