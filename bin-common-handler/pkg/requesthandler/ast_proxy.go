package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/sock"

	"github.com/pkg/errors"
)

// AstProxyRecordingFileMove moves the recording file to the bucket.
func (r *requestHandler) AstProxyRecordingFileMove(ctx context.Context, asteriskID string, filenames []string) error {
	url := "/proxy/recording_file_move"

	type Data struct {
		Filenames []string `json:"filenames,omitempty"`
	}

	tmpData := &Data{
		Filenames: filenames,
	}

	m, err := json.Marshal(tmpData)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal data")
	}

	res, err := r.sendRequestAst(ctx, asteriskID, url, sock.RequestMethodPost, "ast/proxy/recording_file_move", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}
	return nil
}
