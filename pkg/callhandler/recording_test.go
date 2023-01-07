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

func Test_RecordingCreate(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		customerID    uuid.UUID
		referenceType recording.ReferenceType
		referenceID   uuid.UUID
		format        string
		recordingName string
		filenames     []string
		asteriskID    string
		channelIDs    []string

		responseCurTime   string
		responseRecording *recording.Recording

		expectRecording *recording.Recording
	}{
		{
			name: "normal",

			id:            uuid.FromStringOrNil("cc43c270-8eb7-11ed-a74c-97e0729ae677"),
			customerID:    uuid.FromStringOrNil("6c3081c2-d722-11ec-bbc7-3fc3a0bf0ad1"),
			referenceType: recording.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("cc78cbaa-8eb7-11ed-979c-3b92d111ac51"),
			format:        "wav",
			recordingName: "call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z",
			filenames: []string{
				"call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_in.wav",
				"call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_out.wav",
			},
			asteriskID: "3e:50:6b:43:bb:30",
			channelIDs: []string{
				"cc9bb534-8eb7-11ed-af1c-3798bbe09c2d",
				"ccc14d58-8eb7-11ed-a2df-936850ad798b",
			},

			responseCurTime: "2020-04-18T03:22:17.995000",
			responseRecording: &recording.Recording{
				ID: uuid.FromStringOrNil("cc43c270-8eb7-11ed-a74c-97e0729ae677"),
			},

			expectRecording: &recording.Recording{
				ID:            uuid.FromStringOrNil("cc43c270-8eb7-11ed-a74c-97e0729ae677"),
				CustomerID:    uuid.FromStringOrNil("6c3081c2-d722-11ec-bbc7-3fc3a0bf0ad1"),
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("cc78cbaa-8eb7-11ed-979c-3b92d111ac51"),
				Status:        recording.StatusInitiating,
				Format:        "wav",
				RecordingName: "call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z",
				Filenames: []string{
					"call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_in.wav",
					"call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_out.wav",
				},
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelIDs: []string{
					"cc9bb534-8eb7-11ed-af1c-3798bbe09c2d",
					"ccc14d58-8eb7-11ed-a2df-936850ad798b",
				},
				TMStart: dbhandler.DefaultTimeStamp,
				TMEnd:   dbhandler.DefaultTimeStamp,
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

			mockDB.EXPECT().RecordingCreate(ctx, tt.expectRecording).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.id).Return(tt.responseRecording, nil)

			res, err := h.RecordingCreate(ctx, tt.id, tt.customerID, tt.referenceType, tt.referenceID, tt.format, tt.recordingName, tt.filenames, tt.asteriskID, tt.channelIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseRecording, res)
			}
		})
	}
}

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

func Test_RecordingDelete(t *testing.T) {

	tests := []struct {
		name string

		recordingID uuid.UUID

		responseRecording *recording.Recording
	}{
		{
			name: "normal",

			recordingID: uuid.FromStringOrNil("84df7daa-8eb9-11ed-b16e-4b8732219a4e"),

			responseRecording: &recording.Recording{
				ID: uuid.FromStringOrNil("84df7daa-8eb9-11ed-b16e-4b8732219a4e"),
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

			mockDB.EXPECT().RecordingDelete(ctx, tt.recordingID).Return(nil)
			mockReq.EXPECT().StorageV1RecordingDelete(ctx, tt.recordingID).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.recordingID).Return(tt.responseRecording, nil)

			res, err := h.RecordingDelete(ctx, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseRecording, res)
			}
		})
	}
}
