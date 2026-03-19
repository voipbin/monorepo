package chunker

import (
	"os"
	"testing"
)

func TestCSVChunker_Chunk(t *testing.T) {
	content := "name,email,role\nAlice,alice@example.com,admin\nBob,bob@example.com,user\nCharlie,charlie@example.com,user\n"

	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewCSVChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}

	for _, chunk := range chunks {
		if chunk.Text == "" {
			t.Error("expected non-empty chunk text")
		}
	}
}

func TestCSVChunker_NonExistentFile(t *testing.T) {
	c := NewCSVChunker()
	_, err := c.Chunk("/nonexistent/file.csv", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
