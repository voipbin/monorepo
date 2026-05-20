package hook

// Hook defines
type Hook struct {
	ReceviedURI      string `json:"received_uri"`
	ReceivedData     []byte `json:"received_data"`
	ReceivedMethod    string `json:"received_method"`
	ReceivedSignature string `json:"received_signature"`
}
