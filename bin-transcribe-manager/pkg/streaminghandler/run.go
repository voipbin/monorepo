package streaminghandler

// Run is a no-op. With WebSocket transport, transcribe-manager dials out to
// Asterisk per-session in Start() instead of listening on a TCP port. Kept to
// satisfy the StreamingHandler interface.
func (h *streamingHandler) Run() error {
	return nil
}
