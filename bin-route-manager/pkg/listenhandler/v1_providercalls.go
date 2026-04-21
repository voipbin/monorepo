package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/listenhandler/models/request"
)

// v1ProviderCallsPost handles POST /v1/providercalls — create a providercall record.
// The caller (bin-api-manager) has already created the underlying calls; this
// endpoint persists the audit record that correlates the request to those call IDs.
func (h *listenHandler) v1ProviderCallsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ProviderCallsPost",
	})
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataProviderCallsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.providerCallHandler.Create(
		ctx,
		req.CustomerID,
		req.ProviderID,
		req.FlowID,
		req.Source,
		req.Destinations,
		req.Anonymous,
		req.CallIDs,
		req.GroupcallIDs,
	)
	if err != nil {
		log.Errorf("Could not create a new providercall. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ProviderCallsGet handles GET /v1/providercalls — list providercalls with pagination + optional filters.
//
// Supported query params:
//   - page_size, page_token — standard pagination
//   - customer_id (uuid) — narrow to one customer
//   - provider_id (uuid) — narrow to one provider
//
// Soft-deleted records are excluded at the dbhandler layer (tm_delete IS NULL).
func (h *listenHandler) v1ProviderCallsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ProviderCallsGet",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	q := u.Query()

	// pagination
	tmpSize, _ := strconv.Atoi(q.Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := q.Get(PageToken)

	// optional filters
	filters := map[providercall.Field]any{}
	if v := q.Get(string(providercall.FieldCustomerID)); v != "" {
		id, errParse := uuid.FromString(v)
		if errParse != nil {
			log.Errorf("Invalid customer_id filter. err: %v", errParse)
			return nil, errParse
		}
		filters[providercall.FieldCustomerID] = id
	}
	if v := q.Get(string(providercall.FieldProviderID)); v != "" {
		id, errParse := uuid.FromString(v)
		if errParse != nil {
			log.Errorf("Invalid provider_id filter. err: %v", errParse)
			return nil, errParse
		}
		filters[providercall.FieldProviderID] = id
	}

	tmp, err := h.providerCallHandler.List(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get providercalls. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ProviderCallsIDGet handles GET /v1/providercalls/{id} — fetch a single record.
func (h *listenHandler) v1ProviderCallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ProviderCallsIDGet",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/providercalls/{id}"
	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.providerCallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get providercall info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ProviderCallsIDDelete handles DELETE /v1/providercalls/{id} — soft-delete the record.
// Returns the deleted record (ProviderCall.WebhookMessage shape via the marshalled ProviderCall).
func (h *listenHandler) v1ProviderCallsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ProviderCallsIDDelete",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log = log.WithField("providercall_id", id)

	tmp, err := h.providerCallHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the providercall. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
