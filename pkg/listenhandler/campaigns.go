package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// v1CampaignsPost handles /v1/campaigns POST request
func (h *listenHandler) v1CampaignsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1CampaignsPost",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataCampaignsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create a new campaign
	tmp, err := h.campaignHandler.Create(
		ctx,
		req.CustomerID,
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1CampaignsGet handles /v1/campaigns GET request
func (h *listenHandler) v1CampaignsGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get customer_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "v1CampaignsGet",
			"customer_id": customerID,
		},
	)
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.campaignHandler.GetsByCustomerID(ctx, customerID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get campaigns. err: %v", err)
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

// v1CampaignsIDGet handles /v1/campaigns/{id} GET request
func (h *listenHandler) v1CampaignsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "v1CampaignsIDGet",
			"campaign_id": id,
		},
	)
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1CampaignsIDDelete handles /v1/campaigns/{id} DELETE request
func (h *listenHandler) v1CampaignsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "v1CampaignsIDDelete",
			"campaign_id": id,
		},
	)
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CampaignsIDExecutePost handles /v1/campaigns/{id}/execute POST request
func (h *listenHandler) v1CampaignsIDExecutePost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1CampaignsIDExecutePost",
			"campaign_id": id,
		})
	log.Debug("Executing processV1CampaignsIDRunPost.")

	// execute
	h.campaignHandler.Execute(ctx, id)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1CampaignsIDStatusPut handles /v1/campaigns/{id}/status PUT request
func (h *listenHandler) v1CampaignsIDStatusPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1CampaignsIDStatusPut",
			"campaign_id": id,
		})
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CampaignsIDServiceLevelPut handles /v1/campaigns/{id}/service_level PUT request
func (h *listenHandler) v1CampaignsIDServiceLevelPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1CampaignsIDServiceLevelPut",
			"campaign_id": id,
		})
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1CampaignsIDActionsPut handles /v1/campaigns/{id}/actions PUT request
func (h *listenHandler) v1CampaignsIDActionsPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "v1CampaignsIDActionsPut",
			"campaign_id": id,
		})
	log.Debug("Executing v1CampaignsIDActionsPut.")

	var req request.V1DataCampaignsIDActionsPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// update
	tmp, err := h.campaignHandler.UpdateActions(ctx, id, req.Actions)
	if err != nil {
		log.Errorf("Could not update the campaign service_level. err: %v", err)
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
