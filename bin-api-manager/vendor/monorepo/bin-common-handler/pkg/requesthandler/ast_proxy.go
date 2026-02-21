package requesthandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"

	"github.com/pkg/errors"
)

// AstProxyRecordingFileMove moves the recording file to the bucket.
func (r *requestHandler) AstProxyRecordingFileMove(ctx context.Context, asteriskID string, filenames []string, timeout int) error {
	url := "/proxy/recording_file_move"

	type Data struct {
		Filenames []string `json:"filenames,omitempty"`
	}

	m, err := json.Marshal(Data{
		Filenames: filenames,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal data")
	}

	tmp, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/proxy/recording_file_move", timeout, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
