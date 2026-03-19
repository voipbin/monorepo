package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

type htmlChunker struct{}

func NewHTMLChunker() Chunker {
	return &htmlChunker{}
}

func (c *htmlChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	doc, err := html.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("could not parse HTML %s: %w", filePath, err)
	}

	text := extractHTMLText(doc)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	relPath := filepath.Base(filePath)
	parts := splitByTokenLimit(text, maxTokens)
	chunks := make([]Chunk, 0, len(parts))
	for i, part := range parts {
		title := fmt.Sprintf("Part %d", i+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         part,
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}

func extractHTMLText(n *html.Node) string {
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
		return ""
	}

	if n.Type == html.TextNode {
		return n.Data
	}

	var sb strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		sb.WriteString(extractHTMLText(child))
		if child.Type == html.ElementNode {
			switch child.Data {
			case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "br", "li", "tr":
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}
