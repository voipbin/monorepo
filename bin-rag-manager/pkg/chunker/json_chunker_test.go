package chunker

import (
	"os"
	"testing"
)

func TestJSONChunker_Object(t *testing.T) {
	content := `{"users": [{"name": "Alice"}, {"name": "Bob"}], "settings": {"theme": "dark"}}`

	tmpFile, err := os.CreateTemp("", "test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewJSONChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}
}

func TestJSONChunker_Array(t *testing.T) {
	content := `[{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]`

	tmpFile, err := os.CreateTemp("", "test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewJSONChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}
}

func TestJSONChunker_NonExistentFile(t *testing.T) {
	c := NewJSONChunker()
	_, err := c.Chunk("/nonexistent/file.json", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
