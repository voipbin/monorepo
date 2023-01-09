package recordinghandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_Start(t *testing.T) {

	tests := []struct {
		name string

		referenceType recording.ReferenceType
		referenceID   uuid.UUID
		format        string
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
			format:        "wav",
			endOfSilence:  0,
			endOfKey:      "",
			duration:      0,

			responseCall: &call.Call{
				ID:         uuid.FromStringOrNil("d883a3f2-8fd4-11ed-baee-af9907e4df67"),
				CustomerID: uuid.FromStringOrNil("00deda0e-8fd7-11ed-ac78-13dc7fb65df3"),
				AsteriskID: "42:01:0a:a4:00:03",
				ChannelID:  "4f577092-8fd7-11ed-83c6-2fc653ad0b7c",
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
				"context=call-record,call_id=d883a3f2-8fd4-11ed-baee-af9907e4df67,recording_id=e141bb2c-8fd5-11ed-a0f9-9735e31b8411,recording_name=call_d883a3f2-8fd4-11ed-baee-af9907e4df67_2023-01-05T14:58:05Z,direction=in,format=wav,end_of_silence=0,end_of_key=,duration=0",
				"context=call-record,call_id=d883a3f2-8fd4-11ed-baee-af9907e4df67,recording_id=e141bb2c-8fd5-11ed-a0f9-9735e31b8411,recording_name=call_d883a3f2-8fd4-11ed-baee-af9907e4df67_2023-01-05T14:58:05Z,direction=out,format=wav,end_of_silence=0,end_of_key=,duration=0",
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
