package arieventhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

func Test_EventHandlerRecordingStarted(t *testing.T) {

	tests := []struct {
		name  string
		event *ari.RecordingStarted

		responseRecording *recording.Recording

		expectRecordingName string
	}{
		{
			name: "normal",
			event: &ari.RecordingStarted{
				Event: ari.Event{
					Type:        ari.EventTypeRecordingStarted,
					Application: "voipbin",
					Timestamp:   "2020-02-10T13:08:18.888Z",
					AsteriskID:  "42:01:0a:84:00:12",
				},
				Recording: ari.RecordingLive{
					Name:            "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z_in",
					Format:          "wav",
					State:           "recording",
					SilenceDuration: 0,
					Duration:        0,
					TalkingDuration: 0,
					TargetURI:       "channel:test_call",
				},
			},
			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937"),
				},
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("e31efb5e-1f3c-11ec-beea-af98446e3b8e"),
				Filenames: []string{
					"call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z_in.wav",
				},
			},

			expectRecordingName: "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z",
		},
		{
			name: "recording name end with _out",
			event: &ari.RecordingStarted{
				Event: ari.Event{
					Type:        ari.EventTypeRecordingStarted,
					Application: "voipbin",
					Timestamp:   "2020-02-10T13:08:18.888Z",
					AsteriskID:  "42:01:0a:84:00:12",
				},
				Recording: ari.RecordingLive{
					Name:            "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z_out",
					Format:          "wav",
					State:           "recording",
					SilenceDuration: 0,
					Duration:        0,
					TalkingDuration: 0,
					TargetURI:       "channel:test_call",
				},
			},
			responseRecording: &recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937"),
				},
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("e31efb5e-1f3c-11ec-beea-af98446e3b8e"),
				Filenames: []string{
					"call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z_out.wav",
				},
			},

			expectRecordingName: "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockSvc := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := eventHandler{
				db:               mockDB,
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				notifyHandler:    mockNotify,
				callHandler:      mockSvc,
				recordingHandler: mockRecording,
			}

			ctx := context.Background()
			if strings.HasSuffix(tt.event.Recording.Name, "_in") {
				mockRecording.EXPECT().GetByRecordingName(ctx, tt.expectRecordingName).Return(tt.responseRecording, nil)
				mockRecording.EXPECT().Started(ctx, tt.responseRecording.ID).Return(tt.responseRecording, nil)
			}

			if err := h.EventHandlerRecordingStarted(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerRecordingFinished_call(t *testing.T) {

	tests := []struct {
		name              string
		event             *ari.RecordingFinished
		call              *call.Call
		responseRecording *recording.Recording
		timestamp         *time.Time

		expectRecordingName string
	}{
		{
			"normal",
			&ari.RecordingFinished{
				Event: ari.Event{
					Type:        ari.EventTypeRecordingFinished,
					Application: "voipbin",
					Timestamp:   "2020-02-10T13:08:40.888Z",
					AsteriskID:  "42:01:0a:84:00:12",
				},
				Recording: ari.RecordingLive{
					Name:            "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z_in",
					Format:          "wav",
					State:           "done",
					SilenceDuration: 0,
					Duration:        351,
					TalkingDuration: 0,
					TargetURI:       "channel:test_call",
				},
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3b16cef6-2b99-11eb-87eb-571ab4136611"),
				},
			},
			&recording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f367e2c-612c-11eb-b063-676ca5ee546a"),
				},
				Filenames: []string{
					"call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888_in.wav",
					"call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888_out.wav",
				},
				Format:        "wav",
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3b16cef6-2b99-11eb-87eb-571ab4136611"),
			},
			testhelper.TimePtr("2020-02-10T13:08:40.888Z"),

			"call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockSvc := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := eventHandler{
				db:               mockDB,
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				notifyHandler:    mockNotify,
				callHandler:      mockSvc,
				recordingHandler: mockRecording,
			}

			ctx := context.Background()

			mockRecording.EXPECT().GetByRecordingName(ctx, tt.expectRecordingName).Return(tt.responseRecording, nil)
			mockRecording.EXPECT().Stopped(ctx, tt.responseRecording.ID).Return(tt.responseRecording, nil)
			mockSvc.EXPECT().UnsetRecordingID(ctx, tt.responseRecording.ReferenceID, tt.responseRecording.ID).Return(nil)

			if err := h.EventHandlerRecordingFinished(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerRecordingFinished_clearsCallRecordingID(t *testing.T) {

	finished := &ari.RecordingFinished{
		Event: ari.Event{
			Type:        ari.EventTypeRecordingFinished,
			Application: "voipbin",
			Timestamp:   "2020-02-10T13:08:40.888Z",
			AsteriskID:  "42:01:0a:84:00:12",
		},
		Recording: ari.RecordingLive{
			Name:  "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z_in",
			State: "done",
		},
	}
	recordingName := "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z"
	rec := func(refType recording.ReferenceType) *recording.Recording {
		return &recording.Recording{
			Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("4f367e2c-612c-11eb-b063-676ca5ee546a")},
			ReferenceType: refType,
			ReferenceID:   uuid.FromStringOrNil("3b16cef6-2b99-11eb-87eb-571ab4136611"),
		}
	}

	tests := []struct {
		name string

		responseRecording *recording.Recording
		responseStopped   error
		responseUnset     error

		expectUnset bool
		expectErr   bool
	}{
		{
			name:              "call recording: the id is cleared after Stopped",
			responseRecording: rec(recording.ReferenceTypeCall),
			expectUnset:       true,
		},
		{
			name:              "call recording: Stopped fails - the id is still cleared and the error is returned",
			responseRecording: rec(recording.ReferenceTypeCall),
			responseStopped:   errors.New("could not store the recording files"),
			expectUnset:       true,
			expectErr:         true,
		},
		{
			name:              "call recording: clearing fails - only logged, the event result is unchanged",
			responseRecording: rec(recording.ReferenceTypeCall),
			responseUnset:     errors.New("db failure"),
			expectUnset:       true,
		},
		{
			name:              "confbridge recording: the call is not touched",
			responseRecording: rec(recording.ReferenceTypeConfbridge),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := eventHandler{
				callHandler:      mockSvc,
				recordingHandler: mockRecording,
			}

			ctx := context.Background()

			mockRecording.EXPECT().GetByRecordingName(ctx, recordingName).Return(tt.responseRecording, nil)
			var stoppedRes *recording.Recording
			if tt.responseStopped == nil {
				stoppedRes = tt.responseRecording
			}
			mockRecording.EXPECT().Stopped(ctx, tt.responseRecording.ID).Return(stoppedRes, tt.responseStopped)
			if tt.expectUnset {
				mockSvc.EXPECT().UnsetRecordingID(ctx, tt.responseRecording.ReferenceID, tt.responseRecording.ID).Return(tt.responseUnset)
			}

			err := h.EventHandlerRecordingFinished(ctx, finished)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
