package requesthandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"

	pmnotification "monorepo/bin-push-manager/models/notification"
	pmrequest "monorepo/bin-push-manager/pkg/listenhandler/models/request"
)

// requestTimeoutPushNotificationSend is the timeout(ms) of the notification send request. It is above the
// per-device delivery limit of push-manager (5 seconds). A timeout means the delivery is unknown, not failed.
const requestTimeoutPushNotificationSend = 8000

// PushV1NotificationSend sends a request to push-manager to send the notification to the devices of the agent.
// A nil deviceID addresses every live device of the agent.
func (r *requestHandler) PushV1NotificationSend(ctx context.Context, customerID, agentID, deviceID uuid.UUID, kind pmnotification.Kind, data map[string]string) (*pmnotification.SendResult, error) {
	uri := "/v1/notifications"

	req := &pmrequest.V1DataNotificationsPost{
		CustomerID: customerID,
		AgentID:    agentID,
		DeviceID:   deviceID,
		Kind:       kind,
		Data:       data,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestPush(ctx, uri, sock.RequestMethodPost, "push/notifications", requestTimeoutPushNotificationSend, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res pmnotification.SendResult
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
