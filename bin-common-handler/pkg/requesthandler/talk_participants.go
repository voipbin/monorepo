package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/sock"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
)

// TalkV1ParticipantList gets participants for a talk
func (r *requestHandler) TalkV1ParticipantList(ctx context.Context, chatID uuid.UUID) ([]*tkparticipant.Participant, error) {
	uri := fmt.Sprintf("/v1/chats/%s/participants", chatID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/participants", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list participants: status %d", res.StatusCode)
	}

	var participants []*tkparticipant.Participant
	if err := json.Unmarshal(res.Data, &participants); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal participants")
	}

	return participants, nil
}

// TalkV1ParticipantCreate adds a participant to a talk
func (r *requestHandler) TalkV1ParticipantCreate(ctx context.Context, chatID uuid.UUID, ownerType string, ownerID uuid.UUID) (*tkparticipant.Participant, error) {
	uri := fmt.Sprintf("/v1/chats/%s/participants", chatID.String())

	data := map[string]any{
		"owner_type": ownerType,
		"owner_id":   ownerID.String(),
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal request")
	}

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodPost, "talk/participants", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 && res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to create participant: status %d", res.StatusCode)
	}

	var participant tkparticipant.Participant
	if err := json.Unmarshal(res.Data, &participant); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal participant")
	}

	return &participant, nil
}

// TalkV1ParticipantDelete removes a participant from a talk
func (r *requestHandler) TalkV1ParticipantDelete(ctx context.Context, chatID, participantID uuid.UUID) (*tkparticipant.Participant, error) {
	uri := fmt.Sprintf("/v1/chats/%s/participants/%s", chatID.String(), participantID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodDelete, "talk/participants", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to delete participant: status %d", res.StatusCode)
	}

	var participant tkparticipant.Participant
	if err := json.Unmarshal(res.Data, &participant); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal participant")
	}

	return &participant, nil
}

// TalkV1ParticipantListWithFilters gets participants with filters
func (r *requestHandler) TalkV1ParticipantListWithFilters(ctx context.Context, filters map[string]any, pageToken string, pageSize uint64) ([]*tkparticipant.Participant, error) {
	uri := fmt.Sprintf("/v1/participants?page_token=%s&page_size=%d", pageToken, pageSize)

	// Marshal filters to JSON
	data, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal filters")
	}

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/participants", requestTimeoutDefault, 0, ContentTypeJSON, data)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list participants: status %d", res.StatusCode)
	}

	var participants []*tkparticipant.Participant
	if err := json.Unmarshal(res.Data, &participants); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal participants")
	}

	return participants, nil
}
