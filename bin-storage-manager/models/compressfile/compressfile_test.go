package compress_file

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestCompressFileStruct(t *testing.T) {
	fileID1 := uuid.Must(uuid.NewV4())
	fileID2 := uuid.Must(uuid.NewV4())
	fileID3 := uuid.Must(uuid.NewV4())

	cf := CompressFile{
		FileIDs:          []uuid.UUID{fileID1, fileID2, fileID3},
		DownloadURI:      "https://storage.googleapis.com/voipbin-tmp/compressed.zip",
		TMDownloadExpire: "2023-01-02T00:00:00Z",
	}

	if len(cf.FileIDs) != 3 {
		t.Errorf("CompressFile.FileIDs length = %v, expected %v", len(cf.FileIDs), 3)
	}
	if cf.FileIDs[0] != fileID1 {
		t.Errorf("CompressFile.FileIDs[0] = %v, expected %v", cf.FileIDs[0], fileID1)
	}
	if cf.FileIDs[1] != fileID2 {
		t.Errorf("CompressFile.FileIDs[1] = %v, expected %v", cf.FileIDs[1], fileID2)
	}
	if cf.FileIDs[2] != fileID3 {
		t.Errorf("CompressFile.FileIDs[2] = %v, expected %v", cf.FileIDs[2], fileID3)
	}
	if cf.DownloadURI != "https://storage.googleapis.com/voipbin-tmp/compressed.zip" {
		t.Errorf("CompressFile.DownloadURI = %v, expected %v", cf.DownloadURI, "https://storage.googleapis.com/voipbin-tmp/compressed.zip")
	}
	if cf.TMDownloadExpire != "2023-01-02T00:00:00Z" {
		t.Errorf("CompressFile.TMDownloadExpire = %v, expected %v", cf.TMDownloadExpire, "2023-01-02T00:00:00Z")
	}
}

func TestCompressFileStructEmpty(t *testing.T) {
	cf := CompressFile{}

	if cf.FileIDs != nil {
		t.Errorf("CompressFile.FileIDs should be nil, got %v", cf.FileIDs)
	}
	if cf.DownloadURI != "" {
		t.Errorf("CompressFile.DownloadURI should be empty, got %v", cf.DownloadURI)
	}
}
