package store

import (
	"os"
	"testing"

	"monorepo/bin-rag-manager/pkg/chunker"
)

func TestMemoryStore_AddAndSearch(t *testing.T) {
	s := NewMemoryStore()

	chunks := []chunker.Chunk{
		{ID: "1", Text: "How to create a call", SourceFile: "call.rst", DocType: chunker.DocTypeDevDoc, SectionTitle: "Creating calls"},
		{ID: "2", Text: "POST /calls endpoint", SourceFile: "openapi.yaml", DocType: chunker.DocTypeOpenAPI, SectionTitle: "POST /calls"},
		{ID: "3", Text: "Architecture overview", SourceFile: "design.md", DocType: chunker.DocTypeDesign, SectionTitle: "Architecture"},
	}

	embeddings := [][]float32{
		{1.0, 0.0, 0.0},
		{0.9, 0.1, 0.0},
		{0.0, 0.0, 1.0},
	}

	s.Add(chunks, embeddings)

	stats := s.Stats()
	if stats.ChunkCount != 3 {
		t.Errorf("expected 3 chunks, got %d", stats.ChunkCount)
	}

	// Search with query vector close to first two chunks
	results := s.Search([]float32{1.0, 0.0, 0.0}, 2, nil)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if results[0].Chunk.ID != "1" {
		t.Errorf("expected first result to be chunk 1, got %s", results[0].Chunk.ID)
	}

	// Search with doc type filter
	results = s.Search([]float32{1.0, 0.0, 0.0}, 10, []chunker.DocType{chunker.DocTypeOpenAPI})
	if len(results) != 1 {
		t.Errorf("expected 1 result with filter, got %d", len(results))
	}
	if results[0].Chunk.DocType != chunker.DocTypeOpenAPI {
		t.Errorf("expected openapi doc type, got %s", results[0].Chunk.DocType)
	}
}

func TestMemoryStore_DeleteByFile(t *testing.T) {
	s := NewMemoryStore()

	chunks := []chunker.Chunk{
		{ID: "1", Text: "chunk 1", SourceFile: "file1.rst", DocType: chunker.DocTypeDevDoc, SectionTitle: "Section 1"},
		{ID: "2", Text: "chunk 2", SourceFile: "file1.rst", DocType: chunker.DocTypeDevDoc, SectionTitle: "Section 2"},
		{ID: "3", Text: "chunk 3", SourceFile: "file2.rst", DocType: chunker.DocTypeDevDoc, SectionTitle: "Section 3"},
	}

	embeddings := [][]float32{
		{1.0, 0.0},
		{0.0, 1.0},
		{0.5, 0.5},
	}

	s.Add(chunks, embeddings)

	s.DeleteByFile("file1.rst")

	stats := s.Stats()
	if stats.ChunkCount != 1 {
		t.Errorf("expected 1 chunk after delete, got %d", stats.ChunkCount)
	}
}

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "store_test_*.gob")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	// Create and save store
	s1 := NewMemoryStore()
	chunks := []chunker.Chunk{
		{ID: "1", Text: "test chunk", SourceFile: "test.rst", DocType: chunker.DocTypeDevDoc, SectionTitle: "Test"},
	}
	embeddings := [][]float32{{1.0, 0.5, 0.3}}
	s1.Add(chunks, embeddings)

	if err := s1.Save(tmpFile.Name()); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Load into new store
	s2 := NewMemoryStore()
	if err := s2.Load(tmpFile.Name()); err != nil {
		t.Fatalf("load error: %v", err)
	}

	stats := s2.Stats()
	if stats.ChunkCount != 1 {
		t.Errorf("expected 1 chunk after load, got %d", stats.ChunkCount)
	}

	results := s2.Search([]float32{1.0, 0.5, 0.3}, 1, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Chunk.Text != "test chunk" {
		t.Errorf("expected 'test chunk', got '%s'", results[0].Chunk.Text)
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if (result - tt.expected) > 0.001 || (result-tt.expected) < -0.001 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}
