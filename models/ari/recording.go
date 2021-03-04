package ari

// RecordingLive struct
// https://wiki.asterisk.org/wiki/display/AST/Asterisk+17+REST+Data+Models#Asterisk17RESTDataModels-LiveRecording
type RecordingLive struct {
	Name            string `json:"name"`
	Format          string `json:"format"`
	State           string `json:"state"`
	SilenceDuration int    `json:"silence_duration"`
	Cause           string `json:"cause"`
	Duration        int    `json:"duration"`
	TalkingDuration int    `json:"talking_duration"`
	TargetURI       string `json:"target_uri"`
}

// RecordingStarted ARI event struct
// https://wiki.asterisk.org/wiki/display/AST/Asterisk+17+REST+Data+Models#Asterisk17RESTDataModels-PlaybackStarted
type RecordingStarted struct {
	Event
	Recording RecordingLive `json:"recording"`
}

// RecordingFinished ARI event struct
// https://wiki.asterisk.org/wiki/display/AST/Asterisk+17+REST+Data+Models#Asterisk17RESTDataModels-RecordingFinished
type RecordingFinished struct {
	Event
	Recording RecordingLive `json:"recording"`
}

// RecordingFailed ARI event struct
// https://wiki.asterisk.org/wiki/display/AST/Asterisk+17+REST+Data+Models#Asterisk17RESTDataModels-RecordingFailed
type RecordingFailed struct {
	Event
	Recording RecordingLive `json:"recording"`
}
