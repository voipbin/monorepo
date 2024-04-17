
Action
=========================

echo
-------------------------
Echo to voice and DTMFs.

	Duration int  `json:"duration"` // echo duration. ms
	DTMF     bool `json:"dtmf"`     // sending back the dtmf on/off


stream-echo
-------------------------
Echo the incoming streams. Voice/Video including DTMFs.

	// no option


