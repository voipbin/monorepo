package chunker

import (
	"os"
	"strings"
	"testing"
)

func TestOpenAPIChunker_Chunk(t *testing.T) {
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: List users
      description: Returns a list of users
      tags:
        - users
      parameters:
        - name: limit
          in: query
          description: Maximum number of results
      responses:
        200:
          description: Success
    post:
      summary: Create user
      description: Creates a new user
      requestBody:
        description: User data
        content:
          application/json:
            schema:
              type: object
      responses:
        201:
          description: Created
components:
  schemas:
    User:
      type: object
      description: User model
      properties:
        id:
          type: string
          description: User ID
        name:
          type: string
          description: User name
    Profile:
      type: object
      description: Profile model
      properties:
        avatar:
          type: string
          description: Avatar URL
`

	tmpFile, err := os.CreateTemp("", "test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewOpenAPIChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Verify doc type
	for _, chunk := range chunks {
		if chunk.DocType != DocTypeOpenAPI {
			t.Errorf("expected doc type openapi, got %s", chunk.DocType)
		}
	}

	// Verify we have both endpoint and schema chunks
	hasEndpoint := false
	hasSchema := false
	for _, chunk := range chunks {
		if strings.Contains(chunk.SectionTitle, "GET /users") {
			hasEndpoint = true
		}
		if strings.Contains(chunk.SectionTitle, "Schema: User") {
			hasSchema = true
		}
	}
	if !hasEndpoint {
		t.Error("expected endpoint chunk for GET /users")
	}
	if !hasSchema {
		t.Error("expected schema chunk for User")
	}
}

func TestOpenAPIChunker_InvalidYAML(t *testing.T) {
	content := `invalid: yaml: content: [[[`

	tmpFile, err := os.CreateTemp("", "test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewOpenAPIChunker()
	_, err = c.Chunk(tmpFile.Name(), 800)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestOpenAPIChunker_NonExistentFile(t *testing.T) {
	c := NewOpenAPIChunker()
	_, err := c.Chunk("/nonexistent/file.yaml", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestFormatEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		method   string
		details  any
		expected []string
	}{
		{
			name:   "full endpoint details",
			path:   "/users",
			method: "get",
			details: map[string]any{
				"summary":     "List users",
				"description": "Returns a list of users",
				"tags":        []any{"users", "admin"},
				"parameters": []any{
					map[string]any{
						"name":        "limit",
						"in":          "query",
						"description": "Maximum results",
					},
				},
				"requestBody": map[string]any{
					"description": "User data",
					"content": map[string]any{
						"application/json": map[string]any{},
					},
				},
				"responses": map[string]any{
					"200": map[string]any{
						"description": "Success",
					},
				},
			},
			expected: []string{"Endpoint: GET /users", "Summary:", "Description:", "Tags:", "Parameters:", "Request Body:", "Responses:"},
		},
		{
			name:     "minimal endpoint",
			path:     "/health",
			method:   "get",
			details:  map[string]any{},
			expected: []string{"Endpoint: GET /health"},
		},
		{
			name:     "invalid details type",
			path:     "/test",
			method:   "post",
			details:  "invalid",
			expected: []string{"Endpoint: POST /test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatEndpoint(tt.path, tt.method, tt.details)
			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q, got: %s", expected, result)
				}
			}
		})
	}
}

func TestFormatSchema(t *testing.T) {
	tests := []struct {
		name     string
		schemaName string
		schema   any
		expected []string
	}{
		{
			name:       "full schema",
			schemaName: "User",
			schema: map[string]any{
				"type":        "object",
				"description": "User model",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "User ID",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "User name",
					},
				},
			},
			expected: []string{"Schema: User", "Description:", "Type:", "Properties:"},
		},
		{
			name:       "minimal schema",
			schemaName: "Empty",
			schema:     map[string]any{},
			expected:   []string{"Schema: Empty"},
		},
		{
			name:       "invalid schema type",
			schemaName: "Invalid",
			schema:     "invalid",
			expected:   []string{"Schema: Invalid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSchema(tt.schemaName, tt.schema)
			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q, got: %s", expected, result)
				}
			}
		})
	}
}

func TestOpenAPIChunker_EmptyPaths(t *testing.T) {
	content := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
`

	tmpFile, err := os.CreateTemp("", "test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewOpenAPIChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty spec, got %d", len(chunks))
	}
}
