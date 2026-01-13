package recordinghandler

import (
	"context"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Start_call(t *testing.T) {

	tests := []struct {
		name string

		activeflowID  uuid.UUID
		referenceType recording.ReferenceType
		referenceID   uuid.UUID
		format        recording.Format
		endOfSilence  int
		endOfKey      string
		duration      int
		onEndFlowID   uuid.UUID

		responseCall            *call.Call
		responseCallChannel     *channel.Channel
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

			activeflowID:  uuid.FromStringOrNil("24c83af4-0727-11f0-b906-27a506fc80f9"),
			referenceType: recording.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("d883a3f2-8fd4-11ed-baee-af9907e4df67"),
			format:        recording.FormatWAV,
			endOfSilence:  0,
			endOfKey:      "",
			duration:      0,
			onEndFlowID:   uuid.FromStringOrNil("56451fc2-0540-11f0-8f09-8b6c356cada7"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d883a3f2-8fd4-11ed-baee-af9907e4df67"),
					CustomerID: uuid.FromStringOrNil("00deda0e-8fd7-11ed-ac78-13dc7fb65df3"),
				},
				ActiveflowID: uuid.FromStringOrNil("885fe05e-0663-11f0-b231-fb801f78c0c3"),
				ChannelID:    "4f577092-8fd7-11ed-83c6-2fc653ad0b7c",
				Status:       call.StatusProgressing,
			},
			responseCallChannel: &channel.Channel{
				AsteriskID: "42:01:0a:a4:00:03",
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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e141bb2c-8fd5-11ed-a0f9-9735e31b8411"),
					CustomerID: uuid.FromStringOrNil("00deda0e-8fd7-11ed-ac78-13dc7fb65df3"),
				},
				ActiveflowID:  uuid.FromStringOrNil("24c83af4-0727-11f0-b906-27a506fc80f9"),
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("d883a3f2-8fd4-11ed-baee-af9907e4df67"),
				Status:        recording.StatusInitiating,
				Format:        "wav",

				OnEndFlowID: uuid.FromStringOrNil("56451fc2-0540-11f0-8f09-8b6c356cada7"),

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
			mockChannel.EXPECT().Get(ctx, tt.responseCall.ChannelID).Return(tt.responseCallChannel, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeGetCurTimeRFC3339().Return(tt.responseCurTimeRFC)
			for i, direction := range []channel.SnoopDirection{channel.SnoopDirectionIn, channel.SnoopDirectionOut} {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDsChannelIDs[i])
				mockChannel.EXPECT().StartSnoop(ctx, tt.expectTargetChannelID, tt.expectChannelIDs[i], tt.expectArgs[i], direction, channel.SnoopDirectionNone).Return(tt.responseChannels[i], nil)
			}

			mockDB.EXPECT().RecordingCreate(ctx, tt.expectRecording).Return(nil)
			mockDB.EXPECT().RecordingGet(ctx, tt.expectRecording.ID).Return(tt.expectRecording, nil)

			// variableUpdateToReferenceInfo
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, gomock.Any()).Return(nil)

			res, err := h.Start(
				ctx,
				tt.activeflowID,
				tt.referenceType,
				tt.referenceID,
				tt.format,
				tt.endOfSilence,
				tt.endOfKey,
				tt.duration,
				tt.onEndFlowID,
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

		activeflowID  uuid.UUID
		referenceType recording.ReferenceType
		referenceID   uuid.UUID
		format        recording.Format
		endOfSilence  int
		endOfKey      string
		duration      int
		onEndFlowID   uuid.UUID

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

			activeflowID:  uuid.FromStringOrNil("a1645462-0727-11f0-a511-4f58bbcc04ca"),
			referenceType: recording.ReferenceTypeConfbridge,
			referenceID:   uuid.FromStringOrNil("67f358e8-90a2-11ed-b315-2b63c5f83d10"),
			format:        recording.FormatWAV,
			endOfSilence:  0,
			endOfKey:      "",
			duration:      0,
			onEndFlowID:   uuid.FromStringOrNil("76d50b1c-0540-11f0-aeef-37900c5fcfeb"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("67f358e8-90a2-11ed-b315-2b63c5f83d10"),
					CustomerID: uuid.FromStringOrNil("fff4ad02-98f6-11ed-aa9b-4f84a05324f1"),
				},
				BridgeID: "6822e4c8-90a2-11ed-8002-4bf0087d99cb",
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseBridge: &bridge.Bridge{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "6822e4c8-90a2-11ed-8002-4bf0087d99cb",
			},
			responseUUID:       uuid.FromStringOrNil("6856ed5e-90a2-11ed-8f4e-6353d1a3e50b"),
			responseCurTimeRFC: "2023-01-05T14:58:05Z",
			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6856ed5e-90a2-11ed-8f4e-6353d1a3e50b"),
				},
			},

			expectFilename: "confbridge_67f358e8-90a2-11ed-b315-2b63c5f83d10_2023-01-05T14:58:05Z_in",
			expectRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6856ed5e-90a2-11ed-8f4e-6353d1a3e50b"),
					CustomerID: uuid.FromStringOrNil("fff4ad02-98f6-11ed-aa9b-4f84a05324f1"),
				},
				ActiveflowID:  uuid.FromStringOrNil("a1645462-0727-11f0-a511-4f58bbcc04ca"),
				ReferenceType: recording.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("67f358e8-90a2-11ed-b315-2b63c5f83d10"),
				Status:        recording.StatusInitiating,
				Format:        "wav",

				OnEndFlowID: uuid.FromStringOrNil("76d50b1c-0540-11f0-aeef-37900c5fcfeb"),

				RecordingName: "confbridge_67f358e8-90a2-11ed-b315-2b63c5f83d10_2023-01-05T14:58:05Z",
				Filenames: []string{
					"confbridge_67f358e8-90a2-11ed-b315-2b63c5f83d10_2023-01-05T14:58:05Z_in.wav",
				},
				AsteriskID: "42:01:0a:a4:00:03",
				ChannelIDs: []string{},
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
				tt.activeflowID,
				tt.referenceType,
				tt.referenceID,
				tt.format,
				tt.endOfSilence,
				tt.endOfKey,
				tt.duration,
				tt.onEndFlowID,
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("def310b4-9011-11ed-bc02-ab675449097d"),
				},
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
