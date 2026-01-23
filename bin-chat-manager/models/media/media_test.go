package media

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestMediaStruct(t *testing.T) {
	fileID := uuid.Must(uuid.NewV4())

	m := Media{
		Type:    TypeFile,
		FileID:  fileID,
		LinkURL: "https://example.com/file",
	}

	if m.Type != TypeFile {
		t.Errorf("Media.Type = %v, expected %v", m.Type, TypeFile)
	}
	if m.FileID != fileID {
		t.Errorf("Media.FileID = %v, expected %v", m.FileID, fileID)
	}
	if m.LinkURL != "https://example.com/file" {
		t.Errorf("Media.LinkURL = %v, expected %v", m.LinkURL, "https://example.com/file")
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_address", TypeAddress, "address"},
		{"type_agent", TypeAgent, "agent"},
		{"type_file", TypeFile, "file"},
		{"type_link", TypeLink, "link"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
