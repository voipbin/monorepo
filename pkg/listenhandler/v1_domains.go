package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/listenhandler/models/request"
)

// processV1DomainsPost handles /v1/domains request
func (h *listenHandler) processV1DomainsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var reqData request.V1DataDomainsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the request data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// create a new domain
	tmpDomain := &domain.Domain{
		CustomerID: reqData.CustomerID,
		Name:       reqData.Name,
		Detail:     reqData.Detail,
		DomainName: reqData.DomainName,
	}

	d, err := h.domainHandler.DomainCreate(ctx, tmpDomain)
	if err != nil {
		logrus.Errorf("Could not create a new domain correctly. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(d)
	if err != nil {
		logrus.Errorf("Could not marshal the response message. message: %v, err: %v", d, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1DomainsIDGet handles /v1/domains/{id} GET request
func (h *listenHandler) processV1DomainsIDGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/domains/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	domainID := uuid.FromStringOrNil(tmpVals[3])

	domain, err := h.domainHandler.DomainGet(ctx, domainID)
	if err != nil {
		logrus.Errorf("Could not get domain info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(domain)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1DomainsIDPut handles /v1/domains/{id} PUT request
func (h *listenHandler) processV1DomainsIDPut(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/domains/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	domainID := uuid.FromStringOrNil(tmpVals[3])

	var reqData request.V1DataDomainsIDPut
	if err := json.Unmarshal([]byte(req.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the request data. data: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}

	// create a update domain info
	tmpDomain := &domain.Domain{
		ID:     domainID,
		Name:   reqData.Name,
		Detail: reqData.Detail,
	}

	domain, err := h.domainHandler.DomainUpdate(ctx, tmpDomain)
	if err != nil {
		logrus.Errorf("Could not get domain info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(domain)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1DomainsIDDelete handles /v1/domains/{id} DELETE request
func (h *listenHandler) processV1DomainsIDDelete(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/domains/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	domainID := uuid.FromStringOrNil(tmpVals[3])

	if err := h.domainHandler.DomainDelete(ctx, domainID); err != nil {
		logrus.Errorf("Could not get domain info. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1DomainsGet handles /v1/domains GET request
func (h *listenHandler) processV1DomainsGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get user_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

	resDomains, err := h.domainHandler.Gets(ctx, customerID, pageToken, pageSize)
	if err != nil {
		logrus.Errorf("Could not get domains. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resDomains)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
