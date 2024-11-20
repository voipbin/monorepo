package transcribehandler

import (
	"context"
	"reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func Test_Start_referencetype_call(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		referenceType transcribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     transcribe.Direction

		responseCall       *cmcall.Call
		responseUUID       uuid.UUID
		responseStreamings []*streaming.Streaming
		responseTranscribe *transcribe.Transcribe

		expectTranscribe *transcribe.Transcribe
		expectRes        *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("0e259c1c-8211-11ed-a907-5bf5bd61fa6a"),
			transcribe.ReferenceTypeCall,
			uuid.FromStringOrNil("0e5ecd0c-8211-11ed-9c0a-4fa1d29f93c2"),
			"en-US",
			transcribe.DirectionBoth,

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0e5ecd0c-8211-11ed-9c0a-4fa1d29f93c2"),
				},
				Status: cmcall.StatusProgressing,
			},
			uuid.FromStringOrNil("a4b155b6-9875-11ed-9117-1f7140765600"),
			[]*streaming.Streaming{
				{
					ID: uuid.FromStringOrNil("049c01c4-9876-11ed-968a-0f8060a7f327"),
				},
				{
					ID: uuid.FromStringOrNil("0b7ca494-9876-11ed-8927-3b1f974a4122"),
				},
			},
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("5241c614-8216-11ed-9e05-ab1368296bbd"),
			},

			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("a4b155b6-9875-11ed-9117-1f7140765600"),
				CustomerID:    uuid.FromStringOrNil("0e259c1c-8211-11ed-a907-5bf5bd61fa6a"),
				ReferenceType: transcribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("0e5ecd0c-8211-11ed-9c0a-4fa1d29f93c2"),
				Status:        transcribe.StatusProgressing,
				Language:      "en-US",
				Direction:     transcribe.DirectionBoth,
				StreamingIDs: []uuid.UUID{
					uuid.FromStringOrNil("049c01c4-9876-11ed-968a-0f8060a7f327"),
					uuid.FromStringOrNil("0b7ca494-9876-11ed-8927-3b1f974a4122"),
				},
			},
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("5241c614-8216-11ed-9e05-ab1368296bbd"),
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
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &transcribeHandler{
				utilHandler:       mockUtil,
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,
				streamingHandler:  mockStreaming,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)

			// streaming start
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)

			if tt.direction == transcribe.DirectionBoth {
				mockStreaming.EXPECT().Start(ctx, tt.customerID, tt.responseUUID, tt.referenceType, tt.referenceID, tt.language, transcript.DirectionIn).Return(tt.responseStreamings[0], nil)
				mockStreaming.EXPECT().Start(ctx, tt.customerID, tt.responseUUID, tt.referenceType, tt.referenceID, tt.language, transcript.DirectionOut).Return(tt.responseStreamings[1], nil)
			} else {
				mockStreaming.EXPECT().Start(ctx, tt.customerID, tt.responseUUID, tt.referenceType, tt.referenceID, tt.language, tt.direction).Return(tt.responseStreamings[0], nil)
			}

			// create
			mockDB.EXPECT().TranscribeCreate(ctx, tt.expectTranscribe).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, gomock.Any()).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())

			res, err := h.Start(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_isValidReference(t *testing.T) {

	tests := []struct {
		name string

		referenceType transcribe.ReferenceType
		referenceID   uuid.UUID

		responseCall       *cmcall.Call
		responseConfbridge *cmconfbridge.Confbridge

		expectRes bool
	}{
		{
			name: "reference type call",

			referenceType: transcribe.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("918c6c26-98ae-11ed-8a80-a703c7717d9a"),

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("918c6c26-98ae-11ed-8a80-a703c7717d9a"),
				},
				Status: cmcall.StatusProgressing,
			},

			expectRes: true,
		},
		{
			name: "reference type confbridge",

			referenceType: transcribe.ReferenceTypeConfbridge,
			referenceID:   uuid.FromStringOrNil("915fe2c8-98ae-11ed-8b05-bf167f4d8651"),

			responseConfbridge: &cmconfbridge.Confbridge{
				ID:       uuid.FromStringOrNil("915fe2c8-98ae-11ed-8b05-bf167f4d8651"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectRes: true,
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
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &transcribeHandler{
				utilHandler:       mockUtil,
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,
				streamingHandler:  mockStreaming,
			}

			ctx := context.Background()

			switch tt.referenceType {
			case transcribe.ReferenceTypeCall:
				mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID).Return(tt.responseCall, nil)

			case transcribe.ReferenceTypeConfbridge:
				mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.referenceID).Return(tt.responseConfbridge, nil)
			}

			res := h.isValidReference(ctx, tt.referenceType, tt.referenceID)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_startLive(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		referenceType transcribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     transcribe.Direction

		responseUUID       uuid.UUID
		responseStreamings []*streaming.Streaming
		responseTranscribe *transcribe.Transcribe

		expectRes *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("469b200c-8786-11ec-bd4f-bb7ae5541d57"),
			transcribe.ReferenceTypeCall,
			uuid.FromStringOrNil("47b30720-8786-11ec-ac47-f37c07bbbef5"),
			"en-US",
			transcribe.DirectionBoth,

			uuid.FromStringOrNil("ad23290c-9877-11ed-8d54-07172f870dfb"),
			[]*streaming.Streaming{
				{
					ID: uuid.FromStringOrNil("d01d68a0-9877-11ed-b51e-072b2ebe66d1"),
				},
				{
					ID: uuid.FromStringOrNil("d0462556-9877-11ed-a96d-534363ee9536"),
				},
			},
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("49a3529c-8786-11ec-928e-bb8e9b925697"),
			},

			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("49a3529c-8786-11ec-928e-bb8e9b925697"),
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
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &transcribeHandler{
				utilHandler:       mockUtil,
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,
				streamingHandler:  mockStreaming,
			}

			ctx := context.Background()

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, gomock.Any()).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeCreated, tt.responseTranscribe)

			if tt.direction == transcribe.DirectionBoth {
				mockStreaming.EXPECT().Start(ctx, tt.customerID, tt.responseUUID, tt.referenceType, tt.referenceID, tt.language, transcript.DirectionIn).Return(tt.responseStreamings[0], nil)
				mockStreaming.EXPECT().Start(ctx, tt.customerID, tt.responseUUID, tt.referenceType, tt.referenceID, tt.language, transcript.DirectionOut).Return(tt.responseStreamings[1], nil)
			} else {
				mockStreaming.EXPECT().Start(ctx, tt.customerID, tt.responseUUID, tt.referenceType, tt.referenceID, tt.language, tt.direction).Return(tt.responseStreamings[0], nil)
			}

			res, err := h.startLive(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
