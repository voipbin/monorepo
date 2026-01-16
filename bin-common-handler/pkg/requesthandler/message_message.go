package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	mmmessage "monorepo/bin-message-manager/models/message"
	mmrequest "monorepo/bin-message-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
)

// MessageV1MessageSend sends a message
func (r *requestHandler) MessageV1MessageSend(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *address.Address, destinations []address.Address, text string) (*mmmessage.Message, error) {

	uri := "/v1/messages"

	reqData := &mmrequest.V1DataMessagesPost{
		ID:           id,
		CustomerID:   customerID,
		Source:       source,
		Destinations: destinations,
		Text:         text,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestMessage(ctx, uri, sock.RequestMethodPost, "message/messages", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res mmmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// MessageV1MessageGets gets the messages
func (r *requestHandler) MessageV1MessageList(ctx context.Context, pageToken string, pageSize uint64, filters map[mmmessage.Field]any) ([]mmmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestMessage(ctx, uri, sock.RequestMethodGet, "message/messages", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []mmmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// MessageV1MessageGet gets the message
func (r *requestHandler) MessageV1MessageGet(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", id)

	tmp, err := r.sendRequestMessage(ctx, uri, sock.RequestMethodGet, "message/messages", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res mmmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// MessageV1MessageDelete deletes the message
func (r *requestHandler) MessageV1MessageDelete(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", id)

	tmp, err := r.sendRequestMessage(ctx, uri, sock.RequestMethodDelete, "message/messages", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res mmmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
