package ari

// Playback ARI message
// https://wiki.asterisk.org/wiki/display/AST/Asterisk+17+REST+Data+Models#Asterisk17RESTDataModels-Playback
type Playback struct {
	ID           string        `json:"id"`             // ID for this playback operation
	Language     string        `json:"language"`       // For media types that support multiple languages, the language requested for playback.
	MediaURI     string        `json:"media_uri"`      // The URI for the media currently being played back.
	NextMediaURI string        `json:"next_media_uri"` // If a list of URIs is being played, the next media URI to be played back.
	State        PlaybackState `json:"state"`          // Current state of the playback operation.
	TargetURI    string        `json:"target_uri"`     // URI for the channel or bridge to play the media on
}

// PlaybackState type
type PlaybackState string

// List of PlaybackState
const (
	PlaybackStateQueued     PlaybackState = "queued"
	PlaybackStatePlaying    PlaybackState = "playing"
	PlaybackStateContinuing PlaybackState = "continuing"
	PlaybackStateDone       PlaybackState = "done"
)

// PlaybackContinuing ARI event struct
// https://wiki.asterisk.org/wiki/display/AST/Asterisk+17+REST+Data+Models#Asterisk17RESTDataModels-PlaybackFinished
type PlaybackContinuing struct {
	Event
	Playback Playback `json:"playback"`
}

// PlaybackFinished ARI event struct
// https://wiki.asterisk.org/wiki/display/AST/Asterisk+17+REST+Data+Models#Asterisk17RESTDataModels-PlaybackFinished
type PlaybackFinished struct {
	Event
	Playback Playback `json:"playback"`
}

// PlaybackStarted ARI event struct
// https://wiki.asterisk.org/wiki/display/AST/Asterisk+17+REST+Data+Models#Asterisk17RESTDataModels-PlaybackStarted
type PlaybackStarted struct {
	Event
	Playback Playback `json:"playback"`
}
