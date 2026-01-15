package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/listenhandler/models/request"
)

// v1CampaignsPost handles /v1/campaigns POST request
func (h *listenHandler) v1CampaignsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaignsPost",
		"request": m,
	})
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataCampaignsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create a new campaign
	tmp, err := h.campaignHandler.Create(
		ctx,
		req.ID,
		req.CustomerID,
		req.Type,
		req.Name,
		req.Detail,
		req.Actions,
		req.ServiceLevel,
		req.EndHandle,
		req.OutplanID,
		req.OutdialID,
		req.QueueID,
		req.NextCampaignID,
	)
	if err != nil {
		log.Errorf("Could not create a campaign. err: %v", err)
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

// v1CampaignsGet handles /v1/campaigns GET request
func (h *listenHandler) v1CampaignsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaignsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params from URI
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// Parse filters from request data (body)
	var filters map[string]any
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &filters); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return nil, fmt.Errorf("could not unmarshal filters: %w", err)
		}
	}

	log.WithFields(logrus.Fields{
		"filters":          filters,
		"filters_raw_data": string(m.Data),
	}).Debug("v1CampaignsGet: Parsed filters from request body")

	// Convert string map to typed field map
	typedFilters, err := campaign.ConvertStringMapToFieldMap(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, fmt.Errorf("could not convert filters: %w", err)
	}

	log.WithFields(logrus.Fields{
		"typed_filters": typedFilters,
	}).Debug("v1CampaignsGet: Converted filters to typed field map (check customer_id type)")

	tmp, err := h.campaignHandler.Gets(ctx, pageToken, pageSize, typedFilters)
	if err != nil {
		log.Errorf("Could not get campaigns. err: %v", err)
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

// v1CampaignsIDGet handles /v1/campaigns/{id} GET request
func (h *listenHandler) v1CampaignsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaignsIDGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.campaignHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info. err: %v", err)
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

// v1CampaignsIDDelete handles /v1/campaigns/{id} DELETE request
func (h *listenHandler) v1CampaignsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaignsIDDelete",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.campaignHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete campaign info. err: %v", err)
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

// v1CampaignsIDPut handles /v1/campaigns/{id}/service_level PUT request
func (h *listenHandler) v1CampaignsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaignsIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log.Debug("Executing v1CampaignsIDPut.")

	var req request.V1DataCampaignsIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// update
	tmp, err := h.campaignHandler.UpdateBasicInfo(ctx, id, req.Name, req.Detail, req.Type, req.ServiceLevel, req.EndHandle)
	if err != nil {
		log.Errorf("Could not update the campaign service_level. err: %v", err)
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

// processV1CampaignsIDExecutePost handles /v1/campaigns/{id}/execute POST request
func (h *listenHandler) v1CampaignsIDExecutePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CampaignsIDExecutePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log.Debug("Executing v1CampaignsIDExecutePost.")

	// execute
	h.campaignHandler.Execute(ctx, id)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1CampaignsIDStatusPut handles /v1/campaigns/{id}/status PUT request
func (h *listenHandler) v1CampaignsIDStatusPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CampaignsIDStatusPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log.Debug("Executing processV1CampaignsIDRunPost.")

	var req request.V1DataCampaignsIDStatusPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// update
	tmp, err := h.campaignHandler.UpdateStatus(ctx, id, req.Status)
	if err != nil {
		log.Errorf("Could not update the campaign status. err: %v", err)
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

// processV1CampaignsIDServiceLevelPut handles /v1/campaigns/{id}/service_level PUT request
func (h *listenHandler) v1CampaignsIDServiceLevelPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CampaignsIDServiceLevelPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log.Debug("Executing processV1CampaignsIDServiceLevelPut.")

	var req request.V1DataCampaignsIDServiceLevelPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// update
	tmp, err := h.campaignHandler.UpdateServiceLevel(ctx, id, req.ServiceLevel)
	if err != nil {
		log.Errorf("Could not update the campaign service_level. err: %v", err)
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

// v1CampaignsIDActionsPut handles /v1/campaigns/{id}/actions PUT request
func (h *listenHandler) v1CampaignsIDActionsPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaignsIDActionsPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log.Debug("Executing v1CampaignsIDActionsPut.")

	var req request.V1DataCampaignsIDActionsPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// update
	tmp, err := h.campaignHandler.UpdateActions(ctx, id, req.Actions)
	if err != nil {
		log.Errorf("Could not update the campaign actions. err: %v", err)
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

// v1CampaignsIDResourceInfoPut handles /v1/campaigns/{id}/resource_info PUT request
func (h *listenHandler) v1CampaignsIDResourceInfoPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaignsIDResourceInfoPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log.Debug("Executing v1CampaignsIDResourceInfoPut.")

	var req request.V1DataCampaignsIDResourceInfoPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// update
	tmp, err := h.campaignHandler.UpdateResourceInfo(ctx, id, req.OutplanID, req.OutdialID, req.QueueID, req.NextCampaignID)
	if err != nil {
		log.Errorf("Could not update the campaign resource_info. err: %v", err)
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

// v1CampaignsIDNextCampaignIDPut handles /v1/campaigns/{id}/next_campaign_id PUT request
func (h *listenHandler) v1CampaignsIDNextCampaignIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CampaignsIDNextCampaignIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log.Debug("Executing v1CampaignsIDNextCampaignIDPut.")

	var req request.V1DataCampaignsIDNextCampaignIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// update
	tmp, err := h.campaignHandler.UpdateNextCampaignID(ctx, id, req.NextCampaignID)
	if err != nil {
		log.Errorf("Could not update the campaign next_campaign_id. err: %v", err)
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
