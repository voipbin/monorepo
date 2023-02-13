package conferencehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
)

func Test_RecordingStart(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConference *conference.Conference
		responseRecording  *cmrecording.Recording
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("5aee9cbe-90fe-11ed-86b9-f36325211a1b"),

			responseConference: &conference.Conference{
				ID:     uuid.FromStringOrNil("5aee9cbe-90fe-11ed-86b9-f36325211a1b"),
				Type:   conference.TypeConference,
				Status: conference.StatusProgressing,
			},
			responseRecording: &cmrecording.Recording{
				ID: uuid.FromStringOrNil("5b30c45e-90fe-11ed-9b81-0fa580b2847d"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockReq.EXPECT().CallV1RecordingStart(ctx, cmrecording.ReferenceTypeConfbridge, tt.responseConference.ConfbridgeID, cmrecording.FormatWAV, 0, "", defaultRecordingTimeout).Return(tt.responseRecording, nil)
			mockDB.EXPECT().ConferenceSetRecordingID(ctx, tt.id, tt.responseRecording.ID).Return(nil)
			mockDB.EXPECT().ConferenceAddRecordingIDs(ctx, tt.id, tt.responseRecording.ID).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)

			res, err := h.RecordingStart(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConference, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}

func Test_RecordingStop(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConference *conference.Conference
		responseRecording  *cmrecording.Recording
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("56c9e61a-90ff-11ed-83a0-637ed1b2ef21"),

			responseConference: &conference.Conference{
				ID:          uuid.FromStringOrNil("56c9e61a-90ff-11ed-83a0-637ed1b2ef21"),
				Type:        conference.TypeConference,
				Status:      conference.StatusProgressing,
				RecordingID: uuid.FromStringOrNil("57053ff8-90ff-11ed-8583-8feb357ff230"),
			},
			responseRecording: &cmrecording.Recording{
				ID: uuid.FromStringOrNil("57053ff8-90ff-11ed-8583-8feb357ff230"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockReq.EXPECT().CallV1RecordingStop(ctx, tt.responseConference.RecordingID).Return(tt.responseRecording, nil)
			mockDB.EXPECT().ConferenceSetRecordingID(ctx, tt.id, uuid.Nil).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)

			res, err := h.RecordingStop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConference, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}
