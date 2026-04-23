package request

// V1DataProvidersSetupPost is the request body for POST /v1/providers/setup.
type V1DataProvidersSetupPost struct {
	Carrier     string `json:"carrier"`
	Name        string `json:"name"`
	Detail      string `json:"detail"`
	Credentials struct {
		APIKey string `json:"api_key"`
	} `json:"credentials"`
}
