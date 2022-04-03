package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/outdial-manager.git/pkg/listenhandler/models/request"
)

// v1OutdialsPost handles /v1/outdials POST request
func (h *listenHandler) v1OutdialsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1OutdialsPost",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataOutdialsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create
	tmp, err := h.outdialHandler.Create(
		ctx,
		req.CustomerID,
		req.CampaignID,
		req.Name,
		req.Detail,
		req.Data,
	)
	if err != nil {
		log.Errorf("Could not create outdial. err: %v", err)
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

// v1OutdialsGet handles /v1/outdials GET request
func (h *listenHandler) v1OutdialsGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1OutdialsGet",
		},
	)
	log.WithField("request", req).Debug("Received request.")

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get customer_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

	tmp, err := h.outdialHandler.GetsByCustomerID(ctx, customerID, pageToken, pageSize)
	if err != nil {
		logrus.Errorf("Could not get outdials. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
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

// v1OutdialsIDGet handles /v1/outdials/<outdial-id> GET request
func (h *listenHandler) v1OutdialsIDGet(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1OutdialsIDGet",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	outdialID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.outdialHandler.Get(ctx, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial. err: %v", err)
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

// v1OutdialsIDAvailableGet handles /v1/outdials/<outdial-id>/available GET request
func (h *listenHandler) v1OutdialsIDAvailableGet(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1OutdialsIDAvailableGet",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	outdialID := uuid.FromStringOrNil(tmpVals[3])

	// parse the params
	tryCount0, _ := strconv.Atoi(u.Query().Get("try_count_0"))
	tryCount1, _ := strconv.Atoi(u.Query().Get("try_count_1"))
	tryCount2, _ := strconv.Atoi(u.Query().Get("try_count_2"))
	tryCount3, _ := strconv.Atoi(u.Query().Get("try_count_3"))
	tryCount4, _ := strconv.Atoi(u.Query().Get("try_count_4"))
	interval, _ := strconv.Atoi(u.Query().Get("interval"))
	tmpLimit, _ := strconv.Atoi(u.Query().Get("limit"))
	limit := uint64(tmpLimit)

	tmp, err := h.outdialTargetHandler.GetAvailable(ctx, outdialID, tryCount0, tryCount1, tryCount2, tryCount3, tryCount4, time.Millisecond*time.Duration(interval), limit)
	if err != nil {
		log.Errorf("Could not get available targets. err: %v", err)
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

// v1OutdialsIDTargetsPost handles /v1/outdials/<outdial-id>/targets POST request
func (h *listenHandler) v1OutdialsIDTargetsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1OutdialsIDTargetsPost",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	outdialID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataOutdialsIDTargetsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create
	tmp, err := h.outdialTargetHandler.Create(
		ctx,
		outdialID,
		req.Name,
		req.Detail,
		req.Data,
		req.Destination0,
		req.Destination1,
		req.Destination2,
		req.Destination3,
		req.Destination4,
	)
	if err != nil {
		log.Errorf("Could not create outdialtarget. err: %v", err)
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

// v1OutdialsIDTargetsGet handles /v1/outdials/<outdial-id>/targets GET request
func (h *listenHandler) v1OutdialsIDTargetsGet(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1OutdialsIDTargetsGet",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	outdialID := uuid.FromStringOrNil(tmpVals[3])

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	tmp, err := h.outdialTargetHandler.GetsByOutdialID(ctx, outdialID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get outdialtargets. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
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
