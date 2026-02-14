package media

import (
	"testing"
)

func TestMedia(t *testing.T) {
	tests := []struct {
		name string

		mediaType Type
		filename  string
	}{
		{
			name: "creates_media_with_all_fields",

			mediaType: TypeImage,
			filename:  "image.jpg",
		},
		{
			name: "creates_media_with_empty_fields",

			mediaType: "",
			filename:  "",
		},
		{
			name: "creates_media_with_video_type",

			mediaType: TypeVideo,
			filename:  "video.mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Media{
				Type:     tt.mediaType,
				Filename: tt.filename,
			}

			if m.Type != tt.mediaType {
				t.Errorf("Wrong Type. expect: %s, got: %s", tt.mediaType, m.Type)
			}
			if m.Filename != tt.filename {
				t.Errorf("Wrong Filename. expect: %s, got: %s", tt.filename, m.Filename)
			}
		})
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{
			name:     "type_image",
			constant: TypeImage,
			expected: "image",
		},
		{
			name:     "type_video",
			constant: TypeVideo,
			expected: "video",
		},
		{
			name:     "type_audio",
			constant: TypeAudio,
			expected: "audio",
		},
		{
			name:     "type_file",
			constant: TypeFile,
			expected: "file",
		},
		{
			name:     "type_location",
			constant: TypeLocation,
			expected: "location",
		},
		{
			name:     "type_sticker",
			constant: TypeSticker,
			expected: "sticker",
		},
		{
			name:     "type_template",
			constant: TypeTemplate,
			expected: "template",
		},
		{
			name:     "type_imagemap",
			constant: TypeImagemap,
			expected: "imagemap",
		},
		{
			name:     "type_flex",
			constant: TypeFlex,
			expected: "flex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
