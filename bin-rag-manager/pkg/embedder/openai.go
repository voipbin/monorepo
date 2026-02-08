package embedder

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// Embedder defines the interface for text embedding
type Embedder interface {
	EmbedTexts(ctx context.Context, texts []string) ([][]float32, error)
	EmbedText(ctx context.Context, text string) ([]float32, error)
}

// openaiEmbedder implements Embedder using OpenAI's embedding API
type openaiEmbedder struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

// NewOpenAIEmbedder creates a new OpenAI embedder
func NewOpenAIEmbedder(apiKey string, model string) Embedder {
	client := openai.NewClient(apiKey)
	return &openaiEmbedder{
		client: client,
		model:  openai.EmbeddingModel(model),
	}
}

// EmbedTexts embeds multiple texts and returns their embedding vectors
func (e *openaiEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// OpenAI supports up to 2048 inputs per request, batch if needed
	const batchSize = 2048
	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
			Input: batch,
			Model: e.model,
		})
		if err != nil {
			return nil, fmt.Errorf("could not create embeddings: %w", err)
		}

		for _, embedding := range resp.Data {
			allEmbeddings = append(allEmbeddings, embedding.Embedding)
		}
	}

	return allEmbeddings, nil
}

// EmbedText embeds a single text and returns its embedding vector
func (e *openaiEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.EmbedTexts(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}
