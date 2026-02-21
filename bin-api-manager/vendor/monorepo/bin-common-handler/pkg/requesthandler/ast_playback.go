package requesthandler

import (
	"context"
	"fmt"
	"monorepo/bin-common-handler/models/sock"
)

// AstPlaybackStop stops the playback on the channel
func (r *requestHandler) AstPlaybackStop(ctx context.Context, asteriskID string, playabckID string) error {
	url := fmt.Sprintf("/ari/playbacks/%s", playabckID)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/playbacks", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
