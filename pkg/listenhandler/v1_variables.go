package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/listenhandler/models/request"
)

// v1VariablesIDGet handles /v1/variables/{id} GET request
func (h *listenHandler) v1VariablesIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1VariablesIDGet",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/variables/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	variableID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.variableHandler.Get(ctx, variableID)
	if err != nil {
		log.Errorf("Could not get variable info. err: %v", err)
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

// v1VariablesIDVariablesPost handles /v1/variables/{id}/variables POST request
func (h *listenHandler) v1VariablesIDVariablesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1VariablesIDVariablesPost",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/variables/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/variables"
	tmpVals := strings.Split(u.Path, "/")
	variableID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataVariablesIDVariablesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.variableHandler.SetVariable(ctx, variableID, req.Key, req.Value)
	if err != nil {
		log.Errorf("Could not set variable info. key: %s, value: %s, err: %v", req.Key, req.Value, err)
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
