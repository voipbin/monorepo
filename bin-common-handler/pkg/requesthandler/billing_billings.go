package requesthandler

import (
	"context"
	"fmt"
	"net/url"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-common-handler/models/sock"
)

// BillingV1BillingGets returns list of billings.
func (r *requestHandler) BillingV1BillingGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]bmbilling.Billing, error) {
	uri := fmt.Sprintf("/v1/billings?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodGet, "billing/billings", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res []bmbilling.Billing
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
