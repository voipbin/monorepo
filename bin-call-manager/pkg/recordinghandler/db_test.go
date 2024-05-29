package recordinghandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Start_call(t *testing.T) {

	tests := []struct {
		name string

		referenceType recording.ReferenceType
		referenceID   uuid.UUID
		format        recording.Format
		endOfSilence  int
		endOfKey      string
		duration      int

		responseCall            *call.Call
		responseUUID            uuid.UUID
		responseCurTimeRFC      string
		responseUUIDsChannelIDs []uuid.UUID
		responseChannels        []*channel.Channel

		expectAsteriskID      string
		expectTargetChannelID string
		expectChannelIDs      []string
		expectArgs            []string
		expectRecording       *recording.Recording
	}{
		{
			name: "normal reference type call",

			referenceType: recording.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("d883a3f2-8fd4-11ed-baee-af9907e4df67"),
			format:        recording.FormatWAV,
			endOfSilence:  0,
			endOfKey:      "",
			duration:      0,

			responseCall: &call.Call{
				ID:         uuid.FromStringOrNil("d883a3f2-8fd4-11ed-baee-af9907e4df67"),
				CustomerID: uuid.FromStringOrNil("00deda0e-8fd7-11ed-ac78-13dc7fb65df3"),
				ChannelID:  "4f577092-8fd7-11ed-83c6-2fc653ad0b7c",
				Status:     call.StatusProgressing,
			},
			responseUUID:       uuid.FromStringOrNil("e141bb2c-8fd5-11ed-a0f9-9735e31b8411"),
			responseCurTimeRFC: "2023-01-05T14:58:05Z",
			responseUUIDsChannelIDs: []uuid.UUID{
				uuid.FromStringOrNil("4acf57a2-8fd6-11ed-98e5-6f1e54343269"),
				uuid.FromStringOrNil("4af70a04-8fd6-11ed-a294-4b2bc83c1829"),
			},
			responseChannels: []*channel.Channel{
				{
					ID:         "4acf57a2-8fd6-11ed-98e5-6f1e54343269",
					AsteriskID: "42:01:0a:a4:00:03",
				},
				{
					ID:         "4af70a04-8fd6-11ed-a294-4b2bc83c1829",
					AsteriskID: "42:01:0a:a4:00:03",
				},
			},

			expectAsteriskID:      "42:01:0a:a4:00:03",
			expectTargetChannelID: "4f577092-8fd7-11ed-83c6-2fc653ad0b7c",
			expectChannelIDs: []string{
				"4acf57a2-8fd6-11ed-98e5-6f1e54343269",
				"4af70a04-8fd6-11ed-a294-4b2bc83c1829",
			},
			expectArgs: []string{
				"context_type=call,context=call-record,reference_type=call,reference_id=d883a3f2-8fd4-11ed-baee-af9907e4df67,recording_id=e141bb2c-8fd5-11ed-a0f9-9735e31b8411,recording_name=call_d883a3f2-8fd4-11ed-baee-af9907e4df67_2023-01-05T14:58:05Z,recording_direction=in,recording_format=wav,recording_end_of_silence=0,recording_end_of_key=,recording_duration=0",
				"context_type=call,context=call-record,reference_type=call,reference_id=d883a3f2-8fd4-11ed-baee-af9907e4df67,recording_id=e141bb2c-8fd5-11ed-a0f9-9735e31b8411,recording_name=call_d883a3f2-8fd4-11ed-baee-af9907e4df67_2023-01-05T14:58:05Z,recording_direction=out,recording_format=wav,recording_end_of_silence=0,recording_end_of_key=,recording_duration=0",
			},
			expectRecording: &recording.Recording{
				ID:            uuid.FromStringOrNil("e141bb2c-8fd5-11ed-a0f9-9735e31b8411"),
				CustomerID:    uuid.FromStringOrNil("00deda0e-8fd7-11ed-ac78-13dc7fb65df3"),
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("d883a3f2-8fd4-11ed-baee-af9907e4df67"),
				Status:        recording.StatusInitiating,
				Format:        "wav",
				RecordingName: "call_d883a3f2-8fd4-11ed-baee-af9907e4df67_2023-01-05T14:58:05Z",
				Filenames: []string{
					"call_d883a3f2-8fd4-11ed-baee-af9907e4df67_2023-01-05T14:58:05Z_in.wav",
					"call_d883a3f2-8fd4-11ed-baee-af9907e4df67_2023-01-05T14:58:05Z_out.wav",
				},
				AsteriskID: "42:01:0a:a4:00:03",
				ChannelIDs: []string{
					"4acf57a2-8fd6-11ed-98e5-6f1e54343269",
					"4af70a04-8fd6-11ed-a294-4b2bc83c1829",
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

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &recordingHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeGetCurTimeRFC3339().Return(tt.responseCurTimeRFC)
			for i, direction := range []channel.SnoopDirection{channel.SnoopDirectionIn, channel.SnoopDirectionOut} {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDsChannelIDs[i])
				mockChannel.EXPECT().StartSnoop(ctx, tt.expectTargetChannelID, tt.expectChannelIDs[i], tt.expectArgs[i], direction, channel.SnoopDirectionNone).Return(tt.responseChannels[i], nil)
			}

			mockDB.EXPECT().RecordingCreate(ctx, tt.expectRecording).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.expectRecording.ID).Return(tt.expectRecording, nil)

			res, err := h.Start(
				ctx,
				tt.referenceType,
				tt.referenceID,
				tt.format,
				tt.endOfSilence,
				tt.endOfKey,
				tt.duration,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRecording) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRecording, res)
			}
		})
	}
}

func Test_Start_confbridge(t *testing.T) {

	tests := []struct {
		name string

		referenceType recording.ReferenceType
		referenceID   uuid.UUID
		format        recording.Format
		endOfSilence  int
		endOfKey      string
		duration      int

		responseConfbridge *confbridge.Confbridge
		responseBridge     *bridge.Bridge
		responseUUID       uuid.UUID
		responseRecording  *recording.Recording

		responseCurTimeRFC string

		expectFilename  string
		expectRecording *recording.Recording
	}{
		{
			name: "normal",

			referenceType: recording.ReferenceTypeConfbridge,
			referenceID:   uuid.FromStringOrNil("67f358e8-90a2-11ed-b315-2b63c5f83d10"),
			format:        recording.FormatWAV,
			endOfSilence:  0,
			endOfKey:      "",
			duration:      0,

			responseConfbridge: &confbridge.Confbridge{
				ID:         uuid.FromStringOrNil("67f358e8-90a2-11ed-b315-2b63c5f83d10"),
				CustomerID: uuid.FromStringOrNil("fff4ad02-98f6-11ed-aa9b-4f84a05324f1"),
				BridgeID:   "6822e4c8-90a2-11ed-8002-4bf0087d99cb",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseBridge: &bridge.Bridge{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "6822e4c8-90a2-11ed-8002-4bf0087d99cb",
			},
			responseUUID:       uuid.FromStringOrNil("6856ed5e-90a2-11ed-8f4e-6353d1a3e50b"),
			responseCurTimeRFC: "2023-01-05T14:58:05Z",
			responseRecording: &recording.Recording{
				ID: uuid.FromStringOrNil("6856ed5e-90a2-11ed-8f4e-6353d1a3e50b"),
			},

			expectFilename: "confbridge_67f358e8-90a2-11ed-b315-2b63c5f83d10_2023-01-05T14:58:05Z_in",
			expectRecording: &recording.Recording{
				ID:            uuid.FromStringOrNil("6856ed5e-90a2-11ed-8f4e-6353d1a3e50b"),
				CustomerID:    uuid.FromStringOrNil("fff4ad02-98f6-11ed-aa9b-4f84a05324f1"),
				ReferenceType: recording.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("67f358e8-90a2-11ed-b315-2b63c5f83d10"),
				Status:        recording.StatusInitiating,
				Format:        "wav",
				RecordingName: "confbridge_67f358e8-90a2-11ed-b315-2b63c5f83d10_2023-01-05T14:58:05Z",
				Filenames: []string{
					"confbridge_67f358e8-90a2-11ed-b315-2b63c5f83d10_2023-01-05T14:58:05Z_in.wav",
				},
				AsteriskID: "42:01:0a:a4:00:03",
				TMStart:    dbhandler.DefaultTimeStamp,
				TMEnd:      dbhandler.DefaultTimeStamp,
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
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &recordingHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				bridgeHandler: mockBridge,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.referenceID).Return(tt.responseConfbridge, nil)
			mockBridge.EXPECT().Get(ctx, tt.responseConfbridge.BridgeID).Return(tt.responseBridge, nil)
			mockUtil.EXPECT().TimeGetCurTimeRFC3339().Return(tt.responseCurTimeRFC)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().RecordingCreate(ctx, tt.expectRecording).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.expectRecording.ID).Return(tt.responseRecording, nil)
			mockReq.EXPECT().AstBridgeRecord(
				ctx,
				tt.responseBridge.AsteriskID,
				tt.responseBridge.ID,
				tt.expectFilename,
				string(tt.format),
				tt.duration,
				tt.endOfSilence,
				false,
				tt.endOfKey,
				"fail",
			)

			res, err := h.Start(
				ctx,
				tt.referenceType,
				tt.referenceID,
				tt.format,
				tt.endOfSilence,
				tt.endOfKey,
				tt.duration,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseRecording, res)
			}
		})
	}
}

func Test_Started(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCurTime   string
		responseRecording *recording.Recording
	}{
		{
			name: "normal reference type call",

			id: uuid.FromStringOrNil("def310b4-9011-11ed-bc02-ab675449097d"),

			responseCurTime: "2020-04-18 03:22:17.995000",
			responseRecording: &recording.Recording{
				ID:            uuid.FromStringOrNil("def310b4-9011-11ed-bc02-ab675449097d"),
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a89d0f2a-8ff2-11ed-98b5-a35c4608884b"),
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

			mockDB.EXPECT().RecordingSetStatus(ctx, tt.id, recording.StatusRecording).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.id).Return(tt.responseRecording, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseRecording.CustomerID, recording.EventTypeRecordingStarted, tt.responseRecording)

			res, err := h.Started(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseRecording, res)
			}
		})
	}
}

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
					ID: uuid.FromStringOrNil("22bc0808-8ff1-11ed-8f17-1f43c39a199e"),
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

			mockDB.EXPECT().RecordingGets(ctx, tt.size, tt.token, tt.filters).Return(tt.responseRecordings, nil)
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
				ID: uuid.FromStringOrNil("84df7daa-8eb9-11ed-b16e-4b8732219a4e"),
			},
			responseFiles: []smfile.File{
				{
					ID: uuid.FromStringOrNil("24c0bcb0-1d5e-11ef-a361-a3671e0f4f3a"),
				},
				{
					ID: uuid.FromStringOrNil("2aeb1392-1d5e-11ef-b320-cfcb1378a1ad"),
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
				ID: uuid.FromStringOrNil("1e8bcc64-1d5d-11ef-9738-7bc321400c35"),
			},

			responseRecording: &recording.Recording{
				ID: uuid.FromStringOrNil("84df7daa-8eb9-11ed-b16e-4b8732219a4e"),
			},

			responseFiles: []smfile.File{
				{
					ID: uuid.FromStringOrNil("1f076c2a-1d5d-11ef-a4dd-bbf538c3b5b4"),
				},
				{
					ID: uuid.FromStringOrNil("1f2f2954-1d5d-11ef-b10a-c3d4990f73b0"),
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
