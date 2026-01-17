package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	talkparticipant "monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// TalkV1TalkParticipantList gets participants for a talk
func (r *requestHandler) TalkV1TalkParticipantList(ctx context.Context, talkID uuid.UUID) ([]*talkparticipant.Participant, error) {
	uri := fmt.Sprintf("/v1/talk_chats/%s/participants", talkID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/participants", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get participants: status %d", res.StatusCode)
	}

	var participants []*talkparticipant.Participant
	if err := json.Unmarshal(res.Data, &participants); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal participants")
	}

	return participants, nil
}

// TalkV1TalkParticipantCreate adds a participant to a talk
func (r *requestHandler) TalkV1TalkParticipantCreate(ctx context.Context, talkID uuid.UUID, ownerType string, ownerID uuid.UUID) (*talkparticipant.Participant, error) {
	uri := fmt.Sprintf("/v1/talk_chats/%s/participants", talkID.String())

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

	var participant talkparticipant.Participant
	if err := json.Unmarshal(res.Data, &participant); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal participant")
	}

	return &participant, nil
}

// TalkV1TalkParticipantDelete removes a participant from a talk
func (r *requestHandler) TalkV1TalkParticipantDelete(ctx context.Context, talkID uuid.UUID, participantID uuid.UUID) (*talkparticipant.Participant, error) {
	uri := fmt.Sprintf("/v1/talk_chats/%s/participants/%s", talkID.String(), participantID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodDelete, "talk/participants", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to delete participant: status %d", res.StatusCode)
	}

	var participant talkparticipant.Participant
	if err := json.Unmarshal(res.Data, &participant); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal participant")
	}

	return &participant, nil
}
