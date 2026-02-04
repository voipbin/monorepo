package bucketfile

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestBucketFileStruct(t *testing.T) {
	referenceID := uuid.Must(uuid.NewV4())

	bf := BucketFile{
		ReferenceType:    ReferenceTypeRecording,
		ReferenceID:      referenceID,
		BucketURI:        "gs://voipbin-media/recording/2023/01/file.wav",
		DownloadURI:      "https://storage.googleapis.com/voipbin-media/...",
		TMDownloadExpire: "2023-01-02T00:00:00Z",
	}

	if bf.ReferenceType != ReferenceTypeRecording {
		t.Errorf("BucketFile.ReferenceType = %v, expected %v", bf.ReferenceType, ReferenceTypeRecording)
	}
	if bf.ReferenceID != referenceID {
		t.Errorf("BucketFile.ReferenceID = %v, expected %v", bf.ReferenceID, referenceID)
	}
	if bf.BucketURI != "gs://voipbin-media/recording/2023/01/file.wav" {
		t.Errorf("BucketFile.BucketURI = %v, expected %v", bf.BucketURI, "gs://voipbin-media/recording/2023/01/file.wav")
	}
	if bf.DownloadURI != "https://storage.googleapis.com/voipbin-media/..." {
		t.Errorf("BucketFile.DownloadURI = %v, expected %v", bf.DownloadURI, "https://storage.googleapis.com/voipbin-media/...")
	}
	if bf.TMDownloadExpire != "2023-01-02T00:00:00Z" {
		t.Errorf("BucketFile.TMDownloadExpire = %v, expected %v", bf.TMDownloadExpire, "2023-01-02T00:00:00Z")
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
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
