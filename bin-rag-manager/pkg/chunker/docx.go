package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fumiama/go-docx"
)

type docxChunker struct{}

func NewDOCXChunker() Chunker {
	return &docxChunker{}
}

func (c *docxChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not stat file %s: %w", filePath, err)
	}

	doc, err := docx.Parse(f, fi.Size())
	if err != nil {
		return nil, fmt.Errorf("could not parse DOCX %s: %w", filePath, err)
	}

	var sb strings.Builder
	for _, item := range doc.Document.Body.Items {
		p, ok := item.(*docx.Paragraph)
		if !ok {
			continue
		}
		for _, child := range p.Children {
			r, ok := child.(*docx.Run)
			if !ok {
				continue
			}
			for _, rc := range r.Children {
				if t, ok := rc.(*docx.Text); ok {
					sb.WriteString(t.Text)
				}
			}
		}
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
