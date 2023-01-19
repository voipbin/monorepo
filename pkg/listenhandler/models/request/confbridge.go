package request

import "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"

// V1DataConfbridgesPost is
// v1 data type request struct for
// /v1/confbridges POST
type V1DataConfbridgesPost struct {
	Type confbridge.Type `json:"type"`
}

// V1DataConfbridgesIDExternalMediaPost is
// v1 data type for
// /v1/confbridges/<confbridge-id>/external-media POST
type V1DataConfbridgesIDExternalMediaPost struct {
	ExternalHost   string `json:"external_host"`
	Encapsulation  string `json:"encapsulation"`
	Transport      string `json:"transport"`
	ConnectionType string `json:"connection_type"`
	Format         string `json:"format"`
	Direction      string `json:"direction"`
}
