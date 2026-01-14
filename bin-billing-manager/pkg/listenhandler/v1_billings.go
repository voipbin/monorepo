package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/billing"
)

// urlFiltersToBillingFilters converts URL string filters to typed billing.Field filters
func (h *listenHandler) urlFiltersToBillingFilters(u *url.URL) map[billing.Field]any {
	filters := make(map[billing.Field]any)

	// parse all URL query parameters
	for key, values := range u.Query() {
		// skip pagination params
		if key == PageSize || key == PageToken {
			continue
		}

		if len(values) == 0 {
			continue
		}
		value := values[0]

		// map to typed fields
		switch key {
		case "customer_id":
			if id := uuid.FromStringOrNil(value); id != uuid.Nil {
				filters[billing.FieldCustomerID] = id
			}
		case "account_id":
			if id := uuid.FromStringOrNil(value); id != uuid.Nil {
				filters[billing.FieldAccountID] = id
			}
		case "reference_type":
			filters[billing.FieldReferenceType] = value
		case "reference_id":
			if id := uuid.FromStringOrNil(value); id != uuid.Nil {
				filters[billing.FieldReferenceID] = id
			}
		case "status":
			filters[billing.FieldStatus] = value
		case "deleted":
			switch value {
			case "false":
				filters[billing.FieldDeleted] = false
			case "true":
				filters[billing.FieldDeleted] = true
			}
		}
	}

	return filters
}

// processV1BillingsGet handles GET /v1/billings request
func (h *listenHandler) processV1BillingsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1BillingsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[billing.FieldStruct, billing.Field](billing.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	as, err := h.billingHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get billings info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(as)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
