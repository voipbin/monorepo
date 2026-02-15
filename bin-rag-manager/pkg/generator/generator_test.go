package generator

import (
	"context"
	"strings"
	"testing"

	"monorepo/bin-rag-manager/pkg/chunker"
	"monorepo/bin-rag-manager/pkg/store"
)

// mockGenerator is a mock implementation for testing
type mockGenerator struct {
	generateFunc func(ctx context.Context, query string, chunks []store.SearchResult) (string, error)
}

func (m *mockGenerator) Generate(ctx context.Context, query string, chunks []store.SearchResult) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, query, chunks)
	}
	return "mock answer", nil
}

func TestGenerator_Interface(t *testing.T) {
	var _ Generator = &mockGenerator{}
	var _ Generator = &generator{}
}

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator("test-api-key", "gpt-4")
	if gen == nil {
		t.Error("expected non-nil generator")
	}
}

func TestBuildUserMessage(t *testing.T) {
	chunks := []store.SearchResult{
		{
			Chunk: chunker.Chunk{
				ID:           "1",
				Text:         "How to create a call using POST /calls endpoint",
				SourceFile:   "openapi.yaml",
				DocType:      chunker.DocTypeOpenAPI,
				SectionTitle: "POST /calls",
			},
			RelevanceScore: 0.95,
		},
		{
			Chunk: chunker.Chunk{
				ID:           "2",
				Text:         "Call creation requires authentication token",
				SourceFile:   "auth.md",
				DocType:      chunker.DocTypeDesign,
				SectionTitle: "Authentication",
			},
			RelevanceScore: 0.87,
		},
	}

	query := "How do I create a call?"
	result := buildUserMessage(query, chunks)

	// Verify the message contains context
	if !strings.Contains(result, "Context:") {
		t.Error("expected message to contain 'Context:'")
	}

	// Verify the message contains the question
	if !strings.Contains(result, "Question:") {
		t.Error("expected message to contain 'Question:'")
	}
	if !strings.Contains(result, query) {
		t.Error("expected message to contain the query")
	}

	// Verify source information is included
	if !strings.Contains(result, "openapi.yaml") {
		t.Error("expected message to contain source file")
	}
	if !strings.Contains(result, "POST /calls") {
		t.Error("expected message to contain section title")
	}

	// Verify chunk text is included
	if !strings.Contains(result, "How to create a call") {
		t.Error("expected message to contain chunk text")
	}
}

func TestBuildUserMessage_EmptyChunks(t *testing.T) {
	query := "test query"
	result := buildUserMessage(query, []store.SearchResult{})

	if !strings.Contains(result, "Context:") {
		t.Error("expected message to contain 'Context:'")
	}
	if !strings.Contains(result, query) {
		t.Error("expected message to contain the query")
	}
}

func TestBuildUserMessage_SingleChunk(t *testing.T) {
	chunks := []store.SearchResult{
		{
			Chunk: chunker.Chunk{
				ID:           "1",
				Text:         "Test content",
				SourceFile:   "test.md",
				DocType:      chunker.DocTypeDesign,
				SectionTitle: "Test Section",
			},
			RelevanceScore: 0.99,
		},
	}

	query := "test query"
	result := buildUserMessage(query, chunks)

	if !strings.Contains(result, "[1]") {
		t.Error("expected message to contain chunk numbering")
	}
	if !strings.Contains(result, "Test content") {
		t.Error("expected message to contain chunk content")
	}
}

func TestMockGenerator_Generate(t *testing.T) {
	m := &mockGenerator{}
	ctx := context.Background()

	result, err := m.Generate(ctx, "test query", []store.SearchResult{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "mock answer" {
		t.Errorf("expected 'mock answer', got %q", result)
	}
}

func TestMockGenerator_CustomFunction(t *testing.T) {
	expectedAnswer := "custom answer"
	m := &mockGenerator{
		generateFunc: func(ctx context.Context, query string, chunks []store.SearchResult) (string, error) {
			return expectedAnswer, nil
		},
	}

	ctx := context.Background()
	result, err := m.Generate(ctx, "test", []store.SearchResult{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != expectedAnswer {
		t.Errorf("expected %q, got %q", expectedAnswer, result)
	}
}

func TestSystemPrompt(t *testing.T) {
	if systemPrompt == "" {
		t.Error("system prompt should not be empty")
	}

	// Verify key elements are in the system prompt
	expectedElements := []string{
		"VoIPBin",
		"context",
		"information",
	}

	for _, elem := range expectedElements {
		if !strings.Contains(systemPrompt, elem) {
			t.Errorf("expected system prompt to contain %q", elem)
		}
	}
}

func TestBuildUserMessage_MultipleChunksFormatting(t *testing.T) {
	chunks := []store.SearchResult{
		{
			Chunk: chunker.Chunk{
				ID:           "1",
				Text:         "First chunk",
				SourceFile:   "file1.md",
				DocType:      chunker.DocTypeDevDoc,
				SectionTitle: "Section 1",
			},
			RelevanceScore: 0.95,
		},
		{
			Chunk: chunker.Chunk{
				ID:           "2",
				Text:         "Second chunk",
				SourceFile:   "file2.md",
				DocType:      chunker.DocTypeOpenAPI,
				SectionTitle: "Section 2",
			},
			RelevanceScore: 0.85,
		},
		{
			Chunk: chunker.Chunk{
				ID:           "3",
				Text:         "Third chunk",
				SourceFile:   "file3.md",
				DocType:      chunker.DocTypeGuideline,
				SectionTitle: "Section 3",
			},
			RelevanceScore: 0.75,
		},
	}

	result := buildUserMessage("test", chunks)

	// Verify all chunks are numbered
	if !strings.Contains(result, "[1]") {
		t.Error("expected [1] in result")
	}
	if !strings.Contains(result, "[2]") {
		t.Error("expected [2] in result")
	}
	if !strings.Contains(result, "[3]") {
		t.Error("expected [3] in result")
	}

	// Verify all chunk texts are included
	if !strings.Contains(result, "First chunk") {
		t.Error("expected first chunk text")
	}
	if !strings.Contains(result, "Second chunk") {
		t.Error("expected second chunk text")
	}
	if !strings.Contains(result, "Third chunk") {
		t.Error("expected third chunk text")
	}
}
