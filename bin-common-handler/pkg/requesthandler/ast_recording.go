package requesthandler

import (
	"context"
	"fmt"
	"monorepo/bin-common-handler/models/sock"
)

// AstRecordingStop stops the asterisk recording.
func (r *requestHandler) AstRecordingStop(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/stop", recordingName)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/recording/<recording_name>/stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AstRecordingPause pauses asterisk recording
func (r *requestHandler) AstRecordingPause(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/pause", recordingName)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/recording/<recording_name>/pause", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AstRecordingUnpause unpauses asterisk recording
func (r *requestHandler) AstRecordingUnpause(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/pause", recordingName)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/recording/<recording_name>/unpause", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AstRecordingUnpause mutes the asterisk recording
func (r *requestHandler) AstRecordingMute(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/mute", recordingName)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/recording/<recording_name>/mute", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// AstRecordingUnmute unmute the asterisk recording
func (r *requestHandler) AstRecordingUnmute(ctx context.Context, asteriskID, recordingName string) error {
	url := fmt.Sprintf("/ari/recordings/live/%s/mute", recordingName)

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodDelete, "ast/recording/<recording_name>/unmute", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
