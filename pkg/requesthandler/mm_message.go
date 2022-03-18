package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	mmrequest "gitlab.com/voipbin/bin-manager/message-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// MMV1MessageSend sends a message
func (r *requestHandler) MMV1MessageSend(ctx context.Context, customerID uuid.UUID, source *cmaddress.Address, destinations []cmaddress.Address, text string) (*mmmessage.Message, error) {

	uri := "/v1/messages"

	reqData := &mmrequest.V1DataMessagesPost{
		CustomerID:   customerID,
		Source:       source,
		Destinations: destinations,
		Text:         text,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestMM(uri, rabbitmqhandler.RequestMethodPost, resourceMMMessages, requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// MMV1MessageGet gets the messages
func (r *requestHandler) MMV1MessageGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]mmmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	res, err := r.sendRequestMM(uri, rabbitmqhandler.RequestMethodGet, resourceMMMessages, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// MMV1MessageGet gets the message
func (r *requestHandler) MMV1MessageGet(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", id)

	tmp, err := r.sendRequestMM(uri, rabbitmqhandler.RequestMethodGet, resourceMMMessages, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// MMV1MessageDelete deletes the message
func (r *requestHandler) MMV1MessageDelete(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", id)

	tmp, err := r.sendRequestMM(uri, rabbitmqhandler.RequestMethodDelete, resourceMMMessages, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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


