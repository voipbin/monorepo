package conference

import (
	"github.com/gofrs/uuid"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// Conference defines
// used only for the swag.
type Conference struct {
	ID   uuid.UUID         `json:"id"`
	Type cfconference.Type `json:"type"`

	Status cfconference.Status `json:"status"`

	Name    string                 `json:"name"`
	Detail  string                 `json:"detail"`
	Data    map[string]interface{} `json:"data"`
	Timeout int                    `json:"timeout"` // timeout. second

	PreActions  []fmaction.Action `json:"pre_actions"`  // pre actions
	PostActions []fmaction.Action `json:"post_actions"` // post actions

	CallIDs []uuid.UUID `json:"call_ids"` // list of call ids of conference

	RecordingID  uuid.UUID   `json:"recording_id"`
	RecordingIDs []uuid.UUID `json:"recording_ids"`

	WebhookURI string `json:"webhook_uri"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}
