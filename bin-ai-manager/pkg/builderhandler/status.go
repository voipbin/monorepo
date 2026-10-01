package builderhandler

import "monorepo/bin-ai-manager/models/builder"

// Status reports whether the Builder is usable and the input limits the client
// should enforce. Available is true only when the key it needs is configured, so
// a self-hosted install without the key hides the card instead of showing a
// button that always fails.
//
// It takes no semaphore and no counter: it is a plain read.
func (h *builderHandler) Status() *builder.StatusResponse {
	return &builder.StatusResponse{
		Available:       h.opts.KeyConfigured,
		MaxMessages:     builder.MaxMessages,
		MaxMessageChars: builder.MaxMessageRunes,
	}
}
