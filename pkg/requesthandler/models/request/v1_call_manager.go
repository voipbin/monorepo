package request

// V1DataAsterisksIDChannelsIDHealth is
// v1 data type request struct for
// AsterisksIDChannelsIDHealth
// /v1/asterisks/<id>/channels/<id>/health-check POST
type V1DataAsterisksIDChannelsIDHealth struct {
	RetryCount    int `json:"retry_count"`
	RetryCountMax int `json:"retry_count_max"`
	Delay         int `json:"delay"`
}

// V1DataCallsIDHealth is
// v1 data type request struct for
// CallsIDHealth
// /v1/calls/<id>/health-check POST
type V1DataCallsIDHealth struct {
	RetryCount int `json:"retry_count"`
	Delay      int `json:"delay"`
}

// V1DataConferencesIDDelete is
// v1 data type request struct for
// /v1/conferences/<id>" DELETE
type V1DataConferencesIDDelete struct {
	Reason string `json:"reason,omitempty"`
}
