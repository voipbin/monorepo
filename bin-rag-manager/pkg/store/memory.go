package store

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"monorepo/bin-rag-manager/pkg/chunker"
)

// StoredChunk is a chunk with its embedding vector
type StoredChunk struct {
	Chunk     chunker.Chunk
	Embedding []float32
}

// SearchResult represents a search result with relevance score
type SearchResult struct {
	Chunk          chunker.Chunk `json:"chunk"`
	RelevanceScore float64       `json:"relevance_score"`
}

// StoreStats contains statistics about the vector store
type StoreStats struct {
	ChunkCount  int       `json:"chunk_count"`
	LastUpdated time.Time `json:"last_updated"`
}

// Store defines the interface for the vector store
type Store interface {
	Add(chunks []chunker.Chunk, embeddings [][]float32)
	Search(queryEmbedding []float32, topK int, docTypes []chunker.DocType) []SearchResult
	DeleteByFile(sourceFile string)
	Save(filePath string) error
	Load(filePath string) error
	Stats() StoreStats
}

// memoryStore is an in-memory vector store
type memoryStore struct {
	mu          sync.RWMutex
	chunks      []StoredChunk
	lastUpdated time.Time
}

// NewMemoryStore creates a new in-memory vector store
func NewMemoryStore() Store {
	return &memoryStore{}
}

// Add adds chunks with their embeddings to the store
func (s *memoryStore) Add(chunks []chunker.Chunk, embeddings [][]float32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, chunk := range chunks {
		if i < len(embeddings) {
			s.chunks = append(s.chunks, StoredChunk{
				Chunk:     chunk,
				Embedding: embeddings[i],
			})
		}
	}
	s.lastUpdated = time.Now()
}

// Search performs cosine similarity search and returns top-k results
func (s *memoryStore) Search(queryEmbedding []float32, topK int, docTypes []chunker.DocType) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	docTypeFilter := make(map[chunker.DocType]bool)
	for _, dt := range docTypes {
		docTypeFilter[dt] = true
	}

	var results []SearchResult
	for _, stored := range s.chunks {
		// Filter by doc type if specified
		if len(docTypeFilter) > 0 && !docTypeFilter[stored.Chunk.DocType] {
			continue
		}

		score := cosineSimilarity(queryEmbedding, stored.Embedding)
		results = append(results, SearchResult{
			Chunk:          stored.Chunk,
			RelevanceScore: score,
		})
	}

	// Sort by relevance score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results
}

// DeleteByFile removes all chunks from a specific source file
func (s *memoryStore) DeleteByFile(sourceFile string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var remaining []StoredChunk
	for _, stored := range s.chunks {
		if stored.Chunk.SourceFile != sourceFile {
			remaining = append(remaining, stored)
		}
	}
	s.chunks = remaining
	s.lastUpdated = time.Now()
}

// Save serializes the store to a file using gob encoding
func (s *memoryStore) Save(filePath string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", filePath, err)
	}

	enc := gob.NewEncoder(f)
	if err := enc.Encode(s.chunks); err != nil {
		_ = f.Close()
		return fmt.Errorf("could not encode chunks: %w", err)
	}

	return f.Close()
}

// Load deserializes the store from a file
func (s *memoryStore) Load(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// First run, no data to load
			return nil
		}
		return fmt.Errorf("could not open file %s: %w", filePath, err)
	}

	dec := gob.NewDecoder(f)
	if err := dec.Decode(&s.chunks); err != nil {
		_ = f.Close()
		return fmt.Errorf("could not decode chunks: %w", err)
	}

	s.lastUpdated = time.Now()
	return f.Close()
}

// Stats returns statistics about the store
func (s *memoryStore) Stats() StoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return StoreStats{
		ChunkCount:  len(s.chunks),
		LastUpdated: s.lastUpdated,
	}
}

// cosineSimilarity computes the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	denominator := math.Sqrt(normA) * math.Sqrt(normB)
	if denominator == 0 {
		return 0
	}

	return dotProduct / denominator
}
