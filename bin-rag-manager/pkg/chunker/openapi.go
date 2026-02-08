package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// openapiChunker implements Chunker for OpenAPI YAML files
type openapiChunker struct{}

// NewOpenAPIChunker creates a new OpenAPI YAML file chunker
func NewOpenAPIChunker() Chunker {
	return &openapiChunker{}
}

// Chunk parses an OpenAPI YAML file and splits it by endpoints and schemas
func (c *openapiChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
	}

	var spec map[string]any
	if err := yaml.Unmarshal(content, &spec); err != nil {
		return nil, fmt.Errorf("could not parse OpenAPI spec %s: %w", filePath, err)
	}

	relPath := filepath.Base(filePath)
	var chunks []Chunk

	// Parse paths (endpoints)
	if paths, ok := spec["paths"].(map[string]any); ok {
		for path, methods := range paths {
			methodMap, ok := methods.(map[string]any)
			if !ok {
				continue
			}
			for method, details := range methodMap {
				title := fmt.Sprintf("%s %s", strings.ToUpper(method), path)
				text := formatEndpoint(path, method, details)

				id := generateChunkID(relPath, title)
				chunks = append(chunks, Chunk{
					ID:           id,
					Text:         text,
					SourceFile:   filePath,
					DocType:      DocTypeOpenAPI,
					SectionTitle: title,
				})
			}
		}
	}

	// Parse component schemas
	if components, ok := spec["components"].(map[string]any); ok {
		if schemas, ok := components["schemas"].(map[string]any); ok {
			for name, schema := range schemas {
				title := fmt.Sprintf("Schema: %s", name)
				text := formatSchema(name, schema)

				id := generateChunkID(relPath, title)
				chunks = append(chunks, Chunk{
					ID:           id,
					Text:         text,
					SourceFile:   filePath,
					DocType:      DocTypeOpenAPI,
					SectionTitle: title,
				})
			}
		}
	}

	return chunks, nil
}

func formatEndpoint(path, method string, details any) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Endpoint: %s %s\n", strings.ToUpper(method), path))

	detailMap, ok := details.(map[string]any)
	if !ok {
		return sb.String()
	}

	if summary, ok := detailMap["summary"].(string); ok {
		sb.WriteString(fmt.Sprintf("Summary: %s\n", summary))
	}
	if description, ok := detailMap["description"].(string); ok {
		sb.WriteString(fmt.Sprintf("Description: %s\n", description))
	}
	if tags, ok := detailMap["tags"].([]any); ok {
		var tagStrs []string
		for _, t := range tags {
			if s, ok := t.(string); ok {
				tagStrs = append(tagStrs, s)
			}
		}
		sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(tagStrs, ", ")))
	}

	// Parameters
	if params, ok := detailMap["parameters"].([]any); ok {
		sb.WriteString("Parameters:\n")
		for _, p := range params {
			if param, ok := p.(map[string]any); ok {
				name, _ := param["name"].(string)
				in, _ := param["in"].(string)
				desc, _ := param["description"].(string)
				sb.WriteString(fmt.Sprintf("  - %s (in: %s): %s\n", name, in, desc))
			}
		}
	}

	// Request body
	if reqBody, ok := detailMap["requestBody"].(map[string]any); ok {
		sb.WriteString("Request Body:\n")
		if desc, ok := reqBody["description"].(string); ok {
			sb.WriteString(fmt.Sprintf("  Description: %s\n", desc))
		}
		if content, ok := reqBody["content"].(map[string]any); ok {
			for contentType := range content {
				sb.WriteString(fmt.Sprintf("  Content-Type: %s\n", contentType))
			}
		}
	}

	// Responses
	if responses, ok := detailMap["responses"].(map[string]any); ok {
		sb.WriteString("Responses:\n")
		for code, resp := range responses {
			if respMap, ok := resp.(map[string]any); ok {
				desc, _ := respMap["description"].(string)
				sb.WriteString(fmt.Sprintf("  %s: %s\n", code, desc))
			}
		}
	}

	return sb.String()
}

func formatSchema(name string, schema any) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Schema: %s\n", name))

	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return sb.String()
	}

	if desc, ok := schemaMap["description"].(string); ok {
		sb.WriteString(fmt.Sprintf("Description: %s\n", desc))
	}
	if schemaType, ok := schemaMap["type"].(string); ok {
		sb.WriteString(fmt.Sprintf("Type: %s\n", schemaType))
	}

	if properties, ok := schemaMap["properties"].(map[string]any); ok {
		sb.WriteString("Properties:\n")
		for propName, prop := range properties {
			if propMap, ok := prop.(map[string]any); ok {
				propType, _ := propMap["type"].(string)
				propDesc, _ := propMap["description"].(string)
				sb.WriteString(fmt.Sprintf("  - %s (%s): %s\n", propName, propType, propDesc))
			}
		}
	}

	return sb.String()
}
