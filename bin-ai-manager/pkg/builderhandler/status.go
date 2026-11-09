package builderhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/builder"
)

// Status reports whether the Builder is usable and the input limits the client
// should enforce. Available is true only when the key it needs is configured, so
// a self-hosted install without the key hides the card instead of showing a
// button that always fails. The key is read now: a key that was just set or
// cleared shows at once. A key that can not be read is reported as not
// available (the card is hidden until the read works again).
//
// It takes no semaphore and no counter.
func (h *builderHandler) Status(ctx context.Context) *builder.StatusResponse {
	available, err := h.opts.keyConfigured(ctx)
	if err != nil {
		logrus.WithField("func", "Status").WithError(err).Warn("Could not read the builder key; reporting the builder as unavailable.")
		available = false
	}

	return &builder.StatusResponse{
		Available:       available,
		MaxMessages:     builder.MaxMessages,
		MaxMessageChars: builder.MaxMessageRunes,
	}
}
