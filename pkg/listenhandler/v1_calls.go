package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// processV1CallsIDGet handles GET /v1/calls/<id> request
func (h *listenHandler) processV1CallsIDGet(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	c, err := h.db.CallGet(ctx, id)
	if err != nil {
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDPost handles POST /v1/calls/<id> request
// It creates a new call.
func (h *listenHandler) processV1CallsIDPost(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})

	var reqData request.V1DataCallsIDPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		// same call-id is already exsit
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	c, err := h.callHandler.CreateCallOutgoing(id, reqData.UserID, reqData.FlowID, reqData.Source, reqData.Destination)
	if err != nil {
		log.Debugf("Could not create a outgoing call. flow: %s, source: %v, destination: %v, err: %v", reqData.FlowID, reqData.Source, reqData.Destination, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", c, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CallsIDGet handles /v1/calls/<id>/health-check request
func (h *listenHandler) processV1CallsIDHealthPost(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

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
func (h *listenHandler) processV1CallsIDActionTimeoutPost(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	uriItems := strings.Split(m.URI, "/")

	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var data request.V1DataCallsIDActionTimeout
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}

	action := &action.Action{
		ID:        data.ActionID,
		Type:      data.ActionType,
		TMExecute: data.TMExecute,
	}

	if err := h.callHandler.ActionTimeout(id, action); err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
	}

	return res, nil
}

// processV1CallsIDGet handles /v1/calls/<id>/action-next request
func (h *listenHandler) processV1CallsIDActionNextPost(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	var a action.Action
	if err := json.Unmarshal([]byte(m.Data), &a); err != nil {
		return simpleResponse(404), nil
	}

	c, err := h.db.CallGet(context.Background(), id)
	if err != nil {
		return simpleResponse(404), nil
	}

	if err := h.callHandler.ActionNext(c); err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
	}

	return res, nil
}
