package channel

// Channel struct represent asterisk's channel information
type Channel struct {
	// identity
	ID         string
	AsteriskID string
	Name       string
	Tech       string

	// source/destination
	SourceName        string
	SourceNumber      string
	DestinationName   string
	DestinationNumber string

	State string
	Data  map[string]string

	DialResult  string
	HangupCause int

	TMCreate string
	TMUpdate string

	TMAnswer  string
	TMRinging string
	TMEnd     string
}

