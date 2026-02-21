package chunker

// DocType represents the type of document source
type DocType string

const (
	DocTypeDevDoc    DocType = "devdoc"
	DocTypeOpenAPI   DocType = "openapi"
	DocTypeDesign    DocType = "design"
	DocTypeGuideline DocType = "guideline"
)

// Chunk represents a single chunk of text with metadata
type Chunk struct {
	ID           string  `json:"id"`
	Text         string  `json:"text"`
	SourceFile   string  `json:"source_file"`
	DocType      DocType `json:"doc_type"`
	SectionTitle string  `json:"section_title"`
}

// Chunker defines the interface for document chunkers
type Chunker interface {
	Chunk(filePath string, maxTokens int) ([]Chunk, error)
}
