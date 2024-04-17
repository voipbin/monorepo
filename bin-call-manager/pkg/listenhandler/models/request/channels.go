package request

// V1DataChannelsIDHealth is
// v1 data type request struct for
// /v1/channels/<id>/health-check POST
type V1DataChannelsIDHealth struct {
	RetryCount int `json:"retry_count,omitempty"`
}
