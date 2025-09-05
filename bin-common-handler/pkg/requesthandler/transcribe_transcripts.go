package requesthandler

import (
	"context"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
)

// TranscribeV1TranscriptGets sends a request to transcribe-manager
// to getting a list of transcript info.
// it returns detail list of transcript info if it succeed.
func (r *requestHandler) TranscribeV1TranscriptGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]tmtranscript.Transcript, error) {
	uri := fmt.Sprintf("/v1/transcripts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestTranscribe(ctx, uri, sock.RequestMethodGet, "transcribe/transcripts", 30000, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res []tmtranscript.Transcript
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
