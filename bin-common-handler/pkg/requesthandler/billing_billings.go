package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-common-handler/models/sock"

	"github.com/pkg/errors"
)

// BillingV1BillingGets returns list of billings.
func (r *requestHandler) BillingV1BillingGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]bmbilling.Billing, error) {
	uri := fmt.Sprintf("/v1/billings?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodGet, "billing/billings", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("could not get response")
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []bmbilling.Billing
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal the response data")
	}

	return res, nil
}
