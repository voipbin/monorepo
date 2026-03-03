package externalmediahandler

import (
	"context"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
)

func (h *externalMediaHandler) ARIPlaybackFinished(ctx context.Context, br *bridge.Bridge, e *ari.PlaybackFinished) error {
	return nil
}
