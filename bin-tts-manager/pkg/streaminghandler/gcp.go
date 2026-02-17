package streaminghandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/streaming"
)

type gcpHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

func NewGCPHandler(reqHandler requesthandler.RequestHandler, notifyHandler notifyhandler.NotifyHandler) streamer {
	return &gcpHandler{
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}
}

func (h *gcpHandler) Init(ctx context.Context, st *streaming.Streaming) (any, error) {
	return nil, fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) Run(vendorConfig any) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) SayStop(vendorConfig any) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) SayAdd(vendorConfig any, text string) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) SayFlush(vendorConfig any) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) SayFinish(vendorConfig any) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}
