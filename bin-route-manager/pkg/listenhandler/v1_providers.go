package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/pkg/listenhandler/models/request"
)

// v1ProvidersPost handles /v1/providers POST request
// creates a new provider with given data and return the created provider info.
func (h *listenHandler) v1ProvidersPost(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1ProvidersPost",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataProvidersPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create
	tmp, err := h.providerHandler.Create(
		ctx,
		req.Type,
		req.Hostname,
		req.TechPrefix,
		req.TechPostfix,
		req.TechHeaders,
		req.Name,
		req.Detail,
	)
	if err != nil {
		log.Errorf("Could not create a new provider. err: %v", err)
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

// v1ProvidersGet handles /v1/providers GET request
func (h *listenHandler) v1ProvidersGet(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1ProvidersGet",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	tmp, err := h.providerHandler.Gets(ctx, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get providers. err: %v", err)
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

// v1ProvidersIDGet handles /v1/providers/{id} GET request
func (h *listenHandler) v1ProvidersIDGet(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1ProvidersIDGet",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/providers/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.providerHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get provider info. err: %v", err)
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

// v1ProvidersIDPut handles /v1/providers/{id} PUT request
func (h *listenHandler) v1ProvidersIDPut(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1ProvidersIDPut",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/providers/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataProvidersIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.providerHandler.Update(
		ctx,
		id,
		req.Type,
		req.Hostname,
		req.TechPrefix,
		req.TechPostfix,
		req.TechHeaders,
		req.Name,
		req.Detail,
	)
	if err != nil {
		log.Errorf("Could not update the provider info. err: %v", err)
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

// v1ProvidersIDDelete handles /v1/providers/{id} Delete request
func (h *listenHandler) v1ProvidersIDDelete(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1ProvidersIDDelete",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log = log.WithField("provier_id", id)

	tmp, err := h.providerHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the provider. err: %v", err)
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
