package transcripthandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	smfile "monorepo/bin-storage-manager/models/file"

	"monorepo/bin-transcribe-manager/models/transcript"
)

func Test_Recording_listFilesError(t *testing.T) {
	recordingID := uuid.FromStringOrNil("5b0f1a10-0000-11ee-9d23-000000000001")

	tests := []struct {
		name string

		responseFiles []smfile.File
		responseErr   error

		expectRes int
		expectErr bool
	}{
		{
			name:        "file list failure is wrapped and returned",
			responseErr: fmt.Errorf("storage unavailable"),
			expectErr:   true,
		},
		{
			name:          "no files results in an empty transcript list",
			responseFiles: []smfile.File{},
			expectRes:     0,
			expectErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &transcriptHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			expectFilters := map[smfile.Field]any{
				smfile.FieldDeleted:     false,
				smfile.FieldReferenceID: recordingID.String(),
			}
			mockReq.EXPECT().StorageV1FileList(ctx, "", uint64(100), expectFilters).Return(tt.responseFiles, tt.responseErr)

			res, err := h.Recording(ctx, uuid.FromStringOrNil("5b0f1a10-0000-11ee-9d23-000000000002"), uuid.FromStringOrNil("5b0f1a10-0000-11ee-9d23-000000000003"), recordingID, "en-US")
			if tt.expectErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: err, got: ok")
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || len(res) != tt.expectRes {
				t.Errorf("Wrong match. expect: empty non-nil slice of %d, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_recordingGetTranscripts_noFiles(t *testing.T) {
	h := &transcriptHandler{}

	res, err := h.recordingGetTranscripts(context.Background(), nil, "en-US", "5b0f1a10-0000-11ee-9d23-000000000004")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: empty, got: %v", res)
	}
}

func Test_parseDirection_extensions(t *testing.T) {
	tests := []struct {
		name string

		filename string

		expectRes transcript.Direction
	}{
		{"in wav", "call_1234_in.wav", transcript.DirectionIn},
		{"out mp3", "call_1234_out.mp3", transcript.DirectionOut},
		{"in with digits in extension", "call_1234_in.mp4a", transcript.DirectionIn},
		{"no direction suffix", "call_1234.wav", transcript.DirectionBoth},
		{"direction in the middle only", "call_in_1234.wav", transcript.DirectionBoth},
		{"missing extension", "call_1234_in", transcript.DirectionBoth},
		{"uppercase direction is not matched", "call_1234_IN.wav", transcript.DirectionBoth},
		{"unknown direction word", "call_1234_both.wav", transcript.DirectionBoth},
		{"empty name", "", transcript.DirectionBoth},
		{"only the last suffix counts", "call_in_out.wav", transcript.DirectionOut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := parseDirection(tt.filename)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_toOffsetMs_values(t *testing.T) {
	tests := []struct {
		name string

		duration time.Duration

		expectRes int64
	}{
		{"zero", 0, 0},
		{"one and a half seconds", 1500 * time.Millisecond, 1500},
		{"sub millisecond is truncated", 999 * time.Microsecond, 0},
		{"one hour", time.Hour, 3600000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := toOffsetMs(tt.duration)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectRes, res)
			}
		})
	}
}
