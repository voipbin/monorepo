package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	cvaccount "monorepo/bin-conversation-manager/models/account"
	cvrequest "monorepo/bin-conversation-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// ConversationV1AccountGet sends a request to conversation-manager
// to gets the account.
// it returns nil if it succeed.
func (r *requestHandler) ConversationV1AccountGet(ctx context.Context, accountID uuid.UUID) (*cvaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/accounts/<account-id>", 30000, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not send the message")
	}

	var res cvaccount.Account
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConversationV1AccountGets sends a request to conversation-manager
// to getting a list of account info.
// it returns detail list of conversation info if it succeed.
func (r *requestHandler) ConversationV1AccountGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cvaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/accounts", 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cvaccount.Account
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ConversationV1AccountCreate sends a request to conversation-manager
// to create a account info.
// it returns created account info if it succeed.
func (r *requestHandler) ConversationV1AccountCreate(ctx context.Context, customerID uuid.UUID, accountType cvaccount.Type, name string, detail string, secret string, token string) (*cvaccount.Account, error) {
	uri := "/v1/accounts"

	data := &cvrequest.V1DataAccountsPost{
		CustomerID: customerID,
		Type:       accountType,
		Name:       name,
		Detail:     detail,
		Secret:     secret,
		Token:      token,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPost, "conversation/accounts", 30000, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cvaccount.Account
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConversationV1AccountUpdate sends a request to conversation-manager
// to update the account info.
// it returns update account info if it succeed.
func (r *requestHandler) ConversationV1AccountUpdate(ctx context.Context, accountID uuid.UUID, name string, detail string, secret string, token string) (*cvaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	data := &cvrequest.V1DataAccountsIDPut{
		Name:   name,
		Detail: detail,
		Secret: secret,
		Token:  token,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPut, "conversation/conversations", 30000, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cvaccount.Account
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConversationV1AccountDelete sends a request to conversation-manager
// to delete the account info.
// it returns deleted account info if it succeed.
func (r *requestHandler) ConversationV1AccountDelete(ctx context.Context, accountID uuid.UUID) (*cvaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodDelete, "conversation/conversations", 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cvaccount.Account
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
