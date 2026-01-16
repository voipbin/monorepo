package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	ememail "monorepo/bin-email-manager/models/email"
	emrequest "monorepo/bin-email-manager/pkg/listenhandler/models/request"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// EmailV1EmailGets sends a request to email-manager
// to getting a list of emails info.
// it returns detail email info if it succeed.
func (r *requestHandler) EmailV1EmailList(ctx context.Context, pageToken string, pageSize uint64, filters map[ememail.Field]any) ([]ememail.Email, error) {
	uri := fmt.Sprintf("/v1/emails?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestEmail(ctx, uri, sock.RequestMethodGet, "email/emails", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []ememail.Email
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// EmailV1EmailSend sends the request to send the email
func (r *requestHandler) EmailV1EmailSend(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	destinations []address.Address,
	subject string,
	content string,
	attachments []ememail.Attachment,
) (*ememail.Email, error) {
	uri := "/v1/emails"

	reqData := emrequest.V1DataEmailsPost{
		CustomerID:   customerID,
		ActiveflowID: activeflowID,
		Destinations: destinations,
		Subject:      subject,
		Content:      content,
		Attachments:  attachments,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestEmail(ctx, uri, sock.RequestMethodPost, "email/emails", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res ememail.Email
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// EmailV1EmailGet sends a request to email-manager
// to getting a detail email info.
// it returns detail email info if it succeed.
func (r *requestHandler) EmailV1EmailGet(ctx context.Context, emailID uuid.UUID) (*ememail.Email, error) {
	uri := fmt.Sprintf("/v1/emails/%s", emailID)

	tmp, err := r.sendRequestEmail(ctx, uri, sock.RequestMethodGet, "email/emails/<email-id>", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res ememail.Email
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// EmailV1EmailDelete sends the request to delete the email
func (r *requestHandler) EmailV1EmailDelete(ctx context.Context, id uuid.UUID) (*ememail.Email, error) {
	uri := fmt.Sprintf("/v1/emails/%s", id)

	tmp, err := r.sendRequestEmail(ctx, uri, sock.RequestMethodDelete, "email/emails/<email-id>", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res ememail.Email
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
