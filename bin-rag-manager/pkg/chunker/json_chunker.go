package chunker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type jsonChunker struct{}

func NewJSONChunker() Chunker {
	return &jsonChunker{}
}

func (c *jsonChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
	}

	relPath := filepath.Base(filePath)

	var obj map[string]any
	if err := json.Unmarshal(content, &obj); err == nil {
		return c.chunkObject(relPath, filePath, obj, maxTokens)
	}

	var arr []any
	if err := json.Unmarshal(content, &arr); err == nil {
		return c.chunkArray(relPath, filePath, arr, maxTokens)
	}

	parts := splitByTokenLimit(string(content), maxTokens)
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

func (c *jsonChunker) chunkObject(relPath, filePath string, obj map[string]any, maxTokens int) ([]Chunk, error) {
	chunks := []Chunk{}
	maxChars := maxTokens * 4

	for key, val := range obj {
		b, err := json.MarshalIndent(map[string]any{key: val}, "", "  ")
		if err != nil {
			continue
		}

		text := string(b)
		if len(text) <= maxChars {
			title := fmt.Sprintf("Key: %s", key)
			id := generateChunkID(relPath, title)
			chunks = append(chunks, Chunk{
				ID:           id,
				Text:         text,
				SourceFile:   filePath,
				DocType:      DocTypeDevDoc,
				SectionTitle: title,
			})
		} else {
			parts := splitByTokenLimit(text, maxTokens)
			for i, part := range parts {
				title := fmt.Sprintf("Key: %s (part %d)", key, i+1)
				id := generateChunkID(relPath, title)
				chunks = append(chunks, Chunk{
					ID:           id,
					Text:         part,
					SourceFile:   filePath,
					DocType:      DocTypeDevDoc,
					SectionTitle: title,
				})
			}
		}
	}

	return chunks, nil
}

func (c *jsonChunker) chunkArray(relPath, filePath string, arr []any, maxTokens int) ([]Chunk, error) {
	maxChars := maxTokens * 4
	chunks := []Chunk{}
	var current []any
	currentSize := 0
	chunkIdx := 0

	for _, item := range arr {
		b, err := json.Marshal(item)
		if err != nil {
			continue
		}

		if currentSize+len(b) > maxChars && len(current) > 0 {
			text, _ := json.MarshalIndent(current, "", "  ")
			title := fmt.Sprintf("Items (part %d)", chunkIdx+1)
			id := generateChunkID(relPath, title)
			chunks = append(chunks, Chunk{
				ID:           id,
				Text:         string(text),
				SourceFile:   filePath,
				DocType:      DocTypeDevDoc,
				SectionTitle: title,
			})
			chunkIdx++
			current = nil
			currentSize = 0
		}
		current = append(current, item)
		currentSize += len(b)
	}

	if len(current) > 0 {
		text, _ := json.MarshalIndent(current, "", "  ")
		title := fmt.Sprintf("Items (part %d)", chunkIdx+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         string(text),
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}
