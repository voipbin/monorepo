package callhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-call-manager/models/channel"
)

func Test_startContextRecording_covB(t *testing.T) {
	tests := []struct {
		name      string
		recordErr error
		expectErr bool
	}{
		{"success", nil, false},
		{"record error", fmt.Errorf("err"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			cn := &channel.Channel{
				ID: "ch-1",
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeReferenceType:         "call",
					channel.StasisDataTypeReferenceID:           "ref-1",
					channel.StasisDataTypeRecordingID:           "rec-1",
					channel.StasisDataTypeRecordingName:         "name",
					channel.StasisDataTypeRecordingFormat:       "wav",
					channel.StasisDataTypeRecordingDuration:     "60",
					channel.StasisDataTypeRecordingEndOfSilence: "5",
					channel.StasisDataTypeRecordingEndOfKey:     "#",
					channel.StasisDataTypeRecordingDirection:    "in",
				},
			}
			m.channel.EXPECT().Record(ctx, "ch-1", "name_in", "wav", 60, 5, false, "#", "fail").Return(tt.recordErr)

			err := h.startContextRecording(ctx, cn)
			if (err != nil) != tt.expectErr {
				t.Errorf("unexpected err: %v", err)
			}
		})
	}
}

func Test_startContextApplication_covB(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		continueErr error
		expectErr   bool
	}{
		{"amd success", applicationAMD, nil, false},
		{"amd continue error", applicationAMD, fmt.Errorf("err"), true},
		{"unknown application", "covb-unknown", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			cn := &channel.Channel{
				ID:         "ch-1",
				StasisData: map[channel.StasisDataType]string{channel.StasisDataTypeApplicationName: tt.appName},
			}
			if tt.appName == applicationAMD {
				m.channel.EXPECT().Continue(ctx, "ch-1", serviceContextAMD, "", 0, "").Return(tt.continueErr)
			}

			err := h.startContextApplication(ctx, cn)
			if (err != nil) != tt.expectErr {
				t.Errorf("unexpected err: %v", err)
			}
		})
	}
}
