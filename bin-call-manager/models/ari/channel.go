package ari

import "encoding/json"

// CallerID Caller ID
type CallerID struct {
	Name   string `json:"name"`
	Number string `json:"number"`
}

// DialplanCEP Dialplan
type DialplanCEP struct {
	Context  string `json:"context"`
	Exten    string `json:"exten"`
	Priority int    `json:"priority"`
	AppName  string `json:"app_name"`
	AppData  string `json:"app_data"`
}

// ChannelState type
type ChannelState string

// List of ChannelState
const (
	ChannelStateDown           ChannelState = "Down"
	ChannelStateRsrvd          ChannelState = "Rsrvd"
	ChannelStateOffHook        ChannelState = "OffHook"
	ChannelStateDialing        ChannelState = "Dialing"
	ChannelStateRing           ChannelState = "Ring"
	ChannelStateRinging        ChannelState = "Ringing"
	ChannelStateUp             ChannelState = "Up"
	ChannelStateBusy           ChannelState = "Busy"
	ChannelStateDialingOffHook ChannelState = "Dialing Offhook"
	ChannelStatePreRing        ChannelState = "Pre-ring"
	ChannelStateMute           ChannelState = "Mute"
	ChannelStateUnknown        ChannelState = "Unknown"
)

// Channel ARI message
type Channel struct {
	AccountCode  string       `json:"accoutcode"`
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Language     string       `json:"language"`
	CreationTime Timestamp    `json:"creationtime"`
	State        ChannelState `json:"state"`
	Caller       CallerID     `json:"caller"`
	Connected    CallerID     `json:"connected"`
	Dialplan     DialplanCEP  `json:"dialplan"`
	ChannelVars  map[string]string
}

// ChannelCreated ARI event struct
type ChannelCreated struct {
	Event
	Channel Channel `json:"channel"`
}

// ChannelDestroyed ARI event struct
type ChannelDestroyed struct {
	Event
	Channel  Channel      `json:"channel"`
	CauseTxt string       `json:"cause_txt"`
	Cause    ChannelCause `json:"cause"`
}

// ChannelHangupRequest ARI event struct
type ChannelHangupRequest struct {
	Event
	Soft    bool         `json:"soft"`
	Cause   ChannelCause `json:"cause"`
	Channel Channel      `json:"channel"`
}

// ChannelStateChange ARI event struct
type ChannelStateChange struct {
	Event
	Channel Channel `json:"channel"`
}

// ChannelEnteredBridge ARI event struct
type ChannelEnteredBridge struct {
	Event
	Channel Channel `json:"channel"`
	Bridge  Bridge  `json:"bridge"`
}

// ChannelLeftBridge ARI event struct
type ChannelLeftBridge struct {
	Event
	Channel Channel `json:"channel"`
	Bridge  Bridge  `json:"bridge"`
}

// ChannelDtmfReceived ARI event struct
type ChannelDtmfReceived struct {
	Event
	Digit    string  `json:"digit"`
	Duration int     `json:"duration_ms"`
	Channel  Channel `json:"channel"`
}

// ChannelVarset ARI event struct
type ChannelVarset struct {
	Event
	Variable string  `json:"variable"`
	Value    string  `json:"value"`
	Channel  Channel `json:"channel"`
}

// ParseChannel parses message into Channel struct
func ParseChannel(message []byte) (*Channel, error) {
	res := &Channel{}

	err := json.Unmarshal(message, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
