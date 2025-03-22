package recordinghandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_recordingReferenceTypeCall(t *testing.T) {

	tests := []struct {
		name string

		referenceID  uuid.UUID
		format       recording.Format
		endOfSilence int
		endOfKey     string
		duration     int
		onEndFlowID  uuid.UUID

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
			name: "normal",

			referenceID:  uuid.FromStringOrNil("852def0e-f24a-11ed-845f-e32a849e7338"),
			format:       recording.FormatWAV,
			endOfSilence: 0,
			endOfKey:     "",
			duration:     0,
			onEndFlowID:  uuid.FromStringOrNil("770275b6-0540-11f0-bfce-430bc2d612b5"),

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("852def0e-f24a-11ed-845f-e32a849e7338"),
					CustomerID: uuid.FromStringOrNil("00deda0e-8fd7-11ed-ac78-13dc7fb65df3"),
				},
				ActiveFlowID: uuid.FromStringOrNil("5e7f87da-0663-11f0-a195-03f01494aa3c"),
				ChannelID:    "8e5c2a28-f24a-11ed-97f4-5f82e61f6239",
				Status:       call.StatusProgressing,
			},
			responseUUID:       uuid.FromStringOrNil("8e914000-f24a-11ed-b09f-879b31d16030"),
			responseCurTimeRFC: "2023-01-05T14:58:05Z",
			responseUUIDsChannelIDs: []uuid.UUID{
				uuid.FromStringOrNil("8ebcbdb6-f24a-11ed-a11f-43ae14bc23bf"),
				uuid.FromStringOrNil("a3c515c8-f24a-11ed-959a-6711635061dd"),
			},
			responseChannels: []*channel.Channel{
				{
					ID:         "8ebcbdb6-f24a-11ed-a11f-43ae14bc23bf",
					AsteriskID: "42:01:0a:a4:00:03",
				},
				{
					ID:         "a3c515c8-f24a-11ed-959a-6711635061dd",
					AsteriskID: "42:01:0a:a4:00:03",
				},
			},

			expectAsteriskID:      "42:01:0a:a4:00:03",
			expectTargetChannelID: "8e5c2a28-f24a-11ed-97f4-5f82e61f6239",
			expectChannelIDs: []string{
				"8ebcbdb6-f24a-11ed-a11f-43ae14bc23bf",
				"a3c515c8-f24a-11ed-959a-6711635061dd",
			},
			expectArgs: []string{
				"context_type=call,context=call-record,reference_type=call,reference_id=852def0e-f24a-11ed-845f-e32a849e7338,recording_id=8e914000-f24a-11ed-b09f-879b31d16030,recording_name=call_852def0e-f24a-11ed-845f-e32a849e7338_2023-01-05T14:58:05Z,recording_direction=in,recording_format=wav,recording_end_of_silence=0,recording_end_of_key=,recording_duration=0",
				"context_type=call,context=call-record,reference_type=call,reference_id=852def0e-f24a-11ed-845f-e32a849e7338,recording_id=8e914000-f24a-11ed-b09f-879b31d16030,recording_name=call_852def0e-f24a-11ed-845f-e32a849e7338_2023-01-05T14:58:05Z,recording_direction=out,recording_format=wav,recording_end_of_silence=0,recording_end_of_key=,recording_duration=0",
			},
			expectRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8e914000-f24a-11ed-b09f-879b31d16030"),
					CustomerID: uuid.FromStringOrNil("00deda0e-8fd7-11ed-ac78-13dc7fb65df3"),
				},
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("852def0e-f24a-11ed-845f-e32a849e7338"),
				Status:        recording.StatusInitiating,
				Format:        "wav",

				OnEndFlowID: uuid.FromStringOrNil("770275b6-0540-11f0-bfce-430bc2d612b5"),

				RecordingName: "call_852def0e-f24a-11ed-845f-e32a849e7338_2023-01-05T14:58:05Z",
				Filenames: []string{
					"call_852def0e-f24a-11ed-845f-e32a849e7338_2023-01-05T14:58:05Z_in.wav",
					"call_852def0e-f24a-11ed-845f-e32a849e7338_2023-01-05T14:58:05Z_out.wav",
				},
				AsteriskID: "42:01:0a:a4:00:03",
				ChannelIDs: []string{
					"8ebcbdb6-f24a-11ed-a11f-43ae14bc23bf",
					"a3c515c8-f24a-11ed-959a-6711635061dd",
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

			// variableUpdateToReferenceInfo
			mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseCall.ActiveFlowID, gomock.Any()).Return(nil)

			res, err := h.recordingReferenceTypeCall(ctx, tt.referenceID, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration, tt.onEndFlowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRecording) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRecording, res)
			}
		})
	}
}

func Test_recordingReferenceTypeConfbridge(t *testing.T) {

	tests := []struct {
		name string

		referenceID  uuid.UUID
		format       recording.Format
		endOfSilence int
		endOfKey     string
		duration     int
		onEndflowID  uuid.UUID

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

			referenceID:  uuid.FromStringOrNil("4eb0b00a-f24b-11ed-8ceb-9f5eb3969704"),
			format:       recording.FormatWAV,
			endOfSilence: 0,
			endOfKey:     "",
			duration:     0,
			onEndflowID:  uuid.FromStringOrNil("773066c4-0540-11f0-ac8f-6f1699fafec8"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4eb0b00a-f24b-11ed-8ceb-9f5eb3969704"),
					CustomerID: uuid.FromStringOrNil("fff4ad02-98f6-11ed-aa9b-4f84a05324f1"),
				},
				BridgeID: "4ee52ba0-f24b-11ed-a01d-f77eee7d92ee",
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseBridge: &bridge.Bridge{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "4ee52ba0-f24b-11ed-a01d-f77eee7d92ee",
			},
			responseUUID:       uuid.FromStringOrNil("4f1ccb00-f24b-11ed-8dc1-6752696fc7aa"),
			responseCurTimeRFC: "2023-01-05T14:58:05Z",
			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f1ccb00-f24b-11ed-8dc1-6752696fc7aa"),
				},
			},

			expectFilename: "confbridge_4eb0b00a-f24b-11ed-8ceb-9f5eb3969704_2023-01-05T14:58:05Z_in",
			expectRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4f1ccb00-f24b-11ed-8dc1-6752696fc7aa"),
					CustomerID: uuid.FromStringOrNil("fff4ad02-98f6-11ed-aa9b-4f84a05324f1"),
				},
				ReferenceType: recording.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("4eb0b00a-f24b-11ed-8ceb-9f5eb3969704"),
				Status:        recording.StatusInitiating,
				Format:        "wav",

				OnEndFlowID:   uuid.FromStringOrNil("773066c4-0540-11f0-ac8f-6f1699fafec8"),
				RecordingName: "confbridge_4eb0b00a-f24b-11ed-8ceb-9f5eb3969704_2023-01-05T14:58:05Z",
				Filenames: []string{
					"confbridge_4eb0b00a-f24b-11ed-8ceb-9f5eb3969704_2023-01-05T14:58:05Z_in.wav",
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

			res, err := h.recordingReferenceTypeConfbridge(ctx, tt.referenceID, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration, tt.onEndflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRecording) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseRecording, res)
			}
		})
	}
}
