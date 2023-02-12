package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/response"
)

// processV1CallsGet handles GET /v1/calls request
func (h *listenHandler) processV1CallsGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

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

	log := logrus.WithFields(logrus.Fields{
		"user":  customerID,
		"size":  pageSize,
		"token": pageToken,
	})

	calls, err := h.callHandler.Gets(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get recordings. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(calls)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", calls, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDGet handles GET /v1/calls/<call-id> request
func (h *listenHandler) processV1CallsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDGet.")

	c, err := h.callHandler.Get(ctx, id)
	if err != nil {
		return simpleResponse(404), nil
	}
	log.Debugf("Get call. call: %v", c)

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsPost handles POST /v1/calls request
// It creates a new call.
func (h *listenHandler) processV1CallsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1CallsPost",
		})
	log.WithField("request", m).Debug("Executing processV1CallsPost.")

	var req request.V1DataCallsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"user":         req.CustomerID,
		"flow":         req.FlowID,
		"source":       req.Source,
		"destinations": req.Destinations,
	})

	log.Debug("Creating outgoing call.")
	calls, err := h.callHandler.CreateCallsOutgoing(ctx, req.CustomerID, req.FlowID, req.MasterCallID, req.Source, req.Destinations, req.EarlyExecution, req.ExecuteNextMasterOnHangup)
	if err != nil {
		log.Debugf("Could not create a outgoing call. err: %v", err)
		return simpleResponse(500), nil
	}
	log.WithField("calls", calls).Debugf("Created outgoing call. count: %d", len(calls))

	data, err := json.Marshal(calls)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", calls, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDPost handles POST /v1/calls/<call-id> request
// It creates a new call.
func (h *listenHandler) processV1CallsIDPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDPost.")

	var req request.V1DataCallsIDPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"user":        req.CustomerID,
		"flow":        req.FlowID,
		"source":      req.Source,
		"destination": req.Destination,
	})

	log.Debug("Creating outgoing call.")
	c, err := h.callHandler.CreateCallOutgoing(ctx, id, req.CustomerID, req.FlowID, req.ActiveflosID, req.MasterCallID, req.Source, req.Destination, req.EarlyExecution, req.ExecuteNextMasterOnHangup)
	if err != nil {
		log.Debugf("Could not create a outgoing call. flow: %s, source: %v, destination: %v, err: %v", req.FlowID, req.Source, req.Destination, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Created outgoing call. call: %v", c)

	data, err := json.Marshal(c)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", c, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDDelete handles Post /v1/calls/<call-id> request
// It hangs up the call.
func (h *listenHandler) processV1CallsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDDelete.")

	// delete the call
	tmp, err := h.callHandler.Delete(ctx, id)
	if err != nil {
		log.Debugf("Could not delete the call. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDHangupPost handles Post /v1/calls/<call-id>/hangup request
// It hangs up the call.
func (h *listenHandler) processV1CallsIDHangupPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDHangupPost.")

	// hanging up the call
	tmp, err := h.callHandler.HangingUp(ctx, id, call.HangupReasonNormal)
	if err != nil {
		log.Debugf("Could not hanging up the call. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDGet handles /v1/calls/<call-id>/health-check request
func (h *listenHandler) processV1CallsIDHealthPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDHealthPost.")

	var data request.V1DataCallsIDHealthPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}

	h.callHandler.CallHealthCheck(ctx, id, data.RetryCount, data.Delay)
	return nil, nil
}

// processV1CallsIDGet handles /v1/calls/<call-id>/action-timeout request
func (h *listenHandler) processV1CallsIDActionTimeoutPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")

	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDActionTimeoutPost.")

	var data request.V1DataCallsIDActionTimeoutPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}
	log = log.WithFields(logrus.Fields{
		"action":     data.ActionID,
		"type":       data.ActionType,
		"tm_execute": data.TMExecute,
	})

	action := &fmaction.Action{
		ID:        data.ActionID,
		Type:      data.ActionType,
		TMExecute: data.TMExecute,
	}

	log.Debug("Executing the action timeout.")
	if err := h.callHandler.ActionTimeout(ctx, id, action); err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDGet handles /v1/calls/<call-id>/action-next request
func (h *listenHandler) processV1CallsIDActionNextPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDActionNextPost.")

	var data request.V1DataCallsIDActionNextPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}

	c, err := h.callHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info from the database. err: %v", err)
		return simpleResponse(404), nil
	}

	// we run the go runc() here.
	// because we don't want to action's running time caused the request timeout.
	go func() {
		if data.Force {
			if err := h.callHandler.ActionNextForce(ctx, c); err != nil {
				log.Errorf("Could not execute the action next force. err: %v", err)
			}
		} else {
			if err := h.callHandler.ActionNext(ctx, c); err != nil {
				log.Errorf("Could not execute the action next. err: %v", err)
			}
		}
	}()

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDChainedCallIDsPost handles /v1/calls/<call-id>/chained-call-ids POST request
func (h *listenHandler) processV1CallsIDChainedCallIDsPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDChainedCallIDsPost.")

	var req request.V1DataCallsIDChainedCallIDsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"chained_call_ids": req,
	}).Debugf("Parsed request data.")

	tmp, err := h.callHandler.ChainedCallIDAdd(ctx, id, req.ChainedCallID)
	if err != nil {
		log.Debugf("Could not get updated call info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDChainedCallIDsDelete handles /v1/calls/<call-id>/chained-call-ids/<chained-call-id> DELETE request
func (h *listenHandler) processV1CallsIDChainedCallIDsDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	chainedCallID := uuid.FromStringOrNil(uriItems[5])
	log := logrus.WithFields(
		logrus.Fields{
			"id":              id,
			"chained_call_id": chainedCallID,
		})
	log.Debug("Executing processV1CallsIDChainedCallIDsDelete.")

	tmp, err := h.callHandler.ChainedCallIDRemove(ctx, id, chainedCallID)
	if err != nil {
		log.Debugf("Could not get updated call info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDExternalMediaPost handles /v1/calls/<call-id>/external-media POST request
func (h *listenHandler) processV1CallsIDExternalMediaPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDExternalMediaPost.")

	var req request.V1DataCallsIDExternalMediaPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"external_media": req,
	}).Debugf("Parsed request data.")

	tmp, err := h.callHandler.ExternalMediaStart(ctx, id, req.ExternalHost, req.Encapsulation, req.Transport, req.ConnectionType, req.Format, req.Direction)
	if err != nil {
		log.Errorf("Could not start the external media. call: %s, err: %v", id, err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDExternalMediaDelete handles /v1/calls/<call-id>/external-media DELETE request
func (h *listenHandler) processV1CallsIDExternalMediaDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDExternalMediaDelete.")

	tmp, err := h.callHandler.ExternalMediaStop(ctx, id)
	if err != nil {
		log.Errorf("Could not stop the external media. call_id: %s, err: %v", id, err)
		return nil, err
	}
	log.Debugf("Stopped external media channel. external: %v", tmp)

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDDigitsGet handles /v1/calls/<call-id>/digits GET request
func (h *listenHandler) processV1CallsIDDigitsGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDActionNextPost.")

	digit, err := h.callHandler.DigitsGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get call's digits. err: %v", err)
		return simpleResponse(500), nil
	}

	tmp := &response.V1ResponseCallsIDDigitsGet{
		Digits: digit,
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDDigitsSet handles /v1/calls/<call-id>/digits POST request
func (h *listenHandler) processV1CallsIDDigitsSet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDActionNextPost.")

	var req request.V1DataCallsIDDigitsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"request": req,
	}).Debugf("Parsed request data.")

	if err := h.callHandler.DigitsSet(ctx, id, req.Digits); err != nil {
		log.Errorf("Could not get call's digits. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}

// processV1CallsIDRecordingIDPut handles /v1/calls/<call-id>/recording_id PUT request
func (h *listenHandler) processV1CallsIDRecordingIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDRecordingIDPut.")

	var req request.V1DataCallsIDRecordingIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"request": req,
	}).Debugf("Parsed request data.")

	tmp, err := h.callHandler.UpdateRecordingID(ctx, id, req.RecordingID)
	if err != nil {
		log.Errorf("Could not update call's recording id. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDConfbridgeIDPut handles /v1/calls/<call-id>/confbridge_id PUT request
func (h *listenHandler) processV1CallsIDConfbridgeIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDConfbridgeIDPut.")

	var req request.V1DataCallsIDConfbridgeIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"request": req,
	}).Debugf("Parsed request data.")

	tmp, err := h.callHandler.UpdateConfbridgeID(ctx, id, req.ConfbridgeID)
	if err != nil {
		log.Errorf("Could not update call's recording id. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDRecordingStartPost handles /v1/calls/<call-id>/recording_start POST request
func (h *listenHandler) processV1CallsIDRecordingStartPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDRecordingIDPut.")

	var req request.V1DataCallsIDRecordingStartPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"request": req,
	}).Debugf("Parsed request data.")

	tmp, err := h.callHandler.RecordingStart(ctx, id, req.Format, req.EndOfSilence, req.EndOfKey, req.Duration)
	if err != nil {
		log.Errorf("Could not start call recording. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDRecordingStopPost handles /v1/calls/<call-id>/recording_stop POST request
func (h *listenHandler) processV1CallsIDRecordingStopPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDRecordingStopPost.")

	tmp, err := h.callHandler.RecordingStop(ctx, id)
	if err != nil {
		log.Errorf("Could not start call recording. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDTalkPost handles /v1/calls/<call-id>/talk POST request
func (h *listenHandler) processV1CallsIDTalkPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDTalkPost.")

	var req request.V1DataCallsIDTalkPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	if errTalk := h.callHandler.Talk(ctx, id, false, req.Text, req.Gender, req.Language); errTalk != nil {
		log.Errorf("Could not talk to the call. err: %v", errTalk)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDPlayPost handles /v1/calls/<call-id>/play POST request
func (h *listenHandler) processV1CallsIDPlayPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDPlayPost.")

	var req request.V1DataCallsIDPlayPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	if errPlay := h.callHandler.Play(ctx, id, false, req.MediaURLs); errPlay != nil {
		log.Errorf("Could not play the medias. err: %v", errPlay)
		return nil, errors.Wrap(errPlay, "could not play the medias")
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDMediaStopPost handles /v1/calls/<call-id>/media_stop POST request
func (h *listenHandler) processV1CallsIDMediaStopPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"call_id": id,
		})
	log.WithField("request", m).Debug("Executing processV1CallsIDMediaStopPost.")

	if errStop := h.callHandler.MediaStop(ctx, id); errStop != nil {
		log.Errorf("Could not stop the media. err: %v", errStop)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}
