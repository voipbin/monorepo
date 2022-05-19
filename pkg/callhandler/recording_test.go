package callhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_RecordingGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageSize   uint64
		pageToken  string

		responseRecordings []*recording.Recording
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("6c3081c2-d722-11ec-bbc7-3fc3a0bf0ad1"),
			pageSize:   10,
			pageToken:  "2021-08-22 04:10:10.331",

			responseRecordings: []*recording.Recording{
				{
					ID: uuid.FromStringOrNil("c0110626-d723-11ec-b68a-578b56d56866"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().RecordingGets(ctx, tt.customerID, tt.pageSize, tt.pageToken).Return(tt.responseRecordings, nil)

			res, err := h.RecordingGets(ctx, tt.customerID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecordings) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseRecordings, res)
			}
		})
	}
}

func Test_RecordingGet(t *testing.T) {

	tests := []struct {
		name string

		recordingID uuid.UUID

		responseRecording *recording.Recording
	}{
		{
			name: "normal",

			recordingID: uuid.FromStringOrNil("65eeaf30-d724-11ec-be72-7f1f2f384f4c"),

			responseRecording: &recording.Recording{

				ID: uuid.FromStringOrNil("65eeaf30-d724-11ec-be72-7f1f2f384f4c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().RecordingGet(ctx, tt.recordingID).Return(tt.responseRecording, nil)

			res, err := h.RecordingGet(ctx, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseRecording, res)
			}
		})
	}
}
