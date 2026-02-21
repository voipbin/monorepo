package chunker

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// rstChunker implements Chunker for RST files
type rstChunker struct{}

// NewRSTChunker creates a new RST file chunker
func NewRSTChunker() Chunker {
	return &rstChunker{}
}

// Chunk parses an RST file and splits it into chunks by section headers
func (c *rstChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
	}

	sections := splitRSTSections(string(content))
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
				DocType:      DocTypeDevDoc,
				SectionTitle: title,
			})
		}
	}

	return chunks, nil
}

type rstSection struct {
	title string
	text  string
}

// splitRSTSections splits RST content by section headers.
// RST headers are underlined with characters like =, -, ~, ^, etc.
func splitRSTSections(content string) []rstSection {
	lines := strings.Split(content, "\n")
	underlineChars := "=-~^`"

	var sections []rstSection
	currentTitle := "Introduction"
	var currentText []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check if the next line is an underline (indicating current line is a title)
		if i+1 < len(lines) && isRSTUnderline(lines[i+1], underlineChars) && len(strings.TrimSpace(line)) > 0 {
			// Save previous section
			if len(currentText) > 0 {
				sections = append(sections, rstSection{
					title: currentTitle,
					text:  strings.Join(currentText, "\n"),
				})
			}
			currentTitle = strings.TrimSpace(line)
			currentText = nil
			i++ // skip the underline
			continue
		}

		// Check if current line is an overline (title on next line, underline after)
		if isRSTUnderline(line, underlineChars) && i+2 < len(lines) && isRSTUnderline(lines[i+2], underlineChars) {
			if len(currentText) > 0 {
				sections = append(sections, rstSection{
					title: currentTitle,
					text:  strings.Join(currentText, "\n"),
				})
			}
			currentTitle = strings.TrimSpace(lines[i+1])
			currentText = nil
			i += 2 // skip title and underline
			continue
		}

		currentText = append(currentText, line)
	}

	// Save last section
	if len(currentText) > 0 {
		sections = append(sections, rstSection{
			title: currentTitle,
			text:  strings.Join(currentText, "\n"),
		})
	}

	return sections
}

func isRSTUnderline(line string, underlineChars string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	firstChar := trimmed[0]
	if !strings.ContainsRune(underlineChars, rune(firstChar)) {
		return false
	}
	for _, ch := range trimmed {
		if byte(ch) != firstChar {
			return false
		}
	}
	return true
}

// splitByTokenLimit splits text into chunks that don't exceed maxTokens.
// Uses a rough estimate of 4 characters per token.
func splitByTokenLimit(text string, maxTokens int) []string {
	maxChars := maxTokens * 4
	if len(text) <= maxChars {
		return []string{text}
	}

	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current strings.Builder

	for _, para := range paragraphs {
		if current.Len()+len(para)+2 > maxChars && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

func generateChunkID(sourceFile, sectionTitle string) string {
	h := sha256.New()
	h.Write([]byte(sourceFile + "|" + sectionTitle))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
