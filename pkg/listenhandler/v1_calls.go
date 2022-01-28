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
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
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

// processV1CallsIDGet handles GET /v1/calls/<id> request
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

	// generate call id.
	id := uuid.Must(uuid.NewV4())
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsPost.")

	var reqData request.V1DataCallsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		// same call-id is already exsit
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"user":        reqData.CustomerID,
		"flow":        reqData.FlowID,
		"source":      reqData.Source,
		"destination": reqData.Destination,
	})

	log.Debug("Creating outgoing call.")
	c, err := h.callHandler.CreateCallOutgoing(ctx, id, reqData.CustomerID, reqData.FlowID, reqData.Source, reqData.Destination)
	if err != nil {
		log.Debugf("Could not create a outgoing call. err: %v", err)
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

// processV1CallsIDPost handles POST /v1/calls/<id> request
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

	var reqData request.V1DataCallsIDPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		// same call-id is already exsit
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"user":        reqData.CustomerID,
		"flow":        reqData.FlowID,
		"source":      reqData.Source,
		"destination": reqData.Destination,
	})

	log.Debug("Creating outgoing call.")
	c, err := h.callHandler.CreateCallOutgoing(ctx, id, reqData.CustomerID, reqData.FlowID, reqData.Source, reqData.Destination)
	if err != nil {
		log.Debugf("Could not create a outgoing call. flow: %s, source: %v, destination: %v, err: %v", reqData.FlowID, reqData.Source, reqData.Destination, err)
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

// processV1CallsIDDelete handles Delete /v1/calls/<id> request
// It hangs up the exsited call.
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

	// hanging up the call
	if err := h.callHandler.HangingUp(ctx, id, ari.ChannelCauseNormalClearing); err != nil {
		log.Debugf("Could not hanging up the call. err: %v", err)
		return simpleResponse(500), nil
	}

	// get updated call info
	resCall, err := h.callHandler.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get updated call info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(resCall)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", resCall, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDGet handles /v1/calls/<id>/health-check request
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

// processV1CallsIDGet handles /v1/calls/<id>/action-timeout request
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

	action := &action.Action{
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

// processV1CallsIDGet handles /v1/calls/<id>/action-next request
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

// processV1CallsIDChainedCallIDsPost handles /v1/calls/<id>/chained-call-ids POST request
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

	var data request.V1DataCallsIDChainedCallIDsPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"chained_call_ids": data,
	}).Debugf("Parsed request data.")

	if err := h.callHandler.ChainedCallIDAdd(ctx, id, data.ChainedCallID); err != nil {
		log.Errorf("Could not add the chained call id. call: %s, chained_call: %s, err: %v", id, data.ChainedCallID, err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDChainedCallIDsDelete handles /v1/calls/<id>/chained-call-ids/<chained-call-id> DELETE request
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

	if err := h.callHandler.ChainedCallIDRemove(ctx, id, chainedCallID); err != nil {
		log.Errorf("Could not add the chained call id. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDExternalMediaPost handles /v1/calls/<id>/external-media POST request
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

	var data request.V1DataCallsIDExternalMediaPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"external_media": data,
	}).Debugf("Parsed request data.")

	extCh, err := h.callHandler.ExternalMediaStart(ctx, id, false, data.ExternalHost, data.Encapsulation, data.Transport, data.ConnectionType, data.Format, data.Direction)
	if err != nil {
		log.Errorf("Could not start the external media. call: %s, err: %v", id, err)
		return nil, err
	}
	log.Debugf("Created external media channel. external: %v", extCh)

	ip := extCh.Data[callhandler.ChannelValiableExternalMediaLocalAddress].(string)
	port, err := strconv.Atoi(extCh.Data[callhandler.ChannelValiableExternalMediaLocalPort].(string))
	if err != nil {
		log.Errorf("Could not get external media port. err: %v", err)
		return nil, err
	}
	resExt := &response.V1ResponseCallsIDExternalMediaPost{
		MediaAddrIP:   ip,
		MediaAddrPort: port,
	}

	resData, err := json.Marshal(resExt)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", resData, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       resData,
	}

	return res, nil
}
