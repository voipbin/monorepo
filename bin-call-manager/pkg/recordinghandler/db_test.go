package recordinghandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		responseRecordings []*recording.Recording
	}{
		{
			name: "normal reference type call",

			size:  10,
			token: "2020-05-03%2021:35:02.809",
			filters: map[string]string{
				"customer_id": "fc5f8d06-8ff0-11ed-b07c-2776de9bed19",
			},

			responseRecordings: []*recording.Recording{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("22bc0808-8ff1-11ed-8f17-1f43c39a199e"),
					},
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

			mockDB.EXPECT().RecordingGets(ctx, tt.size, tt.token, gomock.Any()).Return(tt.responseRecordings, nil)
			res, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecordings) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseRecordings, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		recordingID uuid.UUID

		responseRecording *recording.Recording
		responseFiles     []smfile.File
		expectFilers      map[string]string
	}{
		{
			name: "normal",

			recordingID: uuid.FromStringOrNil("84df7daa-8eb9-11ed-b16e-4b8732219a4e"),

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("84df7daa-8eb9-11ed-b16e-4b8732219a4e"),
				},
			},
			responseFiles: []smfile.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("24c0bcb0-1d5e-11ef-a361-a3671e0f4f3a"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2aeb1392-1d5e-11ef-b320-cfcb1378a1ad"),
					},
				},
			},
			expectFilers: map[string]string{
				"reference_type": string(smfile.ReferenceTypeRecording),
				"reference_id":   "84df7daa-8eb9-11ed-b16e-4b8732219a4e",
				"deleted":        "false",
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

			mockDB.EXPECT().RecordingDelete(ctx, tt.recordingID).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.recordingID).Return(tt.responseRecording, nil)

			mockReq.EXPECT().StorageV1FileGets(gomock.Any(), "", uint64(1000), tt.expectFilers).Return(tt.responseFiles, nil)
			for _, f := range tt.responseFiles {
				mockReq.EXPECT().StorageV1FileDelete(ctx, f.ID, 60000).Return(&smfile.File{}, nil)
			}

			res, err := h.Delete(ctx, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseRecording, res)
			}
		})
	}
}

func Test_deleteRecordingFiles(t *testing.T) {

	tests := []struct {
		name string

		recording *recording.Recording

		responseRecording *recording.Recording

		responseFiles []smfile.File

		expectFilters map[string]string
	}{
		{
			name: "normal",

			recording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1e8bcc64-1d5d-11ef-9738-7bc321400c35"),
				},
			},

			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("84df7daa-8eb9-11ed-b16e-4b8732219a4e"),
				},
			},

			responseFiles: []smfile.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1f076c2a-1d5d-11ef-a4dd-bbf538c3b5b4"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1f2f2954-1d5d-11ef-b10a-c3d4990f73b0"),
					},
				},
			},
			expectFilters: map[string]string{
				"reference_type": string(smfile.ReferenceTypeRecording),
				"reference_id":   "1e8bcc64-1d5d-11ef-9738-7bc321400c35",
				"deleted":        "false",
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

			mockReq.EXPECT().StorageV1FileGets(gomock.Any(), "", uint64(1000), tt.expectFilters).Return(tt.responseFiles, nil)
			for _, f := range tt.responseFiles {
				mockReq.EXPECT().StorageV1FileDelete(gomock.Any(), f.ID, 60000).Return(&smfile.File{}, nil)
			}

			h.deleteRecordingFiles(tt.recording)
		})
	}
}
