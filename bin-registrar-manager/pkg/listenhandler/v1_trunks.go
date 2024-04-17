package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-registrar-manager/pkg/listenhandler/models/request"
)

// processV1TrunksPost handles /v1/trunks request
func (h *listenHandler) processV1TrunksPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1TrunksPost",
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var req request.V1DataTrunksPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the request data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.trunkHandler.Create(ctx, req.CustomerID, req.Name, req.Detail, req.DomainName, req.Authtypes, req.Username, req.Password, req.AllowedIPs)
	if err != nil {
		log.Errorf("Could not create a new trunk correctly. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TrunksGet handles /v1/trunks GET request
func (h *listenHandler) processV1TrunksGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1TrunksGet",
	})

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

	// parse the filters
	filters := h.utilHandler.URLParseFilters(u)
	if customerID != uuid.Nil {
		filters["customer_id"] = customerID.String()
	}

	trunks, err := h.trunkHandler.Gets(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get trunks. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(trunks)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TrunksIDGet handles /v1/trunks/{id} GET request
func (h *listenHandler) processV1TrunksIDGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1TrunksIDGet",
	})

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/trunks/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.trunkHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get trunk info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TrunksIDPut handles /v1/trunks/{id} PUT request
func (h *listenHandler) processV1TrunksIDPut(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1TrunksIDPut",
	})

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/trunks/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var reqData request.V1DataTrunksIDPut
	if err := json.Unmarshal([]byte(req.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the request data. data: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.trunkHandler.Update(ctx, id, reqData.Name, reqData.Detail, reqData.Authtypes, reqData.Username, reqData.Password, reqData.AllowedIPs)
	if err != nil {
		log.Errorf("Could not get domain info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TrunksIDDelete handles /v1/trunks/{id} DELETE request
func (h *listenHandler) processV1TrunksIDDelete(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1TrunksIDDelete",
	})

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/trunks/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.trunkHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not get trunk info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TrunksDomainNameDomainNameGet handles /v1/trunks/domain_name/<domain-name> GET request
func (h *listenHandler) processV1TrunksDomainNameDomainNameGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1TrunksDomainNameDomainNameGet",
	})

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	domainName := tmpVals[4]

	tmp, err := h.trunkHandler.GetByDomainName(ctx, domainName)
	if err != nil {
		log.Errorf("Could not get domains. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
