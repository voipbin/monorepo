package embedder

import (
	"context"
	"testing"
)

// mockEmbedder is a mock implementation for testing
type mockEmbedder struct {
	embedTextsFunc func(ctx context.Context, texts []string) ([][]float32, error)
	embedTextFunc  func(ctx context.Context, text string) ([]float32, error)
}

func (m *mockEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedTextsFunc != nil {
		return m.embedTextsFunc(ctx, texts)
	}
	// Default implementation
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{0.1, 0.2, 0.3}
	}
	return result, nil
}

func (m *mockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if m.embedTextFunc != nil {
		return m.embedTextFunc(ctx, text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func TestEmbedder_Interface(t *testing.T) {
	var _ Embedder = &mockEmbedder{}
	var _ Embedder = &openaiEmbedder{}
}

func TestNewOpenAIEmbedder(t *testing.T) {
	embedder := NewOpenAIEmbedder("test-api-key", "text-embedding-ada-002")
	if embedder == nil {
		t.Error("expected non-nil embedder")
	}
}

func TestOpenAIEmbedder_EmbedTexts_EmptyInput(t *testing.T) {
	// Create a custom embedder for testing
	e := &openaiEmbedder{}

	ctx := context.Background()
	result, err := e.EmbedTexts(ctx, []string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for empty input")
	}
}

func TestMockEmbedder_EmbedTexts(t *testing.T) {
	tests := []struct {
		name    string
		texts   []string
		wantLen int
	}{
		{
			name:    "single text",
			texts:   []string{"test text"},
			wantLen: 1,
		},
		{
			name:    "multiple texts",
			texts:   []string{"text 1", "text 2", "text 3"},
			wantLen: 3,
		},
		{
			name:    "empty texts",
			texts:   []string{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockEmbedder{}
			ctx := context.Background()

			results, err := m.EmbedTexts(ctx, tt.texts)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(results) != tt.wantLen {
				t.Errorf("expected %d results, got %d", tt.wantLen, len(results))
			}
		})
	}
}

func TestMockEmbedder_EmbedText(t *testing.T) {
	m := &mockEmbedder{}
	ctx := context.Background()

	result, err := m.EmbedText(ctx, "test text")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty embedding")
	}
}

func TestMockEmbedder_CustomFunctions(t *testing.T) {
	customResult := [][]float32{{1.0, 2.0}, {3.0, 4.0}}
	m := &mockEmbedder{
		embedTextsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			return customResult, nil
		},
	}

	ctx := context.Background()
	result, err := m.EmbedTexts(ctx, []string{"test1", "test2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
	if result[0][0] != 1.0 || result[1][0] != 3.0 {
		t.Error("custom function not called correctly")
	}
}
