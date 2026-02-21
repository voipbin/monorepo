package raghandler

//go:generate mockgen -package raghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"monorepo/bin-rag-manager/pkg/chunker"
	"monorepo/bin-rag-manager/pkg/embedder"
	"monorepo/bin-rag-manager/pkg/generator"
	"monorepo/bin-rag-manager/pkg/retriever"
	"monorepo/bin-rag-manager/pkg/store"

	"github.com/sirupsen/logrus"
)

// QueryRequest represents a RAG query request
type QueryRequest struct {
	Query    string            `json:"query"`
	DocTypes []chunker.DocType `json:"doc_types,omitempty"`
	TopK     int               `json:"top_k,omitempty"`
}

// Source represents a source reference in the query response
type Source struct {
	SourceFile     string          `json:"source_file"`
	SectionTitle   string          `json:"section_title"`
	DocType        chunker.DocType `json:"doc_type"`
	RelevanceScore float64         `json:"relevance_score"`
}

// QueryResponse represents a RAG query response
type QueryResponse struct {
	Answer  string   `json:"answer"`
	Sources []Source `json:"sources"`
}

// IndexStatusResponse represents the indexing status
type IndexStatusResponse struct {
	LastRun    time.Time `json:"last_run"`
	ChunkCount int       `json:"chunk_count"`
	Errors     []string  `json:"errors,omitempty"`
}

// RagHandler defines the interface for RAG operations
type RagHandler interface {
	Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error)
	IndexFull(ctx context.Context) error
	IndexIncremental(ctx context.Context, files []string) error
	IndexStatus(ctx context.Context) (*IndexStatusResponse, error)
}

type ragHandler struct {
	retriever  retriever.Retriever
	generator  generator.Generator
	embedder   embedder.Embedder
	store      store.Store
	docsPath   string
	storePath  string
	defaultTopK int

	mu          sync.Mutex
	lastRun     time.Time
	indexErrors []string
}

// NewRagHandler creates a new RagHandler
func NewRagHandler(
	ret retriever.Retriever,
	gen generator.Generator,
	emb embedder.Embedder,
	st store.Store,
	docsPath string,
	storePath string,
	defaultTopK int,
) RagHandler {
	return &ragHandler{
		retriever:   ret,
		generator:   gen,
		embedder:    emb,
		store:       st,
		docsPath:    docsPath,
		storePath:   storePath,
		defaultTopK: defaultTopK,
	}
}

// Query performs a RAG query: retrieve relevant chunks and generate an answer
func (h *ragHandler) Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	log := logrus.WithField("func", "Query")

	topK := req.TopK
	if topK <= 0 {
		topK = h.defaultTopK
	}

	results, err := h.retriever.Query(ctx, req.Query, topK, req.DocTypes)
	if err != nil {
		log.Errorf("Could not retrieve chunks: %v", err)
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}
	log.Debugf("Retrieved %d chunks for query", len(results))

	answer, err := h.generator.Generate(ctx, req.Query, results)
	if err != nil {
		log.Errorf("Could not generate answer: %v", err)
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	var sources []Source
	for _, r := range results {
		sources = append(sources, Source{
			SourceFile:     r.Chunk.SourceFile,
			SectionTitle:   r.Chunk.SectionTitle,
			DocType:        r.Chunk.DocType,
			RelevanceScore: r.RelevanceScore,
		})
	}

	return &QueryResponse{
		Answer:  answer,
		Sources: sources,
	}, nil
}

// IndexFull performs a full re-index of all document sources
func (h *ragHandler) IndexFull(ctx context.Context) error {
	log := logrus.WithField("func", "IndexFull")
	log.Infof("Starting full re-index from path: %s", h.docsPath)

	h.mu.Lock()
	h.indexErrors = nil
	h.mu.Unlock()

	var allFiles []indexFile

	// Collect RST files (dev docs)
	rstPath := filepath.Join(h.docsPath, "bin-api-manager", "docsdev", "source")
	if files, err := collectFiles(rstPath, ".rst"); err == nil {
		for _, f := range files {
			allFiles = append(allFiles, indexFile{path: f, docType: chunker.DocTypeDevDoc})
		}
	}

	// Collect OpenAPI files
	openapiPath := filepath.Join(h.docsPath, "bin-openapi-manager", "openapi")
	if files, err := collectFiles(openapiPath, ".yaml"); err == nil {
		for _, f := range files {
			allFiles = append(allFiles, indexFile{path: f, docType: chunker.DocTypeOpenAPI})
		}
	}
	if files, err := collectFiles(openapiPath, ".yml"); err == nil {
		for _, f := range files {
			allFiles = append(allFiles, indexFile{path: f, docType: chunker.DocTypeOpenAPI})
		}
	}

	// Collect design docs
	designPath := filepath.Join(h.docsPath, "docs", "plans")
	if files, err := collectFiles(designPath, ".md"); err == nil {
		for _, f := range files {
			allFiles = append(allFiles, indexFile{path: f, docType: chunker.DocTypeDesign})
		}
	}

	// Collect CLAUDE.md guideline files
	if files, err := collectCLAUDEMDs(h.docsPath); err == nil {
		for _, f := range files {
			allFiles = append(allFiles, indexFile{path: f, docType: chunker.DocTypeGuideline})
		}
	}

	log.Infof("Found %d files to index", len(allFiles))

	return h.indexFiles(ctx, allFiles)
}

// IndexIncremental re-indexes specific files
func (h *ragHandler) IndexIncremental(ctx context.Context, files []string) error {
	log := logrus.WithField("func", "IndexIncremental")
	log.Infof("Starting incremental re-index for %d files", len(files))

	h.mu.Lock()
	h.indexErrors = nil
	h.mu.Unlock()

	var allFiles []indexFile
	for _, f := range files {
		docType := detectDocType(f)
		allFiles = append(allFiles, indexFile{path: f, docType: docType})
	}

	return h.indexFiles(ctx, allFiles)
}

// IndexStatus returns the current indexing status
func (h *ragHandler) IndexStatus(ctx context.Context) (*IndexStatusResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	stats := h.store.Stats()
	return &IndexStatusResponse{
		LastRun:    h.lastRun,
		ChunkCount: stats.ChunkCount,
		Errors:     h.indexErrors,
	}, nil
}

type indexFile struct {
	path    string
	docType chunker.DocType
}

func (h *ragHandler) indexFiles(ctx context.Context, files []indexFile) error {
	log := logrus.WithField("func", "indexFiles")

	rstChunker := chunker.NewRSTChunker()
	mdDesignChunker := chunker.NewMarkdownChunker(chunker.DocTypeDesign)
	mdGuidelineChunker := chunker.NewMarkdownChunker(chunker.DocTypeGuideline)
	openapiChunker := chunker.NewOpenAPIChunker()

	var allChunks []chunker.Chunk

	for _, f := range files {
		// Delete existing chunks for this file
		h.store.DeleteByFile(f.path)

		var c chunker.Chunker
		switch {
		case strings.HasSuffix(f.path, ".rst"):
			c = rstChunker
		case strings.HasSuffix(f.path, ".yaml") || strings.HasSuffix(f.path, ".yml"):
			c = openapiChunker
		case f.docType == chunker.DocTypeGuideline:
			c = mdGuidelineChunker
		default:
			c = mdDesignChunker
		}

		chunks, err := c.Chunk(f.path, 800)
		if err != nil {
			log.Errorf("Could not chunk file %s: %v", f.path, err)
			h.mu.Lock()
			h.indexErrors = append(h.indexErrors, fmt.Sprintf("chunk error: %s: %v", f.path, err))
			h.mu.Unlock()
			continue
		}

		allChunks = append(allChunks, chunks...)
	}

	if len(allChunks) == 0 {
		log.Infof("No chunks to embed")
		h.mu.Lock()
		h.lastRun = time.Now()
		h.mu.Unlock()
		return nil
	}

	log.Infof("Embedding %d chunks", len(allChunks))

	// Extract texts for embedding
	texts := make([]string, len(allChunks))
	for i, chunk := range allChunks {
		texts[i] = chunk.Text
	}

	embeddings, err := h.embedder.EmbedTexts(ctx, texts)
	if err != nil {
		return fmt.Errorf("could not embed chunks: %w", err)
	}

	h.store.Add(allChunks, embeddings)

	// Persist to disk
	if h.storePath != "" {
		if err := h.store.Save(h.storePath); err != nil {
			log.Errorf("Could not save store: %v", err)
			h.mu.Lock()
			h.indexErrors = append(h.indexErrors, fmt.Sprintf("save error: %v", err))
			h.mu.Unlock()
		}
	}

	h.mu.Lock()
	h.lastRun = time.Now()
	h.mu.Unlock()

	stats := h.store.Stats()
	log.Infof("Indexing complete. Total chunks: %d", stats.ChunkCount)
	return nil
}

func collectFiles(dir string, ext string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ext) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func collectCLAUDEMDs(basePath string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "bin-") {
			claudeMD := filepath.Join(basePath, entry.Name(), "CLAUDE.md")
			if _, err := os.Stat(claudeMD); err == nil {
				files = append(files, claudeMD)
			}
		}
	}
	// Also check root CLAUDE.md
	rootClaude := filepath.Join(basePath, "CLAUDE.md")
	if _, err := os.Stat(rootClaude); err == nil {
		files = append(files, rootClaude)
	}
	return files, nil
}

func detectDocType(filePath string) chunker.DocType {
	if strings.Contains(filePath, "docsdev") && strings.HasSuffix(filePath, ".rst") {
		return chunker.DocTypeDevDoc
	}
	if strings.Contains(filePath, "openapi") && (strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml")) {
		return chunker.DocTypeOpenAPI
	}
	if strings.Contains(filePath, "CLAUDE.md") {
		return chunker.DocTypeGuideline
	}
	return chunker.DocTypeDesign
}
