package tts

// TTS define
type TTS struct {
	Gender          Gender `json:"gender"`
	Text            string `json:"text"`
	Language        string `json:"language"`
	MediaBucketName string `json:"media_bucket_name"`
	MediaFilepath   string `json:"media_filepath"`
}

// Gender define
type Gender string

// list of gender types
const (
	GenderMale    = "male"
	GenderFemale  = "female"
	GenderNeutral = "neutral"
)
