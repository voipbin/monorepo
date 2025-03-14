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
)

// EmailV1EmailGets sends a request to email-manager
// to getting a list of emails info.
// it returns detail email info if it succeed.
func (r *requestHandler) EmailV1EmailGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]ememail.Email, error) {
	uri := fmt.Sprintf("/v1/emails?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	res, err := r.sendRequestEmail(ctx, uri, sock.RequestMethodGet, "email/emails", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData []ememail.Email
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return resData, nil
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

	res, err := r.sendRequestEmail(ctx, uri, sock.RequestMethodPost, "email/emails", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData ememail.Email
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// EmailV1EmailGet sends a request to email-manager
// to getting a detail email info.
// it returns detail email info if it succeed.
func (r *requestHandler) EmailV1EmailGet(ctx context.Context, emailID uuid.UUID) (*ememail.Email, error) {
	uri := fmt.Sprintf("/v1/emails/%s", emailID)

	res, err := r.sendRequestEmail(ctx, uri, sock.RequestMethodGet, "email/emails/<email-id>", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData ememail.Email
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// EmailV1EmailDelete sends the request to delete the email
func (r *requestHandler) EmailV1EmailDelete(ctx context.Context, id uuid.UUID) (*ememail.Email, error) {
	uri := fmt.Sprintf("/v1/emails/%s", id)

	res, err := r.sendRequestEmail(ctx, uri, sock.RequestMethodDelete, "email/emails/<email-id>", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData ememail.Email
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}
