package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/sock"
	talkmessage "monorepo/bin-talk-manager/models/message"
	tmrequest "monorepo/bin-talk-manager/pkg/listenhandler/models/request"
)

// TalkV1MentionList gets the messages that mention the agent, newest first.
func (r *requestHandler) TalkV1MentionList(ctx context.Context, customerID uuid.UUID, agentID uuid.UUID, pageToken string, pageSize uint64) ([]*talkmessage.Message, error) {
	uri := fmt.Sprintf("/v1/mentions?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	data, err := json.Marshal(tmrequest.V1DataMentionsGet{
		CustomerID: customerID.String(),
		AgentID:    agentID.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal the request")
	}

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/mentions", requestTimeoutDefault, 0, ContentTypeJSON, data)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list mentions: status %d", res.StatusCode)
	}

	var messages []*talkmessage.Message
	if err := json.Unmarshal(res.Data, &messages); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal messages")
	}

	return messages, nil
}
