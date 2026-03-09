package pcapwatcher

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/voip-rtpengine-proxy/pkg/gcsuploader"
)

func TestProcessMetadataFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUploader := gcsuploader.NewMockUploader(ctrl)

	tmpDir := t.TempDir()
	pcapDir := filepath.Join(tmpDir, "pcap")
	metadataDir := filepath.Join(tmpDir, "metadata")
	_ = os.MkdirAll(pcapDir, 0755)
	_ = os.MkdirAll(metadataDir, 0755)

	pcapPath := filepath.Join(pcapDir, "testcall-tag1-ssrc1.pcap")
	_ = os.WriteFile(pcapPath, []byte("fake pcap data"), 0644)

	metadataPath := filepath.Join(metadataDir, "testcall-tag1")
	metadataContent := pcapPath + "\n\nSDP mode: offer\n"
	_ = os.WriteFile(metadataPath, []byte(metadataContent), 0644)

	expectedObject := "rtp-recordings/testcall-tag1-ssrc1.pcap"
	mockUploader.EXPECT().
		Upload(pcapPath, expectedObject).
		Return("gs://bucket/"+expectedObject, nil)

	w := &watcher{
		recordingDir: tmpDir,
		uploader:     mockUploader,
	}

	err := w.processMetadataFile(metadataPath)
	if err != nil {
		t.Fatalf("processMetadataFile() error = %v", err)
	}

	if _, err := os.Stat(pcapPath); !os.IsNotExist(err) {
		t.Error("pcap file should have been deleted after upload")
	}

	// Check that the .processing file was also deleted
	processingPath := metadataPath + ".processing"
	if _, err := os.Stat(processingPath); !os.IsNotExist(err) {
		t.Error("processing file should have been deleted after upload")
	}
}

func TestScanAndProcess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUploader := gcsuploader.NewMockUploader(ctrl)

	tmpDir := t.TempDir()
	pcapDir := filepath.Join(tmpDir, "pcap")
	metadataDir := filepath.Join(tmpDir, "metadata")
	_ = os.MkdirAll(pcapDir, 0755)
	_ = os.MkdirAll(metadataDir, 0755)

	// Create two recordings
	pcapPath1 := filepath.Join(pcapDir, "call1-tag1-ssrc1.pcap")
	_ = os.WriteFile(pcapPath1, []byte("pcap1"), 0644)
	metadataPath1 := filepath.Join(metadataDir, "call1-tag1")
	_ = os.WriteFile(metadataPath1, []byte(pcapPath1+"\n"), 0644)

	pcapPath2 := filepath.Join(pcapDir, "call2-tag2-ssrc2.pcap")
	_ = os.WriteFile(pcapPath2, []byte("pcap2"), 0644)
	metadataPath2 := filepath.Join(metadataDir, "call2-tag2")
	_ = os.WriteFile(metadataPath2, []byte(pcapPath2+"\n"), 0644)

	mockUploader.EXPECT().
		Upload(pcapPath1, "rtp-recordings/call1-tag1-ssrc1.pcap").
		Return("gs://bucket/rtp-recordings/call1-tag1-ssrc1.pcap", nil)
	mockUploader.EXPECT().
		Upload(pcapPath2, "rtp-recordings/call2-tag2-ssrc2.pcap").
		Return("gs://bucket/rtp-recordings/call2-tag2-ssrc2.pcap", nil)

	w := &watcher{
		recordingDir: tmpDir,
		uploader:     mockUploader,
	}

	w.scanAndProcess(metadataDir)

	// Verify all files cleaned up
	if _, err := os.Stat(pcapPath1); !os.IsNotExist(err) {
		t.Error("pcap1 should have been deleted")
	}
	if _, err := os.Stat(pcapPath2); !os.IsNotExist(err) {
		t.Error("pcap2 should have been deleted")
	}
}
