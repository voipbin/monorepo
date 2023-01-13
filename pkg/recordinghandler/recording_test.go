package recordinghandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cmconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
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
				AsteriskID: "42:01:0a:a4:00:03",
				ChannelID:  "4f577092-8fd7-11ed-83c6-2fc653ad0b7c",
				Status:     call.StatusProgressing,
			},
			responseUUID:       uuid.FromStringOrNil("e141bb2c-8fd5-11ed-a0f9-9735e31b8411"),
			responseCurTimeRFC: "2023-01-05T14:58:05Z",
			responseUUIDsChannelIDs: []uuid.UUID{
				uuid.FromStringOrNil("4acf57a2-8fd6-11ed-98e5-6f1e54343269"),
				uuid.FromStringOrNil("4af70a04-8fd6-11ed-a294-4b2bc83c1829"),
			},

			expectAsteriskID:      "42:01:0a:a4:00:03",
			expectTargetChannelID: "4f577092-8fd7-11ed-83c6-2fc653ad0b7c",
			expectChannelIDs: []string{
				"4acf57a2-8fd6-11ed-98e5-6f1e54343269",
				"4af70a04-8fd6-11ed-a294-4b2bc83c1829",
			},
			expectArgs: []string{
				"context=call-record,reference_type=call,reference_id=d883a3f2-8fd4-11ed-baee-af9907e4df67,recording_id=e141bb2c-8fd5-11ed-a0f9-9735e31b8411,recording_name=call_d883a3f2-8fd4-11ed-baee-af9907e4df67_2023-01-05T14:58:05Z,direction=in,format=wav,end_of_silence=0,end_of_key=,duration=0",
				"context=call-record,reference_type=call,reference_id=d883a3f2-8fd4-11ed-baee-af9907e4df67,recording_id=e141bb2c-8fd5-11ed-a0f9-9735e31b8411,recording_name=call_d883a3f2-8fd4-11ed-baee-af9907e4df67_2023-01-05T14:58:05Z,direction=out,format=wav,end_of_silence=0,end_of_key=,duration=0",
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

			h := &recordingHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)
			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockUtil.EXPECT().GetCurTimeRFC3339().Return(tt.responseCurTimeRFC)
			for i, direction := range []channel.SnoopDirection{channel.SnoopDirectionIn, channel.SnoopDirectionOut} {
				mockUtil.EXPECT().CreateUUID().Return(tt.responseUUIDsChannelIDs[i])
				mockReq.EXPECT().AstChannelCreateSnoop(ctx, tt.expectAsteriskID, tt.expectTargetChannelID, tt.expectChannelIDs[i], tt.expectArgs[i], direction, channel.SnoopDirectionNone).Return(&channel.Channel{}, nil)
			}

			mockDB.EXPECT().RecordingCreate(ctx, tt.expectRecording).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.expectRecording.ID).Return(tt.expectRecording, nil)

			mockReq.EXPECT().CallV1CallSetRecordingID(ctx, tt.referenceID, tt.expectRecording.ID).Return(&call.Call{}, nil)

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

func Test_Start_conference(t *testing.T) {

	tests := []struct {
		name string

		referenceType recording.ReferenceType
		referenceID   uuid.UUID
		format        recording.Format
		endOfSilence  int
		endOfKey      string
		duration      int

		responseConference *cmconference.Conference
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

			referenceType: recording.ReferenceTypeConference,
			referenceID:   uuid.FromStringOrNil("676d83d0-90a2-11ed-96be-afdeb5b65a08"),
			format:        recording.FormatWAV,
			endOfSilence:  0,
			endOfKey:      "",
			duration:      0,

			responseConference: &cmconference.Conference{
				ID:           uuid.FromStringOrNil("676d83d0-90a2-11ed-96be-afdeb5b65a08"),
				CustomerID:   uuid.FromStringOrNil("67c29f96-90a2-11ed-ae2a-cf22d50cd411"),
				ConfbridgeID: uuid.FromStringOrNil("67f358e8-90a2-11ed-b315-2b63c5f83d10"),
				Status:       cmconference.StatusProgressing,
			},
			responseConfbridge: &confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("67f358e8-90a2-11ed-b315-2b63c5f83d10"),
				BridgeID: "6822e4c8-90a2-11ed-8002-4bf0087d99cb",
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

			expectFilename: "conference_676d83d0-90a2-11ed-96be-afdeb5b65a08_2023-01-05T14:58:05Z_in",
			expectRecording: &recording.Recording{
				ID:            uuid.FromStringOrNil("6856ed5e-90a2-11ed-8f4e-6353d1a3e50b"),
				CustomerID:    uuid.FromStringOrNil("67c29f96-90a2-11ed-ae2a-cf22d50cd411"),
				ReferenceType: recording.ReferenceTypeConference,
				ReferenceID:   uuid.FromStringOrNil("676d83d0-90a2-11ed-96be-afdeb5b65a08"),
				Status:        recording.StatusInitiating,
				Format:        "wav",
				RecordingName: "conference_676d83d0-90a2-11ed-96be-afdeb5b65a08_2023-01-05T14:58:05Z",
				Filenames: []string{
					"conference_676d83d0-90a2-11ed-96be-afdeb5b65a08_2023-01-05T14:58:05Z_in.wav",
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
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &recordingHandler{
				utilHandler:       mockUtil,
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				confbridgeHandler: mockConfbridge,
				bridgeHandler:     mockBridge,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.referenceID).Return(tt.responseConference, nil)
			mockConfbridge.EXPECT().Get(ctx, tt.responseConference.ConfbridgeID).Return(tt.responseConfbridge, nil)
			mockBridge.EXPECT().Get(ctx, tt.responseConfbridge.BridgeID).Return(tt.responseBridge, nil)
			mockUtil.EXPECT().GetCurTimeRFC3339().Return(tt.responseCurTimeRFC)
			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDB.EXPECT().RecordingCreate(ctx, tt.expectRecording).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.expectRecording.ID).Return(tt.responseRecording, nil)
			mockReq.EXPECT().ConferenceV1ConferenceUpdateRecordingID(ctx, tt.responseRecording.ReferenceID, tt.responseRecording.ID).Return(tt.responseConference, nil)
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

func Test_GetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string

		responseRecordings []*recording.Recording
	}{
		{
			name: "normal reference type call",

			customerID: uuid.FromStringOrNil("fc5f8d06-8ff0-11ed-b07c-2776de9bed19"),
			size:       10,
			token:      "2020-05-03%2021:35:02.809",

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

			mockDB.EXPECT().RecordingGets(ctx, tt.customerID, tt.size, tt.token).Return(tt.responseRecordings, nil)
			res, err := h.GetsByCustomerID(ctx, tt.customerID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecordings) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseRecordings, res)
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
				ID:            uuid.FromStringOrNil("85e34fd6-8ff1-11ed-84bc-cf71e0ea8a60"),
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
				ID:            uuid.FromStringOrNil("f3a4776c-90c2-11ed-a1fe-df5a7d2fc896"),
				ReferenceType: recording.ReferenceTypeConference,
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
				ID:            uuid.FromStringOrNil("85e34fd6-8ff1-11ed-84bc-cf71e0ea8a60"),
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

			mockDB.EXPECT().RecordingSetStatus(ctx, tt.id, recording.StatusEnded).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.id).Return(tt.responseRecording, nil)
			mockReq.EXPECT().CallV1CallSetRecordingID(ctx, tt.responseRecording.ReferenceID, uuid.Nil).Return(&call.Call{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseRecording.CustomerID, recording.EventTypeRecordingFinished, tt.responseRecording)

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

func Test_Delete(t *testing.T) {

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

			h := &recordingHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().RecordingDelete(ctx, tt.recordingID).Return(nil)
			mockReq.EXPECT().StorageV1RecordingDelete(ctx, tt.recordingID).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.recordingID).Return(tt.responseRecording, nil)

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
