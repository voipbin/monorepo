package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	talktalk "monorepo/bin-talk-manager/models/talk"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// TalkV1TalkGet gets a talk by ID
func (r *requestHandler) TalkV1TalkGet(ctx context.Context, talkID uuid.UUID) (*talktalk.Talk, error) {
	uri := fmt.Sprintf("/v1/talks/%s", talkID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/talks", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get talk: status %d", res.StatusCode)
	}

	var talk talktalk.Talk
	if err := json.Unmarshal(res.Data, &talk); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal talk")
	}

	return &talk, nil
}

// TalkV1TalkCreate creates a new talk
func (r *requestHandler) TalkV1TalkCreate(ctx context.Context, customerID uuid.UUID, talkType talktalk.Type) (*talktalk.Talk, error) {
	uri := "/v1/talks"

	data := map[string]any{
		"customer_id": customerID.String(),
		"type":        string(talkType),
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal request")
	}

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodPost, "talk/talks", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 && res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to create talk: status %d", res.StatusCode)
	}

	var talk talktalk.Talk
	if err := json.Unmarshal(res.Data, &talk); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal talk")
	}

	return &talk, nil
}

// TalkV1TalkDelete deletes a talk (soft delete)
func (r *requestHandler) TalkV1TalkDelete(ctx context.Context, talkID uuid.UUID) (*talktalk.Talk, error) {
	uri := fmt.Sprintf("/v1/talks/%s", talkID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodDelete, "talk/talks", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to delete talk: status %d", res.StatusCode)
	}

	var talk talktalk.Talk
	if err := json.Unmarshal(res.Data, &talk); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal talk")
	}

	return &talk, nil
}

// TalkV1TalkList gets a list of talks (simplified - for future expansion)
func (r *requestHandler) TalkV1TalkList(ctx context.Context, pageToken string, pageSize uint64) ([]*talktalk.Talk, error) {
	uri := fmt.Sprintf("/v1/talks?page_token=%s&page_size=%d", pageToken, pageSize)

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/talks", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list talks: status %d", res.StatusCode)
	}

	var talks []*talktalk.Talk
	if err := json.Unmarshal(res.Data, &talks); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal talks")
	}

	return talks, nil
}
