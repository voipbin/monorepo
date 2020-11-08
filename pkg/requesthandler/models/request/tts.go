package request

// TTSV1DataSpeechesPost is
// v1 data type struct for
// /v1/speeches POST request
type TTSV1DataSpeechesPost struct {
	Text     string `json:"text"`
	Gender   string `json:"gender"`
	Language string `json:"language"`
}
