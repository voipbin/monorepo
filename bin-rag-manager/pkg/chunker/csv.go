package chunker

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type csvChunker struct{}

func NewCSVChunker() Chunker {
	return &csvChunker{}
}

func (c *csvChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("could not parse CSV %s: %w", filePath, err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	relPath := filepath.Base(filePath)
	header := records[0]
	headerLine := strings.Join(header, ",")
	maxChars := maxTokens * 4

	chunks := []Chunk{}
	var current strings.Builder
	current.WriteString(headerLine)
	current.WriteString("\n")
	chunkIdx := 0

	for _, row := range records[1:] {
		rowLine := strings.Join(row, ",")
		if current.Len()+len(rowLine)+1 > maxChars && current.Len() > len(headerLine)+1 {
			title := fmt.Sprintf("Rows (part %d)", chunkIdx+1)
			id := generateChunkID(relPath, title)
			chunks = append(chunks, Chunk{
				ID:           id,
				Text:         current.String(),
				SourceFile:   filePath,
				DocType:      DocTypeDevDoc,
				SectionTitle: title,
			})
			chunkIdx++
			current.Reset()
			current.WriteString(headerLine)
			current.WriteString("\n")
		}
		current.WriteString(rowLine)
		current.WriteString("\n")
	}

	if current.Len() > len(headerLine)+1 {
		title := fmt.Sprintf("Rows (part %d)", chunkIdx+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         current.String(),
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}
