package recordinghandler

import (
	"context"
	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	smfile "monorepo/bin-storage-manager/models/file"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_storeRecordingFiles(t *testing.T) {

	tests := []struct {
		name string

		recording *recording.Recording

		responseRecording *recording.Recording
	}{
		{
			name: "normal",

			recording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8bf246d8-1d54-11ef-bf58-6bc042955973"),
					CustomerID: uuid.FromStringOrNil("8c80974e-1d54-11ef-929a-4fe8843edba5"),
				},
				Filenames: []string{
					"call_2abd6900-3f33-4a9f-8241-7ca7a0050e15_2024-03-29T21:50:32Z_in.wav",
					"call_2abd6900-3f33-4a9f-8241-7ca7a0050e15_2024-03-29T21:50:32Z_out.wav",
				},
			},

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("84df7daa-8eb9-11ed-b16e-4b8732219a4e"),
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

			h := &recordingHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AstProxyRecordingFileMove(ctx, tt.recording.AsteriskID, tt.recording.Filenames).Return(nil)

			for _, filename := range tt.recording.Filenames {
				filepath := h.getFilepath(filename)
				mockReq.EXPECT().StorageV1FileCreate(
					ctx,
					tt.recording.CustomerID,
					uuid.Nil,
					smfile.ReferenceTypeRecording,
					tt.recording.ID,
					gomock.Any(),
					gomock.Any(),
					filename,
					defaultBucketName,
					filepath,
					30000,
				).Return(&smfile.File{}, nil)
			}

			if errFile := h.storeRecordingFiles(ctx, tt.recording); errFile != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errFile)
			}
		})
	}
}

func Test_Stopped(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseRecording *recording.Recording
	}{
		{
			name: "normal reference type call",

			id: uuid.FromStringOrNil("85e34fd6-8ff1-11ed-84bc-cf71e0ea8a60"),

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("85e34fd6-8ff1-11ed-84bc-cf71e0ea8a60"),
				},
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a89d0f2a-8ff2-11ed-98b5-a35c4608884b"),
				AsteriskID:    "42:01:0a:a4:00:03",
				ChannelIDs: []string{
					"9ce319be-8ff1-11ed-a60d-e354dfcdff50",
					"9d0b60c2-8ff1-11ed-aa1e-b31d5892bff8",
				},
				Filenames: []string{
					"call_a89d0f2a-8ff2-11ed-98b5-a35c4608884b_2024-03-29T21:50:32Z_in.wav",
					"call_a89d0f2a-8ff2-11ed-98b5-a35c4608884b_2024-03-29T21:50:32Z_out.wav",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &recordingHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().RecordingSetStatus(ctx, tt.id, recording.StatusEnded).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.id).Return(tt.responseRecording, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseRecording.CustomerID, recording.EventTypeRecordingFinished, tt.responseRecording)

			mockReq.EXPECT().AstProxyRecordingFileMove(ctx, tt.responseRecording.AsteriskID, tt.responseRecording.Filenames).Return(nil)

			for _, filename := range tt.responseRecording.Filenames {
				filepath := h.getFilepath(filename)
				mockReq.EXPECT().StorageV1FileCreate(
					gomock.Any(),
					tt.responseRecording.CustomerID,
					uuid.Nil,
					smfile.ReferenceTypeRecording,
					tt.responseRecording.ID,
					gomock.Any(),
					gomock.Any(),
					filename,
					defaultBucketName,
					filepath,
					gomock.Any(),
				).Return(&smfile.File{}, nil)
			}

			res, err := h.Stopped(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseRecording, res)
			}
		})
	}
}

func Test_Stop_referenceTypeCall(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseRecording *recording.Recording
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("85e34fd6-8ff1-11ed-84bc-cf71e0ea8a60"),

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("85e34fd6-8ff1-11ed-84bc-cf71e0ea8a60"),
				},
				ReferenceType: recording.ReferenceTypeCall,
				AsteriskID:    "42:01:0a:a4:00:03",
				ChannelIDs: []string{
					"9ce319be-8ff1-11ed-a60d-e354dfcdff50",
					"9d0b60c2-8ff1-11ed-aa1e-b31d5892bff8",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &recordingHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().RecordingGet(ctx, tt.id).Return(tt.responseRecording, nil)
			for _, channelID := range tt.responseRecording.ChannelIDs {
				mockReq.EXPECT().AstChannelHangup(ctx, tt.responseRecording.AsteriskID, channelID, ari.ChannelCauseNormalClearing, 0).Return(nil)
			}
			mockDB.EXPECT().RecordingSetStatus(ctx, tt.id, recording.StatusStopping).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.id).Return(tt.responseRecording, nil)

			res, err := h.Stop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseRecording, nil)
			}
		})
	}
}

func Test_Stop_referenceTypeConference(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseRecording *recording.Recording

		expectRecordingName string
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("f3a4776c-90c2-11ed-a1fe-df5a7d2fc896"),

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f3a4776c-90c2-11ed-a1fe-df5a7d2fc896"),
				},
				ReferenceType: recording.ReferenceTypeConfbridge,
				AsteriskID:    "42:01:0a:a4:00:03",
				RecordingName: "conference_f3eeac6a-90c2-11ed-9bad-9bc50d0bf273_2023-01-05T14:58:05Z",
			},

			expectRecordingName: "conference_f3eeac6a-90c2-11ed-9bad-9bc50d0bf273_2023-01-05T14:58:05Z_in",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &recordingHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().RecordingGet(ctx, tt.id).Return(tt.responseRecording, nil)
			mockReq.EXPECT().AstRecordingStop(ctx, tt.responseRecording.AsteriskID, tt.expectRecordingName).Return(nil)
			mockDB.EXPECT().RecordingSetStatus(ctx, tt.id, recording.StatusStopping).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.id).Return(tt.responseRecording, nil)

			res, err := h.Stop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseRecording, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseRecording, res)
			}
		})
	}
}
