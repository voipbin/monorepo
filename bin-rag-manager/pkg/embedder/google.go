package embedder

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// Embedder defines the interface for text embedding
type Embedder interface {
	EmbedTexts(ctx context.Context, texts []string) ([][]float32, error)
	EmbedText(ctx context.Context, text string) ([]float32, error)
}

// googleEmbedder implements Embedder using Google's Gemini embedding API
type googleEmbedder struct {
	client *genai.Client
	model  string
}

// NewGoogleEmbedder creates a new Google Vertex AI embedder.
// Authentication uses Application Default Credentials (GKE Workload Identity).
func NewGoogleEmbedder(ctx context.Context, project, location, model string) (Embedder, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  project,
		Location: location,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create genai client: %w", err)
	}

	return &googleEmbedder{
		client: client,
		model:  model,
	}, nil
}

// EmbedTexts embeds multiple texts and returns their embedding vectors
func (e *googleEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	allEmbeddings := make([][]float32, 0, len(texts))

	for _, text := range texts {
		result, err := e.client.Models.EmbedContent(
			ctx,
			e.model,
			genai.Text(text),
			&genai.EmbedContentConfig{
				TaskType: "RETRIEVAL_DOCUMENT",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("could not create embedding: %w", err)
		}

		if len(result.Embeddings) == 0 {
			return nil, fmt.Errorf("no embedding returned for text")
		}

		allEmbeddings = append(allEmbeddings, result.Embeddings[0].Values)
	}

	return allEmbeddings, nil
}

// EmbedText embeds a single text and returns its embedding vector
func (e *googleEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	result, err := e.client.Models.EmbedContent(
		ctx,
		e.model,
		genai.Text(text),
		&genai.EmbedContentConfig{
			TaskType: "RETRIEVAL_QUERY",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not create embedding: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return result.Embeddings[0].Values, nil
}
