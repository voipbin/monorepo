package requesthandler

import (
	"context"
	"fmt"
	"monorepo/bin-common-handler/models/sock"
)

// AstPlaybackStop stops the playback on the channel
func (r *requestHandler) AstPlaybackStop(ctx context.Context, asteriskID string, playabckID string) error {
	url := fmt.Sprintf("/ari/playbacks/%s", playabckID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/playbacks", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
