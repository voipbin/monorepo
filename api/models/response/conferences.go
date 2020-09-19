package response

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
)

// BodyConferencesGET is rquest body define for GET /calls
type BodyConferencesGET struct {
	Result []*conference.Conference `json:"result"`
	Pagination
}
