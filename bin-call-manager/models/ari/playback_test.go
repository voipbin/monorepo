package ari

import (
	"reflect"
	"testing"
)

func TestParsePlaybackContinuing(t *testing.T) {

	tests := []struct {
		name      string
		message   string
		expectRes *PlaybackContinuing
	}{
		{
			name:    "normal",
			message: `{"type":"PlaybackContinuing","timestamp":"2020-08-25T20:36:26.174+0000","playback":{"id":"765ea59e-6e39-46e0-a5d2-502bd0a32d0e","media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","next_media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","target_uri":"channel:instance-asterisk-production-europe-west4-a-0-1598387623.33899","language":"en","state":"continuing"},"asterisk_id":"42:01:0a:a4:0f:d0","application":"test"}`,
			expectRes: &PlaybackContinuing{
				Event{
					Type:        EventTypePlaybackContinuing,
					Application: "test",
					Timestamp:   "2020-08-25T20:36:26.174",
					AsteriskID:  "42:01:0a:a4:0f:d0",
				},
				Playback{
					ID:           "765ea59e-6e39-46e0-a5d2-502bd0a32d0e",
					MediaURI:     "sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav",
					NextMediaURI: "sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav",
					TargetURI:    "channel:instance-asterisk-production-europe-west4-a-0-1598387623.33899",
					Language:     "en",
					State:        "continuing",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			e := evt.(*PlaybackContinuing)
			if reflect.DeepEqual(tt.expectRes, e) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, e)
			}
		})
	}
}

func TestParsePlaybackStarted(t *testing.T) {

	tests := []struct {
		name      string
		message   string
		expectRes *PlaybackStarted
	}{
		{
			name:    "normal",
			message: `{"type":"PlaybackStarted","timestamp":"2020-08-25T20:35:31.867+0000","playback":{"id":"765ea59e-6e39-46e0-a5d2-502bd0a32d0e","media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","next_media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","target_uri":"channel:instance-asterisk-production-europe-west4-a-0-1598387623.33899","language":"en","state":"playing"},"asterisk_id":"42:01:0a:a4:0f:d0","application":"test"}`,
			expectRes: &PlaybackStarted{
				Event{
					Type:        EventTypePlaybackStarted,
					Application: "test",
					Timestamp:   "2020-08-25T20:35:31.867",
					AsteriskID:  "42:01:0a:a4:0f:d0",
				},
				Playback{
					ID:           "765ea59e-6e39-46e0-a5d2-502bd0a32d0e",
					MediaURI:     "sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav",
					NextMediaURI: "sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav",
					TargetURI:    "channel:instance-asterisk-production-europe-west4-a-0-1598387623.33899",
					Language:     "en",
					State:        "playing",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			e := evt.(*PlaybackStarted)
			if reflect.DeepEqual(tt.expectRes, e) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, e)
			}
		})
	}
}

func TestParsePlaybackFinished(t *testing.T) {

	tests := []struct {
		name      string
		message   string
		expectRes *PlaybackFinished
	}{
		{
			name:    "normal",
			message: `{"type":"PlaybackFinished","timestamp":"2020-08-25T20:37:20.485+0000","playback":{"id":"765ea59e-6e39-46e0-a5d2-502bd0a32d0e","media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","target_uri":"channel:instance-asterisk-production-europe-west4-a-0-1598387623.33899","language":"en","state":"done"},"asterisk_id":"42:01:0a:a4:0f:d0","application":"test"}`,
			expectRes: &PlaybackFinished{
				Event{
					Type:        EventTypePlaybackFinished,
					Application: "test",
					Timestamp:   "2020-08-25T20:37:20.485",
					AsteriskID:  "42:01:0a:a4:0f:d0",
				},
				Playback{
					ID:        "765ea59e-6e39-46e0-a5d2-502bd0a32d0e",
					MediaURI:  "sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav",
					TargetURI: "channel:instance-asterisk-production-europe-west4-a-0-1598387623.33899",
					Language:  "en",
					State:     "done",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res := evt.(*PlaybackFinished)
			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ParsePlayback(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		expectRes *Playback
	}{
		{
			name:    "normal",
			message: `{"id":"f92d2a44-ea7e-406b-b18d-f8df6585b60e","media_uri":"sound:demo-congrats","target_uri":"channel:01k37s659a7qj3psqrvy90vppa-ch","language":"en","state":"queued"}`,
			expectRes: &Playback{
				ID:        "f92d2a44-ea7e-406b-b18d-f8df6585b60e",
				MediaURI:  "sound:demo-congrats",
				TargetURI: "channel:01k37s659a7qj3psqrvy90vppa-ch",
				Language:  "en",
				State:     "queued",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParsePlayback([]byte(tt.message))
			if err != nil {
				t.Errorf("Wront match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
