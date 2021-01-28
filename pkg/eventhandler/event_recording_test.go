package eventhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestEventHandlerRecordingStarted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockSvc := callhandler.NewMockCallHandler(mc)

	h := eventHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockReq,
		callHandler: mockSvc,
	}

	type test struct {
		name      string
		event     *rabbitmqhandler.Event
		recording *recording.Recording
		timestamp string
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     []byte(`{"type": "RecordingStarted","timestamp": "2020-02-10T13:08:18.888","recording": {"name": "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z","format": "wav","state": "recording","target_uri": "channel:test_call"},"asterisk_id": "42:01:0a:84:00:12","application": "voipbin"}`),
			},
			&recording.Recording{
				ID:       uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937"),
				Filename: "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z.wav",
			},
			"2020-02-10T13:08:18.888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().RecordingGetByFilename(gomock.Any(), tt.recording.Filename).Return(tt.recording, nil)
			mockDB.EXPECT().RecordingSetStatus(gomock.Any(), tt.recording.ID, recording.StatusRecording, tt.timestamp).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
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
	mockSvc := callhandler.NewMockCallHandler(mc)

	h := eventHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockReq,
		callHandler: mockSvc,
	}

	type test struct {
		name      string
		event     *rabbitmqhandler.Event
		recording *recording.Recording
		timestamp string
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     []byte(`{"type": "RecordingFinished","timestamp": "2020-02-10T13:08:40.888","recording": {"name": "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888","format": "wav","state": "done","target_uri": "channel:test_call","duration": 351},"asterisk_id": "42:01:0a:84:00:12","application": "voipbin"}`),
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

			mockDB.EXPECT().RecordingGetByFilename(gomock.Any(), tt.recording.Filename).Return(tt.recording, nil)
			mockDB.EXPECT().RecordingSetStatus(gomock.Any(), tt.recording.ID, recording.StatusEnd, tt.timestamp).Return(nil)
			mockDB.EXPECT().CallSetRecordID(gomock.Any(), tt.recording.ReferenceID, uuid.Nil).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// func TestEventHandlerRecordingFinishedConference(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockDB := dbhandler.NewMockDBHandler(mc)
// 	mockSock := rabbitmqhandler.NewMockRabbit(mc)
// 	mockReq := requesthandler.NewMockRequestHandler(mc)
// 	mockSvc := callhandler.NewMockCallHandler(mc)

// 	h := eventHandler{
// 		db:          mockDB,
// 		rabbitSock:  mockSock,
// 		reqHandler:  mockReq,
// 		callHandler: mockSvc,
// 	}

// 	type test struct {
// 		name      string
// 		event     *rabbitmqhandler.Event
// 		recordID  string
// 		timestamp string
// 		bridgeID  string
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			&rabbitmqhandler.Event{
// 				Type:     "ari_event",
// 				DataType: "application/json",
// 				Data:     []byte(`{"type": "RecordingFinished","timestamp": "2020-02-10T13:08:40.888","recording": {"name": "bridge_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888","format": "wav","state": "done","target_uri": "channel:test_call","duration": 351},"asterisk_id": "42:01:0a:84:00:12","application": "voipbin"}`),
// 			},
// 			"bridge_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888",
// 			"2020-02-10T13:08:40.888",
// 			"3b16cef6-2b99-11eb-87eb-571ab4136611",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			mockDB.EXPECT().RecordingSetStatus(gomock.Any(), tt.recordID, record.StatusEnd, tt.timestamp).Return(nil)
// 			mockDB.EXPECT().BridgeSetRecordID(gomock.Any(), tt.bridgeID, "").Return(nil)

// 			if err := h.processEvent(tt.event); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }
