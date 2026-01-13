package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/pkg/errors"
)

// TranscribeV1TranscriptGets sends a request to transcribe-manager
// to getting a list of transcript info.
// it returns detail list of transcript info if it succeed.
func (r *requestHandler) TranscribeV1TranscriptGets(ctx context.Context, pageToken string, pageSize uint64, filters map[tmtranscript.Field]any) ([]tmtranscript.Transcript, error) {
	uri := fmt.Sprintf("/v1/transcripts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestTranscribe(ctx, uri, sock.RequestMethodGet, "transcribe/transcripts", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []tmtranscript.Transcript
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
