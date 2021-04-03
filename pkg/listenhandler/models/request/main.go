package request

// Pagination is pagination structure for request
type Pagination struct {
	PageSize  uint64 `form:"page_size" json:"page_size"`
	PageToken string `form:"page_token" json:"page_token"`
}

// V1DataAsterisksIDChannelsIDHealth is
// v1 data type request struct for
// AsterisksIDChannelsIDHealth
// /v1/asterisks/<id>/channels/<id>/health-check POST
type V1DataAsterisksIDChannelsIDHealth struct {
	RetryCount    int `json:"retry_count"`
	RetryCountMax int `json:"retry_count_max"`
	Delay         int `json:"delay"`
}
