package recordinghandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_GetByRecordingName_cov(t *testing.T) {
	tests := []struct {
		name string

		responseRecording *recording.Recording
		responseErr       error

		expectErr bool
	}{
		{
			name:              "normal",
			responseRecording: &recording.Recording{RecordingName: "rec_name"},
		},
		{
			name:              "db error",
			responseRecording: &recording.Recording{RecordingName: "rec_name"},
			responseErr:       errors.New("db error"),
			expectErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &recordingHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().RecordingGetByRecordingName(ctx, "rec_name").Return(tt.responseRecording, tt.responseErr)

			res, err := h.GetByRecordingName(ctx, "rec_name")
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong error. expect err: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseRecording, res)
			}
			if tt.expectErr && res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}
