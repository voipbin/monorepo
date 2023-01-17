package tts

type TTS struct {
	Gender        Gender `json:"gender"`
	Text          string `json:"text"`
	Language      string `json:"language"`
	MediaFilepath string `json:"media_filepath"`
}

// Gender define
type Gender string

// list of gender types
const (
	GenderMale    = "male"
	GenderFemale  = "female"
	GenderNeutral = "neutral"
)
