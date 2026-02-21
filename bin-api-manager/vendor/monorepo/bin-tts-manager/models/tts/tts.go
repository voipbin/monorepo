package tts

// Provider defines the TTS provider type
type Provider string

// list of provider types
const (
	ProviderGCP Provider = "gcp"
	ProviderAWS Provider = "aws"
)

// TTS define
type TTS struct {
	Provider        Provider `json:"provider,omitempty"`
	VoiceID         string   `json:"voice_id,omitempty"`
	Text            string   `json:"text,omitempty"`
	Language        string   `json:"language,omitempty"`
	MediaBucketName string   `json:"media_bucket_name,omitempty"`
	MediaFilepath   string   `json:"media_filepath,omitempty"`
}
