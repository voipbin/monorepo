package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/sock"
	tkpin "monorepo/bin-talk-manager/models/pin"
	tmrequest "monorepo/bin-talk-manager/pkg/listenhandler/models/request"
)

// TalkV1PinCreate pins a chat or a message for the owner and returns the pin. Pinning the same
// target again returns the existing pin. A target that does not exist or is not visible to the
// customer is reported as ErrNotFound.
func (r *requestHandler) TalkV1PinCreate(ctx context.Context, customerID uuid.UUID, ownerType string, ownerID uuid.UUID, targetType tkpin.TargetType, targetID uuid.UUID) (*tkpin.Pin, error) {
	uri := "/v1/pins"

	m, err := json.Marshal(tmrequest.V1DataPinsPost{
		CustomerID: customerID.String(),
		OwnerType:  ownerType,
		OwnerID:    ownerID.String(),
		TargetType: string(targetType),
		TargetID:   targetID.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal the request")
	}

	tmp, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodPost, "talk/pins", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tkpin.Pin
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TalkV1PinList returns the pins of the owner that the owner can see, newest first.
// targetType and chatID filter the result when they are set (empty or uuid.Nil: no filter).
func (r *requestHandler) TalkV1PinList(ctx context.Context, customerID uuid.UUID, ownerType string, ownerID uuid.UUID, targetType tkpin.TargetType, chatID uuid.UUID, pageToken string, pageSize uint64) ([]*tkpin.Pin, error) {
	uri := fmt.Sprintf("/v1/pins?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	req := tmrequest.V1DataPinsGet{
		CustomerID: customerID.String(),
		OwnerType:  ownerType,
		OwnerID:    ownerID.String(),
		TargetType: string(targetType),
	}
	if chatID != uuid.Nil {
		req.ChatID = chatID.String()
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal the request")
	}

	tmp, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/pins", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*tkpin.Pin
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// TalkV1PinDelete deletes a pin of the owner and returns the deleted pin. The pin of another owner
// or customer, and an unknown id, are reported as ErrNotFound.
func (r *requestHandler) TalkV1PinDelete(ctx context.Context, pinID, customerID uuid.UUID, ownerType string, ownerID uuid.UUID) (*tkpin.Pin, error) {
	uri := fmt.Sprintf("/v1/pins/%s", pinID.String())

	m, err := json.Marshal(tmrequest.V1DataPinsIDDelete{
		CustomerID: customerID.String(),
		OwnerType:  ownerType,
		OwnerID:    ownerID.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal the request")
	}

	tmp, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodDelete, "talk/pins", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tkpin.Pin
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
