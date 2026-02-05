package transcripthandler

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_sortTranscriptsByTMTranscript(t *testing.T) {

	tests := []struct {
		name string

		transcripts []*transcript.Transcript

		expectRes []*transcript.Transcript
	}{
		{
			name: "normal",

			transcripts: []*transcript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("494f5bfc-7eb5-11ed-a6d7-07162f18f28e"),
					},
					TMTranscript: func() *time.Time { t := time.Date(2022, 1, 1, 0, 0, 1, 10000, time.UTC); return &t }(),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d1f9be6e-0b23-11f0-b828-37e1c878aff0"),
					},
					TMTranscript: func() *time.Time { t := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d21e19ee-0b23-11f0-aad2-73ff70024ad9"),
					},
					TMTranscript: func() *time.Time { t := time.Date(2022, 1, 1, 0, 0, 1, 0, time.UTC); return &t }(),
				},
			},
			expectRes: []*transcript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d1f9be6e-0b23-11f0-b828-37e1c878aff0"),
					},
					TMTranscript: func() *time.Time { t := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d21e19ee-0b23-11f0-aad2-73ff70024ad9"),
					},
					TMTranscript: func() *time.Time { t := time.Date(2022, 1, 1, 0, 0, 1, 0, time.UTC); return &t }(),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("494f5bfc-7eb5-11ed-a6d7-07162f18f28e"),
					},
					TMTranscript: func() *time.Time { t := time.Date(2022, 1, 1, 0, 0, 1, 10000, time.UTC); return &t }(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			sortTranscriptsByTMTranscript(tt.transcripts)
			if !reflect.DeepEqual(tt.transcripts, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, tt.transcripts)
			}
		})
	}
}
