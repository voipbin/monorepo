package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	mmmessage "monorepo/bin-message-manager/models/message"
	mmrequest "monorepo/bin-message-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
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

	tmp, err := r.sendRequestMessage(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceMessageMessages, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not send the message")
	}

	var res mmmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MessageV1MessageGet gets the messages
func (r *requestHandler) MessageV1MessageGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]mmmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	res, err := r.sendRequestMessage(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceMessageMessages, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var f []mmmessage.Message
	if err := json.Unmarshal([]byte(res.Data), &f); err != nil {
		return nil, err
	}

	return f, nil
}

// MessageV1MessageGet gets the message
func (r *requestHandler) MessageV1MessageGet(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", id)

	tmp, err := r.sendRequestMessage(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceMessageMessages, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get a message")
	}

	var res mmmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MessageV1MessageDelete deletes the message
func (r *requestHandler) MessageV1MessageDelete(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", id)

	tmp, err := r.sendRequestMessage(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceMessageMessages, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get a message")
	}

	var res mmmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
