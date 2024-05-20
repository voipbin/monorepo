package requesthandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// AstRecordingStop stops the asterisk recording.
func (r *requestHandler) AstRecordingStop(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/stop", recordingName)

	res, err := r.sendRequestAst(ctx, asteriskID, url, rabbitmqhandler.RequestMethodPost, "ast/recording/<recording_name>/stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstRecordingPause pauses asterisk recording
func (r *requestHandler) AstRecordingPause(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/pause", recordingName)

	res, err := r.sendRequestAst(ctx, asteriskID, url, rabbitmqhandler.RequestMethodPost, "ast/recording/<recording_name>/pause", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstRecordingUnpause unpauses asterisk recording
func (r *requestHandler) AstRecordingUnpause(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/pause", recordingName)

	res, err := r.sendRequestAst(ctx, asteriskID, url, rabbitmqhandler.RequestMethodDelete, "ast/recording/<recording_name>/unpause", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstRecordingUnpause mutes the asterisk recording
func (r *requestHandler) AstRecordingMute(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/mute", recordingName)

	res, err := r.sendRequestAst(ctx, asteriskID, url, rabbitmqhandler.RequestMethodPost, "ast/recording/<recording_name>/mute", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}

// AstRecordingUnmute unmute the asterisk recording
func (r *requestHandler) AstRecordingUnmute(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/mute", recordingName)

	res, err := r.sendRequestAst(ctx, asteriskID, url, rabbitmqhandler.RequestMethodDelete, "ast/recording/<recording_name>/unmute", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
