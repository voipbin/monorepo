package ari

import (
	"encoding/json"
	"strings"
)

// Event is ARI base event
type Event struct {
	Type        EventType `json:"type"`
	Application string    `json:"application"`
	Timestamp   Timestamp `json:"timestamp"`
	AsteriskID  string    `json:"asterisk_id"`
}

// EventType type
type EventType string

// List of ChannelType types
const (
	EventTypeBridgeCreated        EventType = "BridgeCreated"
	EventTypeBridgeDestroyed      EventType = "BridgeDestroyed"
	EventTypeChannelCreated       EventType = "ChannelCreated"
	EventTypeChannelDestroyed     EventType = "ChannelDestroyed"
	EventTypeChannelDtmfReceived  EventType = "ChannelDtmfReceived"
	EventTypeChannelEnteredBridge EventType = "ChannelEnteredBridge"
	EventTypeChannelHangupRequest EventType = "ChannelHangupRequest"
	EventTypeChannelLeftBridge    EventType = "ChannelLeftBridge"
	EventTypeChannelStateChange   EventType = "ChannelStateChange"
	EventTypeChannelVarset        EventType = "ChannelVarset"
	EventTypeContactStatusChange  EventType = "ContactStatusChange"
	EventTypePeerStatusChange     EventType = "PeerStatusChange"
	EventTypePlaybackContinuing   EventType = "PlaybackContinuing"
	EventTypePlaybackFinished     EventType = "PlaybackFinished"
	EventTypePlaybackStarted      EventType = "PlaybackStarted"
	EventTypeRecordingFailed      EventType = "RecordingFailed"
	EventTypeRecordingFinished    EventType = "RecordingFinished"
	EventTypeRecordingStarted     EventType = "RecordingStarted"
	EventTypeStasisEnd            EventType = "StasisEnd"
	EventTypeStasisStart          EventType = "StasisStart"
)

// Timestamp for timestamp
type Timestamp string

// ArgsMap map for args
type ArgsMap map[string]string

// Parse parses received event to corresponded ARI event interface.
// It returns nil interface if the message type's parser does not exists
// in the parseMap.
func Parse(message []byte) (*Event, interface{}, error) {
	event := &Event{}
	if err := json.Unmarshal(message, event); err != nil {
		return nil, nil, err
	}

	var parseMap = map[EventType]interface{}{
		EventTypeBridgeCreated:        &BridgeCreated{},
		EventTypeBridgeDestroyed:      &BridgeDestroyed{},
		EventTypeChannelCreated:       &ChannelCreated{},
		EventTypeChannelDestroyed:     &ChannelDestroyed{},
		EventTypeChannelDtmfReceived:  &ChannelDtmfReceived{},
		EventTypeChannelEnteredBridge: &ChannelEnteredBridge{},
		EventTypeChannelHangupRequest: &ChannelHangupRequest{},
		EventTypeChannelLeftBridge:    &ChannelLeftBridge{},
		EventTypeChannelStateChange:   &ChannelStateChange{},
		EventTypeChannelVarset:        &ChannelVarset{},
		EventTypeContactStatusChange:  &ContactStatusChange{},
		EventTypePeerStatusChange:     &PeerStatusChange{},
		EventTypePlaybackContinuing:   &PlaybackContinuing{},
		EventTypePlaybackFinished:     &PlaybackFinished{},
		EventTypePlaybackStarted:      &PlaybackStarted{},
		EventTypeRecordingFailed:      &RecordingFailed{},
		EventTypeRecordingFinished:    &RecordingFinished{},
		EventTypeRecordingStarted:     &RecordingStarted{},
		EventTypeStasisEnd:            &StasisEnd{},
		EventTypeStasisStart:          &StasisStart{},
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
	var arr []string
	if err := json.Unmarshal(m, &arr); err != nil {
		return err
	}

	// parse into map
	res := ArgsMap{}
	for _, pair := range arr {
		tmp := strings.Split(pair, "=")
		if len(tmp) == 1 {
			res[tmp[0]] = ""
		} else {
			res[tmp[0]] = tmp[1]
		}
	}

	*e = res
	return nil
}

// UnmarshalJSON Timestamp
func (e *Timestamp) UnmarshalJSON(m []byte) error {
	var tmp string

	if err := json.Unmarshal(m, &tmp); err != nil {
		return err
	}
	res := strings.TrimSuffix(tmp, "+0000")
	*e = Timestamp(res)

	return nil
}
