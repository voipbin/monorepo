package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/pkg/listenhandler/models/request"
)

// processV1NotificationsPost handles POST /v1/notifications request
func (h *listenHandler) processV1NotificationsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "processV1NotificationsPost")

	var req request.V1DataNotificationsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal the request. err: %v", err)
		return simpleResponse(400), nil
	}

	// the send errors are typed (400) or internal: both flow through the dispatcher
	res, err := h.notificationHandler.Send(ctx, req.CustomerID, req.AgentID, req.DeviceID, req.Kind, req.Data)
	if err != nil {
		return nil, err
	}

	return jsonResponse(res), nil
}
