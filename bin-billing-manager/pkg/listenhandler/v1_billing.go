package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1BillingGet handles GET /v1/billings/{billing-id} request
func (h *listenHandler) processV1BillingGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1BillingGet",
		"request": m,
	})

	// Parse URL to extract billing ID from path
	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse URI. err: %v", err)
		return simpleResponse(400), nil
	}

	// Extract billing ID from path: /v1/billings/{billing-id}
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) != 3 {
		log.Errorf("Invalid path format. path: %s", u.Path)
		return simpleResponse(400), nil
	}

	billingID := uuid.FromStringOrNil(pathParts[2])
	if billingID == uuid.Nil {
		log.Errorf("Invalid billing ID format. billing_id: %s", pathParts[2])
		return simpleResponse(400), nil
	}

	// Fetch billing from billingHandler (no authorization check)
	billing, err := h.billingHandler.Get(ctx, billingID)
	if err != nil {
		log.Errorf("Could not get billing. billing_id: %s, err: %v", billingID, err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(billing)
	if err != nil {
		log.Errorf("Could not marshal billing. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
