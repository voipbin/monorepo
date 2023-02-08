package chatgpthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package chatgpthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"os"

	"github.com/otiai10/openaigo"
)

// ChatgptHandler define
type ChatgptHandler interface {
	Chat(ctx context.Context, text string) (string, error)
}

const (
	constModel = "text-davinci-003" // default gpt model fot chatgpt
)

// chatgptHandler define
type chatgptHandler struct {
	client *openaigo.Client
}

// NewChatgptHandler define
func NewChatgptHandler(apiKey string) ChatgptHandler {
	client := openaigo.NewClient(os.Getenv(apiKey))

	return &chatgptHandler{
		client: client,
	}
}
