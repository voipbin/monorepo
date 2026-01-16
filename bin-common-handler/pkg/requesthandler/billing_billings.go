package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// BillingV1BillingGets returns list of billings.
func (r *requestHandler) BillingV1BillingList(ctx context.Context, pageToken string, pageSize uint64, filters map[bmbilling.Field]any) ([]bmbilling.Billing, error) {
	uri := fmt.Sprintf("/v1/billings?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodGet, "billing/billings", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []bmbilling.Billing
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// BillingV1BillingGet returns a single billing record by ID.
func (r *requestHandler) BillingV1BillingGet(ctx context.Context, billingID uuid.UUID) (*bmbilling.Billing, error) {
	uri := fmt.Sprintf("/v1/billings/%s", billingID.String())

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodGet, "billing/billing", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res bmbilling.Billing
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
