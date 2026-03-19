package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type textChunker struct{}

func NewTextChunker() Chunker {
	return &textChunker{}
}

func (c *textChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
	}

	textStr := string(content)
	if strings.TrimSpace(textStr) == "" {
		return nil, nil
	}

	relPath := filepath.Base(filePath)
	parts := splitByTokenLimit(textStr, maxTokens)
	chunks := make([]Chunk, 0, len(parts))
	for i, text := range parts {
		title := fmt.Sprintf("Part %d", i+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         text,
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}
