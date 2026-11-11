package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-push-manager/models/device"
	"monorepo/bin-push-manager/models/notification"
)

// V1DataDevicesPost is the request body of POST /v1/devices.
type V1DataDevicesPost struct {
	CustomerID uuid.UUID       `json:"customer_id"`
	AgentID    uuid.UUID       `json:"agent_id"`
	Platform   device.Platform `json:"platform"`
	Token      string          `json:"token"`
}

// V1DataNotificationsPost is the request body of POST /v1/notifications.
// An empty DeviceID addresses every live device of the agent.
type V1DataNotificationsPost struct {
	CustomerID uuid.UUID         `json:"customer_id"`
	AgentID    uuid.UUID         `json:"agent_id"`
	DeviceID   uuid.UUID         `json:"device_id"`
	Kind       notification.Kind `json:"kind"`
	Data       map[string]string `json:"data"`
}

// V1DataSettingsPut is the request body of PUT /v1/settings. Kinds has the notification kinds to switch:
// true is on, false is off. A kind that is not in the map is not changed.
type V1DataSettingsPut struct {
	CustomerID uuid.UUID                  `json:"customer_id"`
	AgentID    uuid.UUID                  `json:"agent_id"`
	Kinds      map[notification.Kind]bool `json:"kinds"`
}
