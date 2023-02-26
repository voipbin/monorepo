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

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/listenhandler/models/request"
)

// processV1ExtensionsPost handles /v1/extensions POST request
func (h *listenHandler) processV1ExtensionsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExtensionsPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var req request.V1DataExtensionsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the request data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.extensionHandler.Create(
		ctx,
		req.CustomerID,
		req.Name,
		req.Detail,
		req.DomainID,
		req.Extension,
		req.Password,
	)
	if err != nil {
		log.Errorf("Could not create a new extension correctly. err: %v", err)
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

// processV1ExtensionsIDGet handles /v1/extensions/{id} GET request
func (h *listenHandler) processV1ExtensionsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExtensionsIDGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/extensions/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	extensionID := uuid.FromStringOrNil(tmpVals[3])

	domain, err := h.extensionHandler.Get(ctx, extensionID)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(domain)
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

// processV1ExtensionsIDPut handles /v1/extensions/{id} PUT request
func (h *listenHandler) processV1ExtensionsIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExtensionsIDPut",
		"Request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/extensions/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	extID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataExtensionsIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the request data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// create a update domain info
	tmpExt := &extension.Extension{
		ID:       extID,
		Name:     req.Name,
		Detail:   req.Detail,
		Password: req.Password,
	}

	tmp, err := h.extensionHandler.Update(ctx, tmpExt)
	if err != nil {
		log.Errorf("Could not update the extension info. err: %v", err)
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

// processV1ExtensionsIDDelete handles /v1/extensions/{id} DELETE request
func (h *listenHandler) processV1ExtensionsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExtensionsIDDelete",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/extensions/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	extID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.extensionHandler.Delete(ctx, extID)
	if err != nil {
		log.Errorf("Could not delete the extension info. err: %v", err)
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

// processV1ExtensionsGet handles /v1/extension GET request
func (h *listenHandler) processV1ExtensionsGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExtensionsGet",
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

	// get domain_id
	domainID := uuid.FromStringOrNil(u.Query().Get("domain_id"))

	resExts, err := h.extensionHandler.GetsByDomainID(ctx, domainID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get extensions. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resExts)
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

// processV1ExtensionsExtensionEndpointGet handles /v1/extensions/endpoint/{endpoint} GET request
func (h *listenHandler) processV1ExtensionsExtensionEndpointGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExtensionsExtensionEndpointGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/extensions/endpoint/test_ext@test_domain"
	tmpVals := strings.Split(u.Path, "/")
	endpoint := tmpVals[4]

	tmp, err := h.extensionHandler.GetByEndpoint(ctx, endpoint)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
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
