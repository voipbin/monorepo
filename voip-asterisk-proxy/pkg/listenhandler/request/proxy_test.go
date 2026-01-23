package request

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestProxyDataRecordingFileMovePost(t *testing.T) {
	tests := []struct {
		name string

		filenames       []string
		expectFilenames []string
	}{
		{
			name: "creates_request_with_single_filename",

			filenames:       []string{"recording-001.wav"},
			expectFilenames: []string{"recording-001.wav"},
		},
		{
			name: "creates_request_with_multiple_filenames",

			filenames:       []string{"recording-001.wav", "recording-002.wav", "recording-003.wav"},
			expectFilenames: []string{"recording-001.wav", "recording-002.wav", "recording-003.wav"},
		},
		{
			name: "creates_request_with_empty_filenames",

			filenames:       []string{},
			expectFilenames: []string{},
		},
		{
			name: "creates_request_with_nil_filenames",

			filenames:       nil,
			expectFilenames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ProxyDataRecordingFileMovePost{
				Filenames: tt.filenames,
			}

			if !reflect.DeepEqual(req.Filenames, tt.expectFilenames) {
				t.Errorf("Wrong Filenames. expect: %v, got: %v", tt.expectFilenames, req.Filenames)
			}
		})
	}
}

func TestProxyDataRecordingFileMovePost_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name string

		req         ProxyDataRecordingFileMovePost
		expectJSON  string
		expectMatch bool
	}{
		{
			name: "marshals_with_filenames",

			req: ProxyDataRecordingFileMovePost{
				Filenames: []string{"recording-001.wav", "recording-002.wav"},
			},
			expectJSON:  `{"filenames":["recording-001.wav","recording-002.wav"]}`,
			expectMatch: true,
		},
		{
			name: "marshals_empty_filenames_with_omitempty",

			req:         ProxyDataRecordingFileMovePost{},
			expectJSON:  `{}`,
			expectMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Failed to marshal JSON: %v", err)
				return
			}

			if tt.expectMatch && string(jsonBytes) != tt.expectJSON {
				t.Errorf("Wrong JSON. expect: %s, got: %s", tt.expectJSON, string(jsonBytes))
			}
		})
	}
}

func TestProxyDataRecordingFileMovePost_JSONUnmarshaling(t *testing.T) {
	tests := []struct {
		name string

		jsonInput       string
		expectFilenames []string
		expectErr       bool
	}{
		{
			name: "unmarshals_with_filenames",

			jsonInput:       `{"filenames":["recording-001.wav","recording-002.wav"]}`,
			expectFilenames: []string{"recording-001.wav", "recording-002.wav"},
			expectErr:       false,
		},
		{
			name: "unmarshals_empty_json",

			jsonInput:       `{}`,
			expectFilenames: nil,
			expectErr:       false,
		},
		{
			name: "unmarshals_with_empty_array",

			jsonInput:       `{"filenames":[]}`,
			expectFilenames: []string{},
			expectErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req ProxyDataRecordingFileMovePost
			err := json.Unmarshal([]byte(tt.jsonInput), &req)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(req.Filenames, tt.expectFilenames) {
				t.Errorf("Wrong Filenames. expect: %v, got: %v", tt.expectFilenames, req.Filenames)
			}
		})
	}
}
