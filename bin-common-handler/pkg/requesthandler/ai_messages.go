package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cbmessage "monorepo/bin-ai-manager/models/message"
	cbrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// AIV1MessageGetsByAIcallID sends a request to ai-manager
// to getting a list of messages info of the given aicall id.
// it returns detail list of message info if it succeed.
func (r *requestHandler) AIV1MessageGetsByAIcallID(ctx context.Context, aicallID uuid.UUID, pageToken string, pageSize uint64, filters map[cbmessage.Field]any) ([]cbmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// Add aicall_id to filters
	if filters == nil {
		filters = make(map[cbmessage.Field]any)
	}
	filters[cbmessage.FieldAIcallID] = aicallID

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/messages", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cbmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// AIV1MessageSend sends a request to ai-manager
// to send a message.
// it returns created message if it succeed.
func (r *requestHandler) AIV1MessageSend(ctx context.Context, aicallID uuid.UUID, role cbmessage.Role, content string, runImmediately bool, audioResponse bool, timeout int) (*cbmessage.Message, error) {
	uri := "/v1/messages"

	data := &cbrequest.V1DataMessagesPost{
		AIcallID: aicallID,
		Role:     role,
		Content:  content,

		RunImmediately: runImmediately,
		AudioResponse:  audioResponse,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "ai/messages", timeout, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cbmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1MessageGet returns the message.
func (r *requestHandler) AIV1MessageGet(ctx context.Context, messageID uuid.UUID) (*cbmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", messageID.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/messages/<message-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cbmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// AIV1MessageDelete sends a request to ai-manager
// to deleting a message.
// it returns deleted message if it succeed.
func (r *requestHandler) AIV1MessageDelete(ctx context.Context, messageID uuid.UUID) (*cbmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages/%s", messageID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "ai/messages/<message-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cbmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
