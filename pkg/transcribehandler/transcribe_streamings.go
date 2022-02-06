package transcribehandler

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
)

// addTranscribeStreamings adds the transcribe streamings
func (h *transcribeHandler) addTranscribeStreamings(transcribeID uuid.UUID, streamings []*streaming.Streaming) {
	defer h.transcribeStreamingsMu.Unlock()

	h.transcribeStreamingsMu.Lock()
	h.transcribeStreamingsMap[transcribeID] = streamings
}

// deleteTranscribeStreamings deletes the transcribe streamings
func (h *transcribeHandler) deleteTranscribeStreamings(transcribeID uuid.UUID) {
	defer h.transcribeStreamingsMu.Unlock()

	h.transcribeStreamingsMu.Lock()
	delete(h.transcribeStreamingsMap, transcribeID)
}

// getServiceStreaming returns streaming
func (h *transcribeHandler) getTranscribeStreamings(transcribeID uuid.UUID) []*streaming.Streaming {
	defer h.transcribeStreamingsMu.Unlock()

	h.transcribeStreamingsMu.Lock()
	if v, ok := h.transcribeStreamingsMap[transcribeID]; ok {
		return v
	}
	return nil
}
