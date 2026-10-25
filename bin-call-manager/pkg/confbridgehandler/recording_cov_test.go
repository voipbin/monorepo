package confbridgehandler

import (
	"context"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/recording"
)

func Test_RecordingStop_cov(t *testing.T) {
	ctx := context.Background()
	deleted := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		responseGet     *confbridge.Confbridge
		responseGetErr  error
		stopErr         error
		expectStop      bool
		setRecordingErr error
		expectSet       bool
	}{
		{
			name:           "get error",
			responseGetErr: errCov,
		},
		{
			name:        "no recording",
			responseGet: &confbridge.Confbridge{Identity: commonidentity.Identity{ID: covID}},
		},
		{
			name:        "already deleted",
			responseGet: &confbridge.Confbridge{Identity: commonidentity.Identity{ID: covID}, RecordingID: covRecording, TMDelete: &deleted},
		},
		{
			name:        "recording stop error",
			responseGet: &confbridge.Confbridge{Identity: commonidentity.Identity{ID: covID}, RecordingID: covRecording},
			expectStop:  true,
			stopErr:     errCov,
		},
		{
			name:            "update recording id error",
			responseGet:     &confbridge.Confbridge{Identity: commonidentity.Identity{ID: covID}, RecordingID: covRecording},
			expectStop:      true,
			expectSet:       true,
			setRecordingErr: errCov,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)

			if tt.responseGetErr != nil {
				m.db.EXPECT().ConfbridgeGet(ctx, covID).Return(nil, tt.responseGetErr)
			} else {
				m.db.EXPECT().ConfbridgeGet(ctx, covID).Return(tt.responseGet, nil)
			}
			if tt.expectStop {
				if tt.stopErr != nil {
					m.recording.EXPECT().Stop(ctx, covRecording).Return(nil, tt.stopErr)
				} else {
					m.recording.EXPECT().Stop(ctx, covRecording).Return(&recording.Recording{Identity: commonidentity.Identity{ID: covRecording}}, nil)
				}
			}
			if tt.expectSet {
				m.db.EXPECT().ConfbridgeSetRecordingID(ctx, covID, uuid.Nil).Return(tt.setRecordingErr)
			}

			if res, err := h.RecordingStop(ctx, covID); err == nil || res != nil {
				t.Errorf("expect error, got %v, %v", res, err)
			}
		})
	}
}
