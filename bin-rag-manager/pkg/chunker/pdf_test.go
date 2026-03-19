package chunker

import (
	"testing"
)

func TestPDFChunker_NonExistentFile(t *testing.T) {
	c := NewPDFChunker()
	_, err := c.Chunk("/nonexistent/file.pdf", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
