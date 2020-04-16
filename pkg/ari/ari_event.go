package ari

import (
	"encoding/json"
	"strings"
)

// Event is ARI base event
type Event struct {
	Type        string `json:"type"`
	Application string `json:"application"`
	Timestamp   string `json:"timestamp"`
	AsteriskID  string `json:"asterisk_id"`
}

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

// Channel ARI message
type Channel struct {
	AccountCode  string      `json:"accoutcode"`
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Language     string      `json:"language"`
	CreationTime string      `json:"creationtime"`
	State        string      `json:"state"`
	Caller       CallerID    `json:"caller"`
	Connected    CallerID    `json:"connected"`
	Dialplan     DialplanCEP `json:"dialplan"`
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
	Channel  Channel `json:"channel"`
	CauseTxt string  `json:"cause_txt"`
	Cause    int     `json:"cause"`
}

// ChannelHangupRequest ARI event struct
type ChannelHangupRequest struct {
	Event
	Soft    bool    `json:"soft"`
	Cause   int     `json:"cause"`
	Channel Channel `json:"channel"`
}

// StasisStart ARI event struct
type StasisStart struct {
	Event
	Args           ArgsMap `json:"args"`
	Channel        Channel `json:"channel"`
	ReplaceChannel Channel `json:"replace_channel"`
}

// ArgsMap map for args
type ArgsMap map[string]string

var parseMap = map[string]interface{}{
	"ChannelCreated":       &ChannelCreated{},
	"ChannelDestroyed":     &ChannelDestroyed{},
	"ChannelHangupRequest": &ChannelHangupRequest{},
	"StasisStart":          &StasisStart{},
}

// Parse parses received event to corresponded ARI event interface.
// It returns nil interface if the message type's parser does not exists
// in the parseMap above.
func Parse(message []byte) (*Event, interface{}, error) {
	event := &Event{}
	if err := json.Unmarshal(message, event); err != nil {
		return nil, nil, err
	}

	res := parseMap[event.Type]
	if res == nil {
		return event, nil, nil
	}

	err := json.Unmarshal(message, res)
	if err != nil {
		return nil, nil, err
	}

	return event, res, nil
}

// UnmarshalJSON StasisStart
func (e *ArgsMap) UnmarshalJSON(m []byte) error {
	res := ArgsMap{}
	var arr []string
	if err := json.Unmarshal(m, &arr); err != nil {
		return err
	}

	// parse into map
	for _, pair := range arr {
		tmp := strings.Split(pair, "=")
		res[tmp[0]] = tmp[1]
	}

	*e = res
	return nil
}
