package requesthandler

import (
	"context"
	"fmt"
	"net/url"

	amparticipant "monorepo/bin-ai-manager/models/participant"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

// AIV1AIcallParticipantList sends a request to ai-manager
// to get a list of participants for the given aicall id.
// it returns a list of participant webhook messages if it succeeds.
func (r *requestHandler) AIV1AIcallParticipantList(ctx context.Context, aicallID uuid.UUID, pageToken string, pageSize uint64) ([]*amparticipant.WebhookMessage, error) {
	uri := fmt.Sprintf("/v1/aicalls/%s/participants?page_size=%d&page_token=%s", aicallID, pageSize, url.QueryEscape(pageToken))

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/aicalls/<aicall-id>/participants", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []*amparticipant.WebhookMessage
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1AIParticipantList sends a request to ai-manager
// to get a list of participants for the given ai id.
// it returns a list of participant webhook messages if it succeeds.
func (r *requestHandler) AIV1AIParticipantList(ctx context.Context, aiID uuid.UUID, pageToken string, pageSize uint64) ([]*amparticipant.WebhookMessage, error) {
	uri := fmt.Sprintf("/v1/ais/%s/participants?page_size=%d&page_token=%s", aiID, pageSize, url.QueryEscape(pageToken))

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/ais/<ai-id>/participants", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []*amparticipant.WebhookMessage
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
