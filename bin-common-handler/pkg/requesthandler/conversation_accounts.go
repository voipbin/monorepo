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
	"github.com/pkg/errors"
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

	var res cvaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConversationV1AccountGets sends a request to conversation-manager
// to getting a list of account info.
// it returns detail list of conversation info if it succeed.
func (r *requestHandler) ConversationV1AccountList(ctx context.Context, pageToken string, pageSize uint64, filters map[cvaccount.Field]any) ([]cvaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/accounts", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cvaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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
	if err != nil {
		return nil, err
	}

	var res cvaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConversationV1AccountUpdate sends a request to conversation-manager
// to update the account info.
// it returns update account info if it succeed.
func (r *requestHandler) ConversationV1AccountUpdate(ctx context.Context, accountID uuid.UUID, fields map[cvaccount.Field]any) (*cvaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	m, err := json.Marshal(fields)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPut, "conversation/conversations", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cvaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConversationV1AccountDelete sends a request to conversation-manager
// to delete the account info.
// it returns deleted account info if it succeed.
func (r *requestHandler) ConversationV1AccountDelete(ctx context.Context, accountID uuid.UUID) (*cvaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodDelete, "conversation/conversations", 30000, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cvaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
