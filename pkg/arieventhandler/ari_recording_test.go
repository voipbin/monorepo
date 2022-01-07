package arieventhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

func TestEventHandlerRecordingStarted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockSvc := callhandler.NewMockCallHandler(mc)

	h := eventHandler{
		db:            mockDB,
		rabbitSock:    mockSock,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		callHandler:   mockSvc,
	}

	tests := []struct {
		name      string
		event     *ari.RecordingStarted
		call      *call.Call
		recording *recording.Recording
		timestamp string
	}{
		{
			"normal",
			&ari.RecordingStarted{
				Event: ari.Event{
					Type:        ari.EventTypeRecordingStarted,
					Application: "voipbin",
					Timestamp:   "2020-02-10T13:08:18.888",
					AsteriskID:  "42:01:0a:84:00:12",
				},
				Recording: ari.RecordingLive{
					Name:            "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z",
					Format:          "wav",
					State:           "recording",
					SilenceDuration: 0,
					Duration:        0,
					TalkingDuration: 0,
					TargetURI:       "channel:test_call",
				},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("e31efb5e-1f3c-11ec-beea-af98446e3b8e"),
				WebhookURI: "test.com/webhook",
			},
			&recording.Recording{
				ID:          uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937"),
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("e31efb5e-1f3c-11ec-beea-af98446e3b8e"),
				Filename:    "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z.wav",
			},
			"2020-02-10T13:08:18.888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().RecordingGetByFilename(gomock.Any(), tt.recording.Filename).Return(tt.recording, nil)
			mockDB.EXPECT().RecordingSetStatus(gomock.Any(), tt.recording.ID, recording.StatusRecording, tt.timestamp).Return(nil)
			mockDB.EXPECT().RecordingGet(gomock.Any(), tt.recording.ID).Return(tt.recording, nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.recording.ReferenceID).Return(tt.call, nil)
			mockNotify.EXPECT().NotifyEvent(gomock.Any(), notifyhandler.EventTypeRecordingStarted, tt.call.WebhookURI, tt.recording)

			if err := h.EventHandlerRecordingStarted(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerRecordingFinishedCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockSvc := callhandler.NewMockCallHandler(mc)

	h := eventHandler{
		db:            mockDB,
		rabbitSock:    mockSock,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		callHandler:   mockSvc,
	}

	tests := []struct {
		name      string
		event     *ari.RecordingFinished
		call      *call.Call
		recording *recording.Recording
		timestamp string
	}{
		{
			"normal",
			&ari.RecordingFinished{
				Event: ari.Event{
					Type:        ari.EventTypeRecordingFinished,
					Application: "voipbin",
					Timestamp:   "2020-02-10T13:08:40.888",
					AsteriskID:  "42:01:0a:84:00:12",
				},
				Recording: ari.RecordingLive{
					Name:            "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888",
					Format:          "wav",
					State:           "done",
					SilenceDuration: 0,
					Duration:        351,
					TalkingDuration: 0,
					TargetURI:       "channel:test_call",
				},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("3b16cef6-2b99-11eb-87eb-571ab4136611"),
				WebhookURI: "test.com/webhook",
			},
			&recording.Recording{
				ID:          uuid.FromStringOrNil("4f367e2c-612c-11eb-b063-676ca5ee546a"),
				Filename:    "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888.wav",
				Format:      "wav",
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("3b16cef6-2b99-11eb-87eb-571ab4136611"),
			},
			"2020-02-10T13:08:40.888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().RecordingGetByFilename(gomock.Any(), tt.recording.Filename).Return(tt.recording, nil)
			mockDB.EXPECT().RecordingSetStatus(gomock.Any(), tt.recording.ID, recording.StatusEnd, tt.timestamp).Return(nil)
			mockDB.EXPECT().CallSetRecordID(gomock.Any(), tt.recording.ReferenceID, uuid.Nil).Return(nil)
			mockDB.EXPECT().RecordingGet(gomock.Any(), tt.recording.ID).Return(tt.recording, nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.recording.ReferenceID).Return(tt.call, nil)
			mockNotify.EXPECT().NotifyEvent(gomock.Any(), notifyhandler.EventTypeRecordingFinished, tt.call.WebhookURI, tt.recording)

			if err := h.EventHandlerRecordingFinished(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerRecordingFinishedConference(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockSvc := callhandler.NewMockCallHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := eventHandler{
		db:            mockDB,
		rabbitSock:    mockSock,
		reqHandler:    mockReq,
		callHandler:   mockSvc,
		notifyHandler: mockNotify,
	}

	tests := []struct {
		name      string
		event     *ari.RecordingFinished
		call      *call.Call
		recording *recording.Recording
		timestamp string
	}{
		{
			"normal",
			&ari.RecordingFinished{
				Event: ari.Event{
					Type:        ari.EventTypeRecordingFinished,
					Application: "voipbin",
					Timestamp:   "2020-02-10T13:08:40.888",
					AsteriskID:  "42:01:0a:84:00:12",
				},
				Recording: ari.RecordingLive{
					Name:            "bridge_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888",
					Format:          "wav",
					State:           "done",
					SilenceDuration: 0,
					Duration:        351,
					TalkingDuration: 0,
					TargetURI:       "channel:test_call",
				},
			},
			&call.Call{
				ID: uuid.FromStringOrNil("037b88fe-1547-11ec-836c-235bd80d876e"),
			},
			&recording.Recording{
				ID:          uuid.FromStringOrNil("34585192-1546-11ec-b592-63304ff57c57"),
				Filename:    "something_filename.wav",
				Type:        recording.TypeConference,
				ReferenceID: uuid.FromStringOrNil("037b88fe-1547-11ec-836c-235bd80d876e"),
			},
			"2020-02-10T13:08:40.888",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().RecordingGetByFilename(gomock.Any(), gomock.Any()).Return(tt.recording, nil)
			mockDB.EXPECT().RecordingSetStatus(gomock.Any(), tt.recording.ID, recording.StatusEnd, tt.timestamp).Return(nil)
			mockDB.EXPECT().RecordingGet(gomock.Any(), tt.recording.ID).Return(tt.recording, nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.recording.ReferenceID).Return(tt.call, nil)
			mockNotify.EXPECT().NotifyEvent(gomock.Any(), notifyhandler.EventTypeRecordingFinished, tt.call.WebhookURI, tt.recording)
			mockDB.EXPECT().ConfbridgeSetRecordID(gomock.Any(), tt.recording.ReferenceID, uuid.Nil).Return(nil)

			if err := h.EventHandlerRecordingFinished(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
