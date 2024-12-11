package mediahandler

import "monorepo/bin-api-manager/pkg/streamhandler"

type MediaHandler interface {
}

type mediaHandler struct {
	streamHandler streamhandler.StreamHandler
}

func NewMediaHandler(streamHandler streamhandler.StreamHandler) MediaHandler {
	return &mediaHandler{
		streamHandler: streamHandler,
	}
}
