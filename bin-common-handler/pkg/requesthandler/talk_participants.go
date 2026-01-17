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
	uri := fmt.Sprintf("/v1/talks/%s/participants", talkID.String())

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
