package requesthandler

import (
	"context"
	"fmt"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// AstPlaybackStop stops the playback on the channel
func (r *requestHandler) AstPlaybackStop(ctx context.Context, asteriskID string, playabckID string) error {
	url := fmt.Sprintf("/ari/playbacks/%s", playabckID)

	res, err := r.sendRequestAst(ctx, asteriskID, url, rabbitmqhandler.RequestMethodDelete, resourceAstPlaybacks, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
