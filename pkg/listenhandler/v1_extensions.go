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

// processV1ExtensionsPost handles /v1/extensions request
func (h *listenHandler) processV1ExtensionsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var reqData request.V1DataExtensionsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the request data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// parameter validatoin
	if reqData.Extension == "" {
		logrus.Errorf("Empty extension create is not allowed. data: %v", m.Data)
		return simpleResponse(400), nil
	}

	// create a new extension
	e := &models.Extension{
		UserID: reqData.UserID,

		Name:     reqData.Name,
		Detail:   reqData.Detail,
		DomainID: reqData.DomainID,

		Extension: reqData.Extension,
		Password:  reqData.Password,
	}
	ext, err := h.extensionHandler.ExtensionCreate(ctx, e)
	if err != nil {
		logrus.Errorf("Could not create a new extension correctly. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(ext)
	if err != nil {
		logrus.Errorf("Could not marshal the response message. message: %v, err: %v", ext, err)
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
func (h *listenHandler) processV1ExtensionsIDGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/extensions/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	extensionID := uuid.FromStringOrNil(tmpVals[3])

	domain, err := h.extensionHandler.ExtensionGet(ctx, extensionID)
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

// processV1ExtensionsIDPut handles /v1/extensions/{id} PUT request
func (h *listenHandler) processV1ExtensionsIDPut(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/extensions/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	extID := uuid.FromStringOrNil(tmpVals[3])

	var reqData request.V1DataExtensionsIDPut
	if err := json.Unmarshal([]byte(req.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the request data. data: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}

	// create a update domain info
	tmpExt := &models.Extension{
		ID:       extID,
		Name:     reqData.Name,
		Detail:   reqData.Detail,
		Password: reqData.Password,
	}

	domain, err := h.extensionHandler.ExtensionUpdate(ctx, tmpExt)
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

// processV1ExtensionsIDDelete handles /v1/extensions/{id} DELETE request
func (h *listenHandler) processV1ExtensionsIDDelete(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/extensions/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	extID := uuid.FromStringOrNil(tmpVals[3])

	if err := h.extensionHandler.ExtensionDelete(ctx, extID); err != nil {
		logrus.Errorf("Could not get domain info. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1ExtensionsGet handles /v1/extension GET request
func (h *listenHandler) processV1ExtensionsGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get domain_id
	domainID := uuid.FromStringOrNil(u.Query().Get("domain_id"))

	resExts, err := h.extensionHandler.ExtensionGetsByDomainID(ctx, domainID, pageToken, pageSize)
	if err != nil {
		logrus.Errorf("Could not get extensions. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(resExts)
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
