package streaminghandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/streaming"
)

type awsHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	accessKey string
	secretKey string
}

func NewAWSHandler(reqHandler requesthandler.RequestHandler, notifyHandler notifyhandler.NotifyHandler, accessKey string, secretKey string) streamer {
	return &awsHandler{
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		accessKey:     accessKey,
		secretKey:     secretKey,
	}
}

func (h *awsHandler) Init(ctx context.Context, st *streaming.Streaming) (any, error) {
	return nil, fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) Run(vendorConfig any) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) SayStop(vendorConfig any) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) SayAdd(vendorConfig any, text string) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) SayFlush(vendorConfig any) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) SayFinish(vendorConfig any) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}
