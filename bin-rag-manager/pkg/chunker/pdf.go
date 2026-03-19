package chunker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

type pdfChunker struct{}

func NewPDFChunker() Chunker {
	return &pdfChunker{}
}

func (c *pdfChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open PDF %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	var sb strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}

	textStr := sb.String()
	if strings.TrimSpace(textStr) == "" {
		return nil, nil
	}

	relPath := filepath.Base(filePath)
	parts := splitByTokenLimit(textStr, maxTokens)
	chunks := make([]Chunk, 0, len(parts))
	for i, text := range parts {
		title := fmt.Sprintf("Page group %d", i+1)
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
