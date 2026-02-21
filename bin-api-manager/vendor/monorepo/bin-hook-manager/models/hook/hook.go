package hook

// Hook defines
type Hook struct {
	ReceviedURI  string `json:"received_uri"`
	ReceivedData []byte `json:"received_data"`
}
