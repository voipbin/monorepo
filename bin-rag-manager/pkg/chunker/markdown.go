package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// markdownChunker implements Chunker for Markdown files
type markdownChunker struct {
	docType DocType
}

// NewMarkdownChunker creates a new Markdown file chunker
func NewMarkdownChunker(docType DocType) Chunker {
	return &markdownChunker{docType: docType}
}

// Chunk parses a Markdown file and splits it into chunks by headings
func (c *markdownChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
	}

	sections := splitMarkdownSections(string(content))
	relPath := filepath.Base(filePath)

	var chunks []Chunk
	for _, section := range sections {
		if strings.TrimSpace(section.text) == "" {
			continue
		}

		sectionChunks := splitByTokenLimit(section.text, maxTokens)
		for i, text := range sectionChunks {
			title := section.title
			if len(sectionChunks) > 1 {
				title = fmt.Sprintf("%s (part %d)", section.title, i+1)
			}

			id := generateChunkID(relPath, title)
			chunks = append(chunks, Chunk{
				ID:           id,
				Text:         text,
				SourceFile:   filePath,
				DocType:      c.docType,
				SectionTitle: title,
			})
		}
	}

	return chunks, nil
}

type mdSection struct {
	title string
	text  string
}

// splitMarkdownSections splits Markdown content by ## headings
func splitMarkdownSections(content string) []mdSection {
	lines := strings.Split(content, "\n")

	var sections []mdSection
	currentTitle := "Introduction"
	var currentText []string

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# ") {
			// Save previous section
			if len(currentText) > 0 {
				sections = append(sections, mdSection{
					title: currentTitle,
					text:  strings.Join(currentText, "\n"),
				})
			}
			currentTitle = strings.TrimSpace(strings.TrimLeft(line, "#"))
			currentText = nil
			continue
		}

		currentText = append(currentText, line)
	}

	// Save last section
	if len(currentText) > 0 {
		sections = append(sections, mdSection{
			title: currentTitle,
			text:  strings.Join(currentText, "\n"),
		})
	}

	return sections
}
