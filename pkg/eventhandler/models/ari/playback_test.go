package ari

import (
	"reflect"
	"testing"
)

func TestParsePlaybackContinuing(t *testing.T) {

	type test struct {
		name        string
		message     string
		expectEvent *PlaybackContinuing
	}

	tests := []test{
		{
			"normal",
			`{"type":"PlaybackContinuing","timestamp":"2020-08-25T20:36:26.174+0000","playback":{"id":"765ea59e-6e39-46e0-a5d2-502bd0a32d0e","media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","next_media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","target_uri":"channel:instance-asterisk-production-europe-west4-a-0-1598387623.33899","language":"en","state":"continuing"},"asterisk_id":"42:01:0a:a4:0f:d0","application":"test"}`,
			&PlaybackContinuing{
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
			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectEvent, e)
			}
		})
	}
}

func TestParsePlaybackStarted(t *testing.T) {

	type test struct {
		name        string
		message     string
		expectEvent *PlaybackStarted
	}

	tests := []test{
		{
			"normal",
			`{"type":"PlaybackStarted","timestamp":"2020-08-25T20:35:31.867+0000","playback":{"id":"765ea59e-6e39-46e0-a5d2-502bd0a32d0e","media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","next_media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","target_uri":"channel:instance-asterisk-production-europe-west4-a-0-1598387623.33899","language":"en","state":"playing"},"asterisk_id":"42:01:0a:a4:0f:d0","application":"test"}`,
			&PlaybackStarted{
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
			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectEvent, e)
			}
		})
	}
}

func TestParsePlaybackFinished(t *testing.T) {

	type test struct {
		name        string
		message     string
		expectEvent *PlaybackFinished
	}

	tests := []test{
		{
			"normal",
			`{"type":"PlaybackFinished","timestamp":"2020-08-25T20:37:20.485+0000","playback":{"id":"765ea59e-6e39-46e0-a5d2-502bd0a32d0e","media_uri":"sound:https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav","target_uri":"channel:instance-asterisk-production-europe-west4-a-0-1598387623.33899","language":"en","state":"done"},"asterisk_id":"42:01:0a:a4:0f:d0","application":"test"}`,
			&PlaybackFinished{
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

			e := evt.(*PlaybackFinished)
			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectEvent, e)
			}
		})
	}
}
