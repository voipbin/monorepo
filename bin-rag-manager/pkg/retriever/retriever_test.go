package retriever

import (
	"context"
	"errors"
	"testing"

	"monorepo/bin-rag-manager/pkg/chunker"
	"monorepo/bin-rag-manager/pkg/store"
)

// mockEmbedder for testing
type mockEmbedder struct {
	embedTextFunc func(ctx context.Context, text string) ([]float32, error)
}

func (m *mockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if m.embedTextFunc != nil {
		return m.embedTextFunc(ctx, text)
	}
	return []float32{1.0, 0.0, 0.0}, nil
}

func (m *mockEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		emb, err := m.EmbedText(ctx, texts[i])
		if err != nil {
			return nil, err
		}
		result[i] = emb
	}
	return result, nil
}

// mockStore for testing
type mockStore struct {
	searchFunc func(queryEmbedding []float32, topK int, docTypes []chunker.DocType) []store.SearchResult
}

func (m *mockStore) Search(queryEmbedding []float32, topK int, docTypes []chunker.DocType) []store.SearchResult {
	if m.searchFunc != nil {
		return m.searchFunc(queryEmbedding, topK, docTypes)
	}
	return []store.SearchResult{
		{
			Chunk: chunker.Chunk{
				ID:           "1",
				Text:         "test chunk",
				SourceFile:   "test.md",
				DocType:      chunker.DocTypeDesign,
				SectionTitle: "Test",
			},
			RelevanceScore: 0.95,
		},
	}
}

func (m *mockStore) Add(chunks []chunker.Chunk, embeddings [][]float32) {}

func (m *mockStore) DeleteByFile(sourceFile string) {}

func (m *mockStore) Save(filePath string) error {
	return nil
}

func (m *mockStore) Load(filePath string) error {
	return nil
}

func (m *mockStore) Stats() store.StoreStats {
	return store.StoreStats{ChunkCount: 1}
}

func TestRetriever_Interface(t *testing.T) {
	var _ Retriever = &retriever{}
}

func TestNewRetriever(t *testing.T) {
	emb := &mockEmbedder{}
	st := &mockStore{}

	r := NewRetriever(emb, st)
	if r == nil {
		t.Error("expected non-nil retriever")
	}
}

func TestRetriever_Query_Success(t *testing.T) {
	emb := &mockEmbedder{
		embedTextFunc: func(ctx context.Context, text string) ([]float32, error) {
			return []float32{1.0, 0.5, 0.3}, nil
		},
	}

	st := &mockStore{
		searchFunc: func(queryEmbedding []float32, topK int, docTypes []chunker.DocType) []store.SearchResult {
			return []store.SearchResult{
				{
					Chunk: chunker.Chunk{
						ID:           "1",
						Text:         "result 1",
						SourceFile:   "file1.md",
						DocType:      chunker.DocTypeDesign,
						SectionTitle: "Section 1",
					},
					RelevanceScore: 0.95,
				},
				{
					Chunk: chunker.Chunk{
						ID:           "2",
						Text:         "result 2",
						SourceFile:   "file2.md",
						DocType:      chunker.DocTypeOpenAPI,
						SectionTitle: "Section 2",
					},
					RelevanceScore: 0.85,
				},
			}
		},
	}

	r := NewRetriever(emb, st)
	ctx := context.Background()

	results, err := r.Query(ctx, "test query", 5, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestRetriever_Query_WithDocTypeFilter(t *testing.T) {
	emb := &mockEmbedder{}
	st := &mockStore{
		searchFunc: func(queryEmbedding []float32, topK int, docTypes []chunker.DocType) []store.SearchResult {
			// Verify docTypes filter is passed through
			if len(docTypes) != 2 {
				return nil
			}
			return []store.SearchResult{
				{
					Chunk: chunker.Chunk{
						ID:           "1",
						Text:         "openapi result",
						SourceFile:   "api.yaml",
						DocType:      chunker.DocTypeOpenAPI,
						SectionTitle: "API",
					},
					RelevanceScore: 0.90,
				},
			}
		},
	}

	r := NewRetriever(emb, st)
	ctx := context.Background()

	docTypes := []chunker.DocType{chunker.DocTypeOpenAPI, chunker.DocTypeDevDoc}
	results, err := r.Query(ctx, "test", 10, docTypes)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Chunk.DocType != chunker.DocTypeOpenAPI {
		t.Error("expected OpenAPI doc type")
	}
}

func TestRetriever_Query_EmbedError(t *testing.T) {
	expectedErr := errors.New("embedding failed")
	emb := &mockEmbedder{
		embedTextFunc: func(ctx context.Context, text string) ([]float32, error) {
			return nil, expectedErr
		},
	}
	st := &mockStore{}

	r := NewRetriever(emb, st)
	ctx := context.Background()

	_, err := r.Query(ctx, "test query", 5, nil)
	if err == nil {
		t.Error("expected error from embedder")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap embedding error, got: %v", err)
	}
}

func TestRetriever_Query_EmptyResults(t *testing.T) {
	emb := &mockEmbedder{}
	st := &mockStore{
		searchFunc: func(queryEmbedding []float32, topK int, docTypes []chunker.DocType) []store.SearchResult {
			return []store.SearchResult{}
		},
	}

	r := NewRetriever(emb, st)
	ctx := context.Background()

	results, err := r.Query(ctx, "test", 5, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestRetriever_Query_TopKParameter(t *testing.T) {
	emb := &mockEmbedder{}

	topKReceived := 0
	st := &mockStore{
		searchFunc: func(queryEmbedding []float32, topK int, docTypes []chunker.DocType) []store.SearchResult {
			topKReceived = topK
			return []store.SearchResult{}
		},
	}

	r := NewRetriever(emb, st)
	ctx := context.Background()

	expectedTopK := 15
	_, err := r.Query(ctx, "test", expectedTopK, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if topKReceived != expectedTopK {
		t.Errorf("expected topK=%d to be passed to store, got %d", expectedTopK, topKReceived)
	}
}

func TestRetriever_Query_ContextCancellation(t *testing.T) {
	emb := &mockEmbedder{
		embedTextFunc: func(ctx context.Context, text string) ([]float32, error) {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return []float32{1.0}, nil
			}
		},
	}
	st := &mockStore{}

	r := NewRetriever(emb, st)

	// Create and cancel context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.Query(ctx, "test", 5, nil)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}
