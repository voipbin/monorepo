package servicehandler

import (
	"os"
	"testing"

	"go.uber.org/mock/gomock"
)

func Test_recordingFileMove(t *testing.T) {

	tests := []struct {
		name                       string
		recordingAsteriskDirectory string
		recordingBucketDirectory   string

		filename string
	}{
		{
			name:                       "normal",
			recordingAsteriskDirectory: "/tmp/asterisk",
			recordingBucketDirectory:   "/tmp/bucket",

			filename: "caa168fa-0877-11f0-9887-3be913496cb6.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &serviceHandler{
				recordingAsteriskDirectory: tt.recordingAsteriskDirectory,
				recordingBucketDirectory:   tt.recordingBucketDirectory,
			}

			// create directories
			if err := os.MkdirAll(tt.recordingAsteriskDirectory, os.ModePerm); err != nil {
				t.Fatalf("Failed to create asterisk directory: %v", err)
			}
			if err := os.MkdirAll(tt.recordingBucketDirectory, os.ModePerm); err != nil {
				t.Fatalf("Failed to create bucket directory: %v", err)
			}

			// create file
			filePath := tt.recordingAsteriskDirectory + "/" + tt.filename
			file, err := os.Create(filePath)
			if err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}
			file.Close()

			if errMove := h.recordingFileMove(tt.filename); errMove != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errMove)
			}

			// check if the file is moved
			if _, err := os.Stat(tt.recordingBucketDirectory + "/" + tt.filename); os.IsNotExist(err) {
				t.Errorf("The file is not moved. err: %v", err)
			}
		})
	}
}
