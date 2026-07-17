package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-webchat-manager/models/widget"
)

// V1DataWidgetsPost is
// v1 data type request struct for
// /v1/widgets POST
type V1DataWidgetsPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`

	Name string `json:"name,omitempty"`

	SessionFlowID uuid.UUID `json:"session_flow_id,omitempty"`
	MessageFlowID uuid.UUID `json:"message_flow_id,omitempty"`

	SessionIdleTimeout int `json:"session_idle_timeout,omitempty"`

	ThemeConfig *widget.ThemeConfig `json:"theme_config,omitempty"`
}

// V1DataWidgetsIDPut is
// v1 data type request struct for
// /v1/widgets/<widget-id> PUT
type V1DataWidgetsIDPut struct {
	Name string `json:"name,omitempty"`

	SessionFlowID uuid.UUID `json:"session_flow_id,omitempty"`
	MessageFlowID uuid.UUID `json:"message_flow_id,omitempty"`

	SessionIdleTimeout int `json:"session_idle_timeout,omitempty"`

	ThemeConfig *widget.ThemeConfig `json:"theme_config,omitempty"`
}
