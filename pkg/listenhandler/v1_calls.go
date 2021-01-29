package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1CallsGet handles GET /v1/calls request
func (h *listenHandler) processV1CallsGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get user_id
	tmpUserID, _ := strconv.Atoi(u.Query().Get("user_id"))
	userID := uint64(tmpUserID)

	log := logrus.WithFields(logrus.Fields{
		"user":  userID,
		"size":  pageSize,
		"token": pageToken,
	})

	log.Debug("Getting calls.")
	recordings, err := h.db.CallGets(context.Background(), userID, pageSize, pageToken)
	if err != nil {
		log.Debugf("Could not get recordings. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(recordings)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", recordings, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDGet handles GET /v1/calls/<id> request
func (h *listenHandler) processV1CallsIDGet(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

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

	c, err := h.db.CallGet(ctx, id)
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
func (h *listenHandler) processV1CallsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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
		"user":        reqData.UserID,
		"flow":        reqData.FlowID,
		"source":      reqData.Source,
		"destination": reqData.Destination,
	})

	log.Debug("Creating outgoing call.")
	c, err := h.callHandler.CreateCallOutgoing(id, reqData.UserID, reqData.FlowID, reqData.Source, reqData.Destination)
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
func (h *listenHandler) processV1CallsIDPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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
		"user":        reqData.UserID,
		"flow":        reqData.FlowID,
		"source":      reqData.Source,
		"destination": reqData.Destination,
	})

	log.Debug("Creating outgoing call.")
	c, err := h.callHandler.CreateCallOutgoing(id, reqData.UserID, reqData.FlowID, reqData.Source, reqData.Destination)
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
func (h *listenHandler) processV1CallsIDDelete(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	// get call
	ctx := context.Background()
	c, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Debugf("Could not get call info. err: %v", err)
		return simpleResponse(404), nil
	}

	// hanging up the call
	if err := h.callHandler.HangingUp(c, ari.ChannelCauseNormalClearing); err != nil {
		log.Debugf("Could not hanging up the call. err: %v", err)
		return simpleResponse(500), nil
	}

	// get updated call info
	resCall, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Debugf("Could not get updated call info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(resCall)
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

// processV1CallsIDGet handles /v1/calls/<id>/health-check request
func (h *listenHandler) processV1CallsIDHealthPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

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

	var data request.V1DataCallsIDHealth
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}

	c, err := h.db.CallGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not find a call. call: %s", id)
	}

	// send a channel heaclth check
	_, err = h.reqHandler.AstChannelGet(c.AsteriskID, c.ChannelID)
	if err != nil {
		data.RetryCount++
	} else {
		data.RetryCount = 0
	}

	// send another health check.
	if err := h.reqHandler.CallCallHealth(id, data.Delay, data.RetryCount); err != nil {
		return nil, err
	}

	return nil, nil
}

// processV1CallsIDGet handles /v1/calls/<id>/action-timeout request
func (h *listenHandler) processV1CallsIDActionTimeoutPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	var data request.V1DataCallsIDActionTimeout
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
	if err := h.callHandler.ActionTimeout(id, action); err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDGet handles /v1/calls/<id>/action-next request
func (h *listenHandler) processV1CallsIDActionNextPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDActionNextPost.")

	c, err := h.db.CallGet(context.Background(), id)
	if err != nil {
		log.Errorf("Could not get call info from the database. err: %v", err)
		return simpleResponse(404), nil
	}

	if err := h.callHandler.ActionNext(c); err != nil {
		log.Errorf("Could not get call info from the database. err: %v", err)
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDChainedCallIDsPost handles /v1/calls/<id>/chained-call-ids POST request
func (h *listenHandler) processV1CallsIDChainedCallIDsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	var data request.V1DataCallsIDChainedCallIDs
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"chained_call_ids": data,
	}).Debugf("Parsed request data.")

	if err := h.callHandler.ChainedCallIDAdd(id, data.ChainedCallID); err != nil {
		log.Errorf("Could not add the chained call id. call: %s, chained_call: %s, err: %v", id, data.ChainedCallID, err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDChainedCallIDsDelete handles /v1/calls/<id>/chained-call-ids/<chained-call-id> DELETE request
func (h *listenHandler) processV1CallsIDChainedCallIDsDelete(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	if err := h.callHandler.ChainedCallIDRemove(id, chainedCallID); err != nil {
		log.Errorf("Could not add the chained call id. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
	}

	return res, nil
}
