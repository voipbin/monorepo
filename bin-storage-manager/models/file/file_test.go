package file

import (
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestFileStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())
	accountID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	f := File{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerID: ownerID,
		},
		AccountID:     accountID,
		ReferenceType: ReferenceTypeRecording,
		ReferenceID:   referenceID,
		Name:          "test_file.wav",
		Detail:        "Test recording file",
		BucketName:    "voipbin-media",
		Filename:      "recording.wav",
		Filepath:      "recording/2023/01/",
		Filesize:      1048576,
		URIBucket:     "gs://voipbin-media/recording/2023/01/recording.wav",
		URIDownload:   "https://storage.googleapis.com/...",
	}

	if f.ID != id {
		t.Errorf("File.ID = %v, expected %v", f.ID, id)
	}
	if f.CustomerID != customerID {
		t.Errorf("File.CustomerID = %v, expected %v", f.CustomerID, customerID)
	}
	if f.OwnerID != ownerID {
		t.Errorf("File.OwnerID = %v, expected %v", f.OwnerID, ownerID)
	}
	if f.AccountID != accountID {
		t.Errorf("File.AccountID = %v, expected %v", f.AccountID, accountID)
	}
	if f.ReferenceType != ReferenceTypeRecording {
		t.Errorf("File.ReferenceType = %v, expected %v", f.ReferenceType, ReferenceTypeRecording)
	}
	if f.ReferenceID != referenceID {
		t.Errorf("File.ReferenceID = %v, expected %v", f.ReferenceID, referenceID)
	}
	if f.Name != "test_file.wav" {
		t.Errorf("File.Name = %v, expected %v", f.Name, "test_file.wav")
	}
	if f.Detail != "Test recording file" {
		t.Errorf("File.Detail = %v, expected %v", f.Detail, "Test recording file")
	}
	if f.BucketName != "voipbin-media" {
		t.Errorf("File.BucketName = %v, expected %v", f.BucketName, "voipbin-media")
	}
	if f.Filename != "recording.wav" {
		t.Errorf("File.Filename = %v, expected %v", f.Filename, "recording.wav")
	}
	if f.Filepath != "recording/2023/01/" {
		t.Errorf("File.Filepath = %v, expected %v", f.Filepath, "recording/2023/01/")
	}
	if f.Filesize != 1048576 {
		t.Errorf("File.Filesize = %v, expected %v", f.Filesize, 1048576)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_none", ReferenceTypeNone, ""},
		{"reference_type_normal", ReferenceTypeNormal, "normal"},
		{"reference_type_recording", ReferenceTypeRecording, "recording"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
