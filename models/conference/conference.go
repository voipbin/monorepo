package conference

import (
	uuid "github.com/gofrs/uuid"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
)

// Conference type for client show
type Conference struct {
	ID     uuid.UUID `json:"id"`      // Conference's ID.
	UserID uint64    `json:"user_id"` // Conference owner's ID.
	Type   Type      `json:"type"`    // Conference's type.

	Status Status `json:"status"` // Status.
	Name   string `json:"name"`   // Name.
	Detail string `json:"detail"` // Detail.

	PreActions  []action.Action // actions before joining to the conference.
	PostActions []action.Action // actions after leaving from the conference.

	CallIDs []uuid.UUID `json:"call_ids"` // Currently joined call IDs.

	RecordingID  uuid.UUID   `json:"recording_id"`  // Currently recording ID.
	RecordingIDs []uuid.UUID `json:"recording_ids"` // Recorded recording IDs.

	WebhookURI string `json:"webhook_uri"` // webhook uri

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// Type conference types
type Type string

// List of conference(bridge) types
const (
	TypeNone       Type = ""
	TypeConference Type = "conference" // conference for more than 3 calls join
)

// Status type
type Status string

// List of Status types
const (
	StatusStarting    Status = "starting"
	StatusProgressing Status = "progressing"
	StatusTerminating Status = "terminating"
	StatusTerminated  Status = "terminated"
)

// Convert returns conference.Conference from cfconference.Conference
func Convert(conf *cfconference.Conference) *Conference {

	preActions := []action.Action{}
	for _, a := range conf.PreActions {
		preActions = append(preActions, *action.ConvertAction(&a))
	}

	postActions := []action.Action{}
	for _, a := range conf.PostActions {
		postActions = append(postActions, *action.ConvertAction(&a))
	}

	res := &Conference{
		ID:     conf.ID,
		UserID: conf.UserID,
		Type:   Type(conf.Type),

		Status: Status(conf.Status),
		Name:   conf.Name,
		Detail: conf.Detail,

		PreActions:  preActions,
		PostActions: postActions,

		CallIDs: conf.CallIDs,

		RecordingID:  conf.RecordingID,
		RecordingIDs: conf.RecordingIDs,

		WebhookURI: conf.WebhookURI,

		TMCreate: conf.TMCreate,
		TMUpdate: conf.TMUpdate,
		TMDelete: conf.TMDelete,
	}

	return res
}
