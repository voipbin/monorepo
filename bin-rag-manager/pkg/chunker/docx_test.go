package chunker

import (
	"testing"
)

func TestDOCXChunker_NonExistentFile(t *testing.T) {
	c := NewDOCXChunker()
	_, err := c.Chunk("/nonexistent/file.docx", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
