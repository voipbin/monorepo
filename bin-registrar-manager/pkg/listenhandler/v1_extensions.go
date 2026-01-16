package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/listenhandler/models/request"
)

// processV1ExtensionsPost handles /v1/extensions POST request
func (h *listenHandler) processV1ExtensionsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ExtensionsIDGet handles /v1/extensions/{id} GET request
func (h *listenHandler) processV1ExtensionsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ExtensionsIDPut handles /v1/extensions/{id} PUT request
func (h *listenHandler) processV1ExtensionsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
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
	extensionID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataExtensionsIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the request data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	fields := map[extension.Field]any{
		extension.FieldName:     req.Name,
		extension.FieldDetail:   req.Detail,
		extension.FieldPassword: req.Password,
	}

	tmp, err := h.extensionHandler.Update(ctx, extensionID, fields)
	if err != nil {
		log.Errorf("Could not update the extension info. err: %v", err)
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

// processV1ExtensionsIDDelete handles /v1/extensions/{id} DELETE request
func (h *listenHandler) processV1ExtensionsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ExtensionsGet handles /v1/extension GET request
func (h *listenHandler) processV1ExtensionsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
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

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[extension.FieldStruct, extension.Field](extension.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	extensions, err := h.extensionHandler.List(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Errorf("Could not get extensions. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(extensions)
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

// processV1ExtensionsExtensionExtensionGet handles /v1/extensions/extension/{extension} GET request
func (h *listenHandler) processV1ExtensionsExtensionExtensionGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ExtensionsExtensionExtensionGet",
		"request": m,
	})

	// Parse filters from request body
	var reqData request.V1DataExtensionsExtensionExtensionGet
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &reqData); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return nil, err
		}
	}

	// "/v1/extensions/extension/test_ext"
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}
	tmpVals := strings.Split(u.Path, "/")
	ext := tmpVals[4]

	log.WithFields(logrus.Fields{
		"customer_id":      reqData.CustomerID,
		"extension":        ext,
		"filters_raw_data": string(m.Data),
	}).Debug("processV1ExtensionsExtensionExtensionGet: Parsed filters from request body")

	tmp, err := h.extensionHandler.GetByExtension(ctx, reqData.CustomerID, ext)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
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
