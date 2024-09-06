package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-outdial-manager/pkg/listenhandler/models/request"
)

// v1OutdialtargetsIDGet handles /v1/outdialtargets/<outdialtarget-id> GET request
func (h *listenHandler) v1OutdialtargetsIDGet(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialtargetsIDGet",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialtargetsIDGet.")

	tmp, err := h.outdialTargetHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not delete outdialtarget. err: %v", err)
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

// v1OutdialtargetsIDDelete handles /v1/outdialtargets/<outdialtarget-id> DELETE request
func (h *listenHandler) v1OutdialtargetsIDDelete(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialtargetsIDDelete",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialtargetsIDDelete.")

	tmp, err := h.outdialTargetHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete outdialtarget. err: %v", err)
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

// v1OutdialtargetsIDProgressingPost handles /v1/outdialtargets/<outdialtarget-id>/progressing POST request
func (h *listenHandler) v1OutdialtargetsIDProgressingPost(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialtargetsIDPut",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialtargetsIDPut.")

	var req request.V1DataOutdialtargetsIDProgressingPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.outdialTargetHandler.UpdateProgressing(ctx, id, req.DestinationIndex)
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

// v1OutdialtargetsIDStatusPut handles /v1/outdialtargets/<outdialtarget-id>/status PUT request
func (h *listenHandler) v1OutdialtargetsIDStatusPut(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":       "v1OutdialtargetsIDStatusPut",
			"outdial_id": id,
		},
	)
	log.WithField("request", m).Debug("Executing v1OutdialtargetsIDPut.")

	var req request.V1DataOutdialtargetsIDStatusPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.outdialTargetHandler.UpdateStatus(ctx, id, req.Status)
	if err != nil {
		log.Errorf("Could not update outdialtarget status. err: %v", err)
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
