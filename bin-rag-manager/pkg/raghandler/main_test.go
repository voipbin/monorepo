package raghandler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"monorepo/bin-rag-manager/pkg/chunker"
	"monorepo/bin-rag-manager/pkg/store"
)

// mockRetriever for testing
type mockRetriever struct {
	queryFunc func(ctx context.Context, query string, topK int, docTypes []chunker.DocType) ([]store.SearchResult, error)
}

func (m *mockRetriever) Query(ctx context.Context, query string, topK int, docTypes []chunker.DocType) ([]store.SearchResult, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, topK, docTypes)
	}
	return []store.SearchResult{
		{
			Chunk: chunker.Chunk{
				ID:           "1",
				Text:         "test result",
				SourceFile:   "test.md",
				DocType:      chunker.DocTypeDesign,
				SectionTitle: "Test Section",
			},
			RelevanceScore: 0.95,
		},
	}, nil
}

// mockGenerator for testing
type mockGenerator struct {
	generateFunc func(ctx context.Context, query string, chunks []store.SearchResult) (string, error)
}

func (m *mockGenerator) Generate(ctx context.Context, query string, chunks []store.SearchResult) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, query, chunks)
	}
	return "mock answer", nil
}

// mockEmbedder for testing
type mockEmbedder struct {
	embedTextsFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *mockEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedTextsFunc != nil {
		return m.embedTextsFunc(ctx, texts)
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{1.0, 0.0, 0.0}
	}
	return result, nil
}

func (m *mockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	return []float32{1.0, 0.0, 0.0}, nil
}

func TestRagHandler_Interface(t *testing.T) {
	var _ RagHandler = &ragHandler{}
}

func TestNewRagHandler(t *testing.T) {
	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, "/tmp/docs", "/tmp/store.gob", 5)
	if h == nil {
		t.Error("expected non-nil handler")
	}
}

func TestRagHandler_Query_Success(t *testing.T) {
	ret := &mockRetriever{
		queryFunc: func(ctx context.Context, query string, topK int, docTypes []chunker.DocType) ([]store.SearchResult, error) {
			return []store.SearchResult{
				{
					Chunk: chunker.Chunk{
						ID:           "1",
						Text:         "test chunk",
						SourceFile:   "test.md",
						DocType:      chunker.DocTypeDesign,
						SectionTitle: "Section",
					},
					RelevanceScore: 0.9,
				},
			}, nil
		},
	}
	gen := &mockGenerator{
		generateFunc: func(ctx context.Context, query string, chunks []store.SearchResult) (string, error) {
			return "Generated answer", nil
		},
	}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, "/tmp/docs", "", 5)
	ctx := context.Background()

	req := &QueryRequest{
		Query: "test question",
		TopK:  3,
	}

	resp, err := h.Query(ctx, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Answer != "Generated answer" {
		t.Errorf("expected 'Generated answer', got %q", resp.Answer)
	}
	if len(resp.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(resp.Sources))
	}
}

func TestRagHandler_Query_DefaultTopK(t *testing.T) {
	topKReceived := 0
	ret := &mockRetriever{
		queryFunc: func(ctx context.Context, query string, topK int, docTypes []chunker.DocType) ([]store.SearchResult, error) {
			topKReceived = topK
			return []store.SearchResult{}, nil
		},
	}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	defaultTopK := 7
	h := NewRagHandler(ret, gen, emb, st, "/tmp/docs", "", defaultTopK)
	ctx := context.Background()

	req := &QueryRequest{
		Query: "test",
		TopK:  0, // Should use default
	}

	_, err := h.Query(ctx, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if topKReceived != defaultTopK {
		t.Errorf("expected topK=%d, got %d", defaultTopK, topKReceived)
	}
}

func TestRagHandler_Query_RetrievalError(t *testing.T) {
	expectedErr := errors.New("retrieval failed")
	ret := &mockRetriever{
		queryFunc: func(ctx context.Context, query string, topK int, docTypes []chunker.DocType) ([]store.SearchResult, error) {
			return nil, expectedErr
		},
	}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, "/tmp/docs", "", 5)
	ctx := context.Background()

	req := &QueryRequest{Query: "test"}

	_, err := h.Query(ctx, req)
	if err == nil {
		t.Error("expected error from retrieval")
	}
}

func TestRagHandler_Query_GenerationError(t *testing.T) {
	ret := &mockRetriever{}
	expectedErr := errors.New("generation failed")
	gen := &mockGenerator{
		generateFunc: func(ctx context.Context, query string, chunks []store.SearchResult) (string, error) {
			return "", expectedErr
		},
	}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, "/tmp/docs", "", 5)
	ctx := context.Background()

	req := &QueryRequest{Query: "test"}

	_, err := h.Query(ctx, req)
	if err == nil {
		t.Error("expected error from generation")
	}
}

func TestRagHandler_IndexStatus(t *testing.T) {
	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, "/tmp/docs", "", 5)
	ctx := context.Background()

	status, err := h.IndexStatus(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.ChunkCount != 0 {
		t.Errorf("expected 0 chunks, got %d", status.ChunkCount)
	}
}

func TestRagHandler_IndexIncremental(t *testing.T) {
	// Create temporary test files
	tmpDir, err := os.MkdirTemp("", "rag_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	mdFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Test\n\nContent here"), 0644); err != nil {
		t.Fatal(err)
	}

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	ctx := context.Background()

	err = h.IndexIncremental(ctx, []string{mdFile})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify indexing happened
	status, _ := h.IndexStatus(ctx)
	if status.ChunkCount == 0 {
		t.Error("expected chunks to be indexed")
	}
}

func TestCollectFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "collect_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(tmpDir, "file2.md")
	file3 := filepath.Join(tmpDir, "other.txt")

	for _, f := range []string{file1, file2, file3} {
		if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := collectFiles(tmpDir, ".md")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 .md files, got %d", len(files))
	}
}

func TestCollectFiles_NonExistentDir(t *testing.T) {
	_, err := collectFiles("/nonexistent/directory", ".md")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestDetectDocType(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected chunker.DocType
	}{
		{
			name:     "devdoc rst file",
			path:     "/path/to/docsdev/source/intro.rst",
			expected: chunker.DocTypeDevDoc,
		},
		{
			name:     "openapi yaml file",
			path:     "/path/to/openapi/spec.yaml",
			expected: chunker.DocTypeOpenAPI,
		},
		{
			name:     "openapi yml file",
			path:     "/path/to/openapi/spec.yml",
			expected: chunker.DocTypeOpenAPI,
		},
		{
			name:     "CLAUDE.md guideline",
			path:     "/path/to/bin-service/CLAUDE.md",
			expected: chunker.DocTypeGuideline,
		},
		{
			name:     "design doc",
			path:     "/path/to/docs/plans/design.md",
			expected: chunker.DocTypeDesign,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectDocType(tt.path)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCollectCLAUDEMDs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claude_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create bin- directories with CLAUDE.md files
	binDir1 := filepath.Join(tmpDir, "bin-service1")
	binDir2 := filepath.Join(tmpDir, "bin-service2")
	regularDir := filepath.Join(tmpDir, "not-bin-dir")

	for _, dir := range []string{binDir1, binDir2, regularDir} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create CLAUDE.md files
	claude1 := filepath.Join(binDir1, "CLAUDE.md")
	claude2 := filepath.Join(binDir2, "CLAUDE.md")
	claudeRegular := filepath.Join(regularDir, "CLAUDE.md")
	rootClaude := filepath.Join(tmpDir, "CLAUDE.md")

	for _, f := range []string{claude1, claude2, claudeRegular, rootClaude} {
		if err := os.WriteFile(f, []byte("# CLAUDE.md"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := collectCLAUDEMDs(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should collect bin-* CLAUDE.md files + root CLAUDE.md
	// Should NOT collect CLAUDE.md from non-bin directories
	if len(files) != 3 {
		t.Errorf("expected 3 CLAUDE.md files, got %d", len(files))
	}
}

func TestRagHandler_IndexIncremental_EmbedError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	mdFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Test\n\nSome content to chunk"), 0644); err != nil {
		t.Fatal(err)
	}

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{
		embedTextsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			if len(texts) > 0 {
				return nil, errors.New("embed error")
			}
			return [][]float32{}, nil
		},
	}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	ctx := context.Background()

	err = h.IndexIncremental(ctx, []string{mdFile})
	if err == nil {
		t.Error("expected error from embedding")
	}
}

func TestRagHandler_IndexStatus_AfterIndexing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	mdFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Test\n\nSome content"), 0644); err != nil {
		t.Fatal(err)
	}

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	ctx := context.Background()

	// Index the file
	if err := h.IndexIncremental(ctx, []string{mdFile}); err != nil {
		t.Fatal(err)
	}

	// Check status
	status, err := h.IndexStatus(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if status.ChunkCount == 0 {
		t.Error("expected chunks after indexing")
	}
	if status.LastRun.IsZero() {
		t.Error("expected LastRun to be set")
	}
}

func TestRagHandler_Query_WithDocTypeFilter(t *testing.T) {
	docTypesReceived := []chunker.DocType{}
	ret := &mockRetriever{
		queryFunc: func(ctx context.Context, query string, topK int, docTypes []chunker.DocType) ([]store.SearchResult, error) {
			docTypesReceived = docTypes
			return []store.SearchResult{}, nil
		},
	}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, "/tmp/docs", "", 5)
	ctx := context.Background()

	req := &QueryRequest{
		Query:    "test",
		DocTypes: []chunker.DocType{chunker.DocTypeOpenAPI, chunker.DocTypeDesign},
	}

	_, err := h.Query(ctx, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(docTypesReceived) != 2 {
		t.Errorf("expected 2 doc types to be passed, got %d", len(docTypesReceived))
	}
}

func TestRagHandler_IndexIncremental_NoChunks(t *testing.T) {
	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, "/tmp/docs", "", 5)
	ctx := context.Background()

	// Index with empty file list
	err := h.IndexIncremental(ctx, []string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// LastRun should still be updated
	impl := h.(*ragHandler)
	impl.mu.Lock()
	lastRun := impl.lastRun
	impl.mu.Unlock()

	if lastRun.IsZero() {
		t.Error("expected LastRun to be set even with no chunks")
	}
}

func TestRagHandler_IndexIncremental_WithSaveError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	mdFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Test\n\nContent here"), 0644); err != nil {
		t.Fatal(err)
	}

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	// Use invalid store path to trigger save error
	invalidPath := filepath.Join(tmpDir, "nonexistent", "store.gob")

	h := NewRagHandler(ret, gen, emb, st, tmpDir, invalidPath, 5)
	ctx := context.Background()

	// Should not return error but should log it
	err = h.IndexIncremental(ctx, []string{mdFile})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check that error was recorded
	status, _ := h.IndexStatus(ctx)
	if len(status.Errors) == 0 {
		t.Error("expected save error to be recorded")
	}
}

func TestRagHandler_IndexFiles_ChunkError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	ctx := context.Background()

	// Use non-existent file to trigger chunk error
	files := []indexFile{
		{path: "/nonexistent/file.md", docType: chunker.DocTypeDesign},
	}

	impl := h.(*ragHandler)
	err = impl.indexFiles(ctx, files)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check that error was recorded
	impl.mu.Lock()
	hasError := len(impl.indexErrors) > 0
	impl.mu.Unlock()

	if !hasError {
		t.Error("expected chunk error to be recorded")
	}
}

func TestRagHandler_LastRunTimestamp(t *testing.T) {
	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, "/tmp/docs", "", 5)
	ctx := context.Background()

	before := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Trigger indexing with empty file list
	if err := h.IndexIncremental(ctx, []string{}); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	after := time.Now()

	status, _ := h.IndexStatus(ctx)
	if status.LastRun.Before(before) || status.LastRun.After(after) {
		t.Error("LastRun timestamp not within expected range")
	}
}

func TestRagHandler_IndexFull(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "rag_full_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create bin-api-manager/docsdev/source directory
	rstDir := filepath.Join(tmpDir, "bin-api-manager", "docsdev", "source")
	if err := os.MkdirAll(rstDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create RST file
	rstFile := filepath.Join(rstDir, "test.rst")
	if err := os.WriteFile(rstFile, []byte("Test\n====\n\nContent"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create bin-openapi-manager/openapi directory
	openapiDir := filepath.Join(tmpDir, "bin-openapi-manager", "openapi")
	if err := os.MkdirAll(openapiDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create OpenAPI YAML file
	yamlFile := filepath.Join(openapiDir, "openapi.yaml")
	yamlContent := `openapi: 3.0.0
paths:
  /test:
    get:
      summary: Test endpoint
`
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create docs/plans directory
	designDir := filepath.Join(tmpDir, "docs", "plans")
	if err := os.MkdirAll(designDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create design doc
	mdFile := filepath.Join(designDir, "design.md")
	if err := os.WriteFile(mdFile, []byte("# Design\n\nContent"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create bin-service directory with CLAUDE.md
	binService := filepath.Join(tmpDir, "bin-test-service")
	if err := os.Mkdir(binService, 0755); err != nil {
		t.Fatal(err)
	}
	claudeFile := filepath.Join(binService, "CLAUDE.md")
	if err := os.WriteFile(claudeFile, []byte("# CLAUDE.md\n\nGuideline"), 0644); err != nil {
		t.Fatal(err)
	}

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	ctx := context.Background()

	err = h.IndexFull(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify chunks were indexed
	status, _ := h.IndexStatus(ctx)
	if status.ChunkCount == 0 {
		t.Error("expected chunks to be indexed")
	}
}

func TestRagHandler_IndexFull_NonExistentDirs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_full_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	// Use tmpDir that doesn't have expected subdirectories
	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	ctx := context.Background()

	// Should not error even if no directories exist
	err = h.IndexFull(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have no chunks
	status, _ := h.IndexStatus(ctx)
	if status.ChunkCount != 0 {
		t.Errorf("expected 0 chunks, got %d", status.ChunkCount)
	}
}

func TestRagHandler_IndexFull_WithYmlFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_yml_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create bin-openapi-manager/openapi directory
	openapiDir := filepath.Join(tmpDir, "bin-openapi-manager", "openapi")
	if err := os.MkdirAll(openapiDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .yml file (not .yaml)
	ymlFile := filepath.Join(openapiDir, "spec.yml")
	ymlContent := `openapi: 3.0.0
paths:
  /api:
    get:
      summary: API endpoint
`
	if err := os.WriteFile(ymlFile, []byte(ymlContent), 0644); err != nil {
		t.Fatal(err)
	}

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	ctx := context.Background()

	err = h.IndexFull(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	status, _ := h.IndexStatus(ctx)
	if status.ChunkCount == 0 {
		t.Error("expected .yml file to be indexed")
	}
}

func TestRagHandler_IndexFull_WithRootCLAUDEMD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_claude_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create root CLAUDE.md
	rootClaude := filepath.Join(tmpDir, "CLAUDE.md")
	if err := os.WriteFile(rootClaude, []byte("# Root Guidelines\n\nContent"), 0644); err != nil {
		t.Fatal(err)
	}

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	ctx := context.Background()

	err = h.IndexFull(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	status, _ := h.IndexStatus(ctx)
	if status.ChunkCount == 0 {
		t.Error("expected root CLAUDE.md to be indexed")
	}
}

func TestCollectFiles_WithSubdirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "collect_sub_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create subdirectories
	subdir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files at different levels
	file1 := filepath.Join(tmpDir, "file1.md")
	file2 := filepath.Join(subdir, "file2.md")

	for _, f := range []string{file1, file2} {
		if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := collectFiles(tmpDir, ".md")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files (including subdirectories), got %d", len(files))
	}
}

func TestRagHandler_IndexFiles_DifferentChunkerTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_chunker_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create RST file
	rstFile := filepath.Join(tmpDir, "test.rst")
	if err := os.WriteFile(rstFile, []byte("Title\n=====\n\nContent"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create YAML file
	yamlFile := filepath.Join(tmpDir, "spec.yaml")
	yamlContent := `openapi: 3.0.0
paths:
  /test:
    get:
      summary: Test
`
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create guideline MD file
	guideFile := filepath.Join(tmpDir, "guide.md")
	if err := os.WriteFile(guideFile, []byte("# Guide\n\nContent"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create design MD file
	designFile := filepath.Join(tmpDir, "design.md")
	if err := os.WriteFile(designFile, []byte("# Design\n\nContent"), 0644); err != nil {
		t.Fatal(err)
	}

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	impl := h.(*ragHandler)
	ctx := context.Background()

	files := []indexFile{
		{path: rstFile, docType: chunker.DocTypeDevDoc},
		{path: yamlFile, docType: chunker.DocTypeOpenAPI},
		{path: guideFile, docType: chunker.DocTypeGuideline},
		{path: designFile, docType: chunker.DocTypeDesign},
	}

	err = impl.indexFiles(ctx, files)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	stats := st.Stats()
	if stats.ChunkCount == 0 {
		t.Error("expected chunks to be indexed")
	}
}

func TestRagHandler_IndexFull_ErrorClearance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_error_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	impl := h.(*ragHandler)

	// Set some errors
	impl.mu.Lock()
	impl.indexErrors = []string{"previous error"}
	impl.mu.Unlock()

	ctx := context.Background()
	err = h.IndexFull(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Errors should be cleared
	status, _ := h.IndexStatus(ctx)
	if len(status.Errors) != 0 {
		t.Error("expected errors to be cleared on new indexing")
	}
}

func TestRagHandler_IndexIncremental_ErrorClearance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rag_error_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ret := &mockRetriever{}
	gen := &mockGenerator{}
	emb := &mockEmbedder{}
	st := store.NewMemoryStore()

	h := NewRagHandler(ret, gen, emb, st, tmpDir, "", 5)
	impl := h.(*ragHandler)

	// Set some errors
	impl.mu.Lock()
	impl.indexErrors = []string{"previous error"}
	impl.mu.Unlock()

	ctx := context.Background()
	err = h.IndexIncremental(ctx, []string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Errors should be cleared
	status, _ := h.IndexStatus(ctx)
	if len(status.Errors) != 0 {
		t.Error("expected errors to be cleared on new indexing")
	}
}

func TestCollectCLAUDEMDs_ErrorHandling(t *testing.T) {
	// Test with non-existent directory
	_, err := collectCLAUDEMDs("/nonexistent/directory")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestCollectCLAUDEMDs_NoBinDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claude_nobin_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create some non-bin directories
	for _, dir := range []string{"src", "docs", "test"} {
		if err := os.Mkdir(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	files, err := collectCLAUDEMDs(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files without bin- directories, got %d", len(files))
	}
}
