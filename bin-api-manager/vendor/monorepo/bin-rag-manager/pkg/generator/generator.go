package generator

import (
	"context"
	"fmt"
	"strings"

	"monorepo/bin-rag-manager/pkg/store"

	openai "github.com/sashabaranov/go-openai"
)

const systemPrompt = `You are a helpful assistant for VoIPBin, a cloud-native Communication Platform as a Service (CPaaS).
Answer the user's question using ONLY the provided context below. Follow these rules:
1. Base your answer strictly on the provided context. Do not use external knowledge.
2. Cite your sources by referencing the source file and section title.
3. If the context does not contain enough information to answer the question, say "I don't have enough information to answer that question" and suggest what documentation the user might look for.
4. Be concise and direct in your answers.
5. When referencing API endpoints, include the HTTP method and path.`

// Generator defines the interface for answer generation
type Generator interface {
	Generate(ctx context.Context, query string, chunks []store.SearchResult) (string, error)
}

type generator struct {
	client *openai.Client
	model  string
}

// NewGenerator creates a new Generator
func NewGenerator(apiKey string, model string) Generator {
	client := openai.NewClient(apiKey)
	return &generator{
		client: client,
		model:  model,
	}
}

// Generate builds a prompt from the query and retrieved chunks, then calls the LLM
func (g *generator) Generate(ctx context.Context, query string, chunks []store.SearchResult) (string, error) {
	if len(chunks) == 0 {
		return "I don't have any relevant documentation to answer that question.", nil
	}

	userMessage := buildUserMessage(query, chunks)

	resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: g.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userMessage,
			},
		},
		Temperature: 0.1,
	})
	if err != nil {
		return "", fmt.Errorf("could not generate answer: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

func buildUserMessage(query string, chunks []store.SearchResult) string {
	var sb strings.Builder

	sb.WriteString("Context:\n\n")
	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("[%d] Source: %s | Section: %s | Type: %s\n",
			i+1,
			chunk.Chunk.SourceFile,
			chunk.Chunk.SectionTitle,
			chunk.Chunk.DocType,
		))
		sb.WriteString(chunk.Chunk.Text)
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("Question: %s", query))

	return sb.String()
}
