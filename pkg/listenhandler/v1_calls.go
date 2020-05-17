package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// processV1CallsIDGet handles /v1/calls/<id> request
func (h *listenHandler) processV1CallsIDGet(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return response404, nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	c, err := h.db.CallGet(ctx, id)
	if err != nil {
		return response404, nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return response404, nil
	}

	res := &rabbitmq.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       string(data),
	}

	return res, nil
}

// processV1CallsIDGet handles /v1/calls/<id>/health-check request
func (h *listenHandler) processV1CallsIDHealthPost(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return nil, fmt.Errorf("wrong uri")
	}

	id := uuid.FromStringOrNil(uriItems[3])
	type Data struct {
		RetryCount int `json:"retry_count"`
		Delay      int `json:"delay"`
	}

	var data Data
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
