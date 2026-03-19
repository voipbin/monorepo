package chunker

import (
	"os"
	"strings"
	"testing"
)

func TestHTMLChunker_Chunk(t *testing.T) {
	content := `<html><body><h1>Title</h1><p>Paragraph one.</p><h2>Section</h2><p>Paragraph two.</p></body></html>`

	tmpFile, err := os.CreateTemp("", "test_*.html")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewHTMLChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}

	for _, chunk := range chunks {
		if strings.Contains(chunk.Text, "<html>") {
			t.Error("expected HTML tags to be stripped")
		}
	}
}

func TestHTMLChunker_NonExistentFile(t *testing.T) {
	c := NewHTMLChunker()
	_, err := c.Chunk("/nonexistent/file.html", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
