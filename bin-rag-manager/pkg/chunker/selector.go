package chunker

import (
	"path/filepath"
	"strings"
)

var contentTypeToExt = map[string]string{
	"text/plain":       ".txt",
	"text/html":        ".html",
	"text/csv":         ".csv",
	"text/markdown":    ".md",
	"application/pdf":  ".pdf",
	"application/json": ".json",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
	"text/x-rst":           ".rst",
	"text/restructuredtext": ".rst",
	"application/x-yaml":   ".yaml",
	"text/yaml":            ".yaml",
}

func GetChunkerByExtension(ext string) Chunker {
	switch strings.ToLower(ext) {
	case ".rst":
		return NewRSTChunker()
	case ".md":
		return NewMarkdownChunker(DocTypeDevDoc)
	case ".yaml", ".yml":
		return NewOpenAPIChunker()
	case ".txt":
		return NewTextChunker()
	case ".pdf":
		return NewPDFChunker()
	case ".html", ".htm":
		return NewHTMLChunker()
	case ".csv":
		return NewCSVChunker()
	case ".docx":
		return NewDOCXChunker()
	case ".json":
		return NewJSONChunker()
	default:
		return NewTextChunker()
	}
}

func DetectExtensionFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	return strings.ToLower(ext)
}

func DetectExtensionFromContentType(contentType string) string {
	ct := contentType
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = strings.ToLower(ct)

	if ext, ok := contentTypeToExt[ct]; ok {
		return ext
	}
	return ""
}
