package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/sock"
	smaccount "monorepo/bin-storage-manager/models/account"
	smrequest "monorepo/bin-storage-manager/pkg/listenhandler/models/request"
	"net/url"

	"github.com/gofrs/uuid"
)

// StorageV1AccountCreate sends a request to storage-manager
// to creating an account.
// it returns created account if it succeed.
func (r *requestHandler) StorageV1AccountCreate(ctx context.Context, customerID uuid.UUID) (*smaccount.Account, error) {
	uri := "/v1/accounts"

	data := &smrequest.V1DataAccountsPost{
		CustomerID: customerID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodPost, "storage/accounts", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res smaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// StorageV1AccountGets sends a request to storage-manager
// to getting a list of accounts.
// it returns file list of accounts if it succeed.
func (r *requestHandler) StorageV1AccountGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]smaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodGet, "storage/accounts", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []smaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// StorageV1AccountGet sends a request to storage-manager
// to getting a account info.
// it returns account info if it succeed.
func (r *requestHandler) StorageV1AccountGet(ctx context.Context, accountID uuid.UUID) (*smaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodGet, "storage/accounts/<account-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res smaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// StorageV1AccountDelete sends a request to storage-manager
// to deleting a accounts.
// it returns error if it fails
func (r *requestHandler) StorageV1AccountDelete(ctx context.Context, fileID uuid.UUID, requestTimeout int) (*smaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", fileID)

	tmp, err := r.sendRequestStorage(ctx, uri, sock.RequestMethodDelete, "storage/accounts/<account-id>", requestTimeout, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res smaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
