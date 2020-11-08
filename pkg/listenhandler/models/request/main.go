package request

// V1DataSpeechesPost is
// v1 data type request struct for
// /v1/speeches POST
type V1DataSpeechesPost struct {
	Text     string `json:"text"`
	Gender   string `json:"gender"`
	Language string `json:"language"`
}
