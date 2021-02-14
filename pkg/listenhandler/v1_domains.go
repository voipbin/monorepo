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
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
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
	tmpDomain := &models.Domain{
		UserID:     reqData.UserID,
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
	tmpUserID, _ := strconv.Atoi(u.Query().Get("user_id"))
	userID := uint64(tmpUserID)

	resDomains, err := h.domainHandler.DomainGetsByUserID(ctx, userID, pageToken, pageSize)
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
