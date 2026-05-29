package geminiproposalhandler

//go:generate mockgen -package geminiproposalhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/pkg/geminiaudithandler"
)

const (
	geminiEndpoint         = "https://generativelanguage.googleapis.com/v1beta/openai/"
	geminiModel            = "gemini-2.5-pro"
	maxProposedPromptChars = 32000
	maxRationaleChars      = 4000
)

// proposalJSONSchema is the JSON Schema passed to Gemini via response_format.json_schema.
var proposalJSONSchema = json.RawMessage(`{
  "type": "object",
  "required": ["proposed_prompt", "rationale"],
  "properties": {
    "proposed_prompt": {"type": "string"},
    "rationale":       {"type": "string"}
  }
}`)

// AuditBlock holds the evaluation + transcript for one source audit.
type AuditBlock struct {
	Index           int
	OverallScore    int
	HelpfulnessR    string
	AccuracyR       string
	ToneR           string
	GoalCompletionR string
	ToolUsageR      string
	Summary         string
	Transcript      string
}

// ProposalResponse is the parsed Gemini result.
type ProposalResponse struct {
	ProposedPrompt string `json:"proposed_prompt"`
	Rationale      string `json:"rationale"`
}

// GeminiProposalHandler wraps the Gemini call for prompt-rewrite proposals.
type GeminiProposalHandler interface {
	Evaluate(ctx context.Context, originalPrompt string, audits []AuditBlock, language string) (*ProposalResponse, error)
	BuildPrompt(originalPrompt string, audits []AuditBlock, language string) string
	ParseProposalResponse(data []byte) (*ProposalResponse, error)
}

type geminiProposalHandler struct {
	client *openai.Client
}

// NewGeminiProposalHandler creates a new handler using the given API key.
func NewGeminiProposalHandler(apiKey string) *geminiProposalHandler {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = geminiEndpoint
	return &geminiProposalHandler{client: openai.NewClientWithConfig(cfg)}
}

// sanitize delegates to geminiaudithandler so the delimiter convention has one source of truth.
func sanitize(text string) string {
	return geminiaudithandler.NewGeminiAuditHandler("").Sanitize(text)
}

// Suppress unused-import warnings until later tasks use these.
var (
	_ = fmt.Sprint
	_ = strings.Contains
	_ = logrus.New
)
