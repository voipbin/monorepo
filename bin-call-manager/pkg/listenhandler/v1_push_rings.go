package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
)

// processV1PushRingsPost handles /v1/push-rings POST request
func (h *listenHandler) processV1PushRingsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1PushRingsPost",
	})

	var req request.V1DataPushRingsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the request. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.pushRingHandler.Start(ctx, req.CustomerID, req.Destination, req.ConfbridgeID, req.MasterCallID, req.FlowID, req.Source)
	if err != nil {
		log.Errorf("Could not start the push ring session. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1PushRingsIDExpirePost handles /v1/push-rings/<push-ring-id>/expire POST request
func (h *listenHandler) processV1PushRingsIDExpirePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1PushRingsIDExpirePost",
	})

	id, ok := pushRingIDFromURI(m.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	if err := h.pushRingHandler.Expire(ctx, id); err != nil {
		log.Errorf("Could not expire the push ring session. err: %v", err)
		return errorResponse(err), nil
	}

	return simpleResponse(200), nil
}

// processV1PushRingsIDClosePost handles /v1/push-rings/<push-ring-id>/close POST request
func (h *listenHandler) processV1PushRingsIDClosePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1PushRingsIDClosePost",
	})

	id, ok := pushRingIDFromURI(m.URI)
	if !ok {
		return simpleResponse(400), nil
	}

	if err := h.pushRingHandler.Close(ctx, id); err != nil {
		log.Errorf("Could not close the push ring session. err: %v", err)
		return errorResponse(err), nil
	}

	return simpleResponse(200), nil
}

// pushRingIDFromURI returns the session id of /v1/push-rings/<id>/<action>.
func pushRingIDFromURI(uri string) (uuid.UUID, bool) {
	items := strings.Split(uri, "/")
	if len(items) < 5 {
		return uuid.Nil, false
	}

	id := uuid.FromStringOrNil(items[3])
	return id, id != uuid.Nil
}
