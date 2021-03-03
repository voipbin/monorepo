package request

import "gitlab.com/voipbin/bin-manager/api-manager.git/models"

// ParamConferencesGET is rquest param define for GET /conferences
type ParamConferencesGET struct {
	Pagination
}

// BodyConferencesPOST is rquest body define for POST /conferences
type BodyConferencesPOST struct {
	Type   models.ConferenceType `json:"type" binding:"required"`
	Name   string                `json:"name"`
	Detail string                `json:"detail"`
}
