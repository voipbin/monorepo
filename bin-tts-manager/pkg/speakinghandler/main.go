package speakinghandler

import (
	"context"

	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/dbhandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"

	"github.com/gofrs/uuid"
)

//go:generate mockgen -package speakinghandler -destination mock_main.go -source main.go

// SpeakingHandler handles speaking session lifecycle
type SpeakingHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, referenceType streaming.ReferenceType, referenceID uuid.UUID, language, provider, voiceID string, direction streaming.Direction) (*speaking.Speaking, error)
	Get(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
	Gets(ctx context.Context, token string, size uint64, filters map[speaking.Field]any) ([]*speaking.Speaking, error)
	Say(ctx context.Context, id uuid.UUID, text string) (*speaking.Speaking, error)
	Flush(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
	Stop(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
	Delete(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
}

type speakingHandler struct {
	db               dbhandler.DBHandler
	streamingHandler streaminghandler.StreamingHandler
	podID            string
}

// NewSpeakingHandler creates a new SpeakingHandler
func NewSpeakingHandler(
	db dbhandler.DBHandler,
	streamingHandler streaminghandler.StreamingHandler,
	podID string,
) SpeakingHandler {
	return &speakingHandler{
		db:               db,
		streamingHandler: streamingHandler,
		podID:            podID,
	}
}
