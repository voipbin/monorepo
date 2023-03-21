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

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/listenhandler/models/request"
)

// v1OutplansPost handles /v1/outplans POST request
func (h *listenHandler) v1OutplansPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1OutplansPost",
		"request": m,
	})
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataOutplansPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create a new outplan
	tmp, err := h.outplanHandler.Create(
		ctx,
		req.CustomerID,
		req.Name,
		req.Detail,
		req.Source,
		req.DialTimeout,
		req.TryInterval,
		req.MaxTryCount0,
		req.MaxTryCount1,
		req.MaxTryCount2,
		req.MaxTryCount3,
		req.MaxTryCount4,
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

// v1OutplansGet handles /v1/outplans GET request
func (h *listenHandler) v1OutplansGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1OutplansGet",
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

	// get customer_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.outplanHandler.GetsByCustomerID(ctx, customerID, pageToken, pageSize)
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

// v1OutplansIDGet handles /v1/outplans/{id} GET request
func (h *listenHandler) v1OutplansIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1OutplansIDGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.outplanHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan info. err: %v", err)
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

// v1OutplansIDDelete handles /v1/outplans/{id} Delete request
func (h *listenHandler) v1OutplansIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1OutplansIDDelete",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.outplanHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete outplan info. err: %v", err)
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

// v1OutplansIDPut handles /v1/outplans/<outplan_id> PUT request
func (h *listenHandler) v1OutplansIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1OutplansIDPut",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataOutplansIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create a new outplan
	tmp, err := h.outplanHandler.UpdateBasicInfo(ctx, id, req.Name, req.Detail)
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

// v1OutplansIDPut handles /v1/outplans/<outplan_id>/dials PUT request
func (h *listenHandler) v1OutplansIDDialsPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1OutplansIDPut",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataOutplansIDDialsPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create a new outplan
	tmp, err := h.outplanHandler.UpdateDialInfo(ctx, id, req.Source, req.DialTimeout, req.TryInterval, req.MaxTryCount0, req.MaxTryCount1, req.MaxTryCount2, req.MaxTryCount3, req.MaxTryCount4)
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
