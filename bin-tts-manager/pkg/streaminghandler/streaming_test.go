package streaminghandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/streaming"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		podID         string
		customerID    uuid.UUID
		referenceType streaming.ReferenceType
		referenceID   uuid.UUID
		language      string
		gender        streaming.Gender
		direction     streaming.Direction

		responseID uuid.UUID

		expectRes *streaming.Streaming
	}{
		{
			name: "normal",

			podID:         "14b9f816-5af4-11f0-947b-c3120e6156c8",
			customerID:    uuid.FromStringOrNil("14433528-5af4-11f0-a98c-37f80c2463fe"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("146d1230-5af4-11f0-a19c-fb7727ed4659"),
			language:      "en-US",
			gender:        streaming.GenderFemale,
			direction:     streaming.DirectionIncoming,

			responseID: uuid.FromStringOrNil("14907f86-5af4-11f0-99e2-2bb802f86fcd"),

			expectRes: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14907f86-5af4-11f0-99e2-2bb802f86fcd"),
					CustomerID: uuid.FromStringOrNil("14433528-5af4-11f0-a98c-37f80c2463fe"),
				},
				PodID:         "14b9f816-5af4-11f0-947b-c3120e6156c8",
				ReferenceType: streaming.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("146d1230-5af4-11f0-a19c-fb7727ed4659"),
				Gender:        streaming.GenderFemale,
				Language:      "en-US",
				Direction:     streaming.DirectionIncoming,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &streamingHandler{
				utilHandler:    mockUtil,
				requestHandler: mockReq,
				notifyHandler:  mockNotify,
				mapStreaming:   make(map[uuid.UUID]*streaming.Streaming),
				podID:          tt.podID,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseID)
			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingCreated, gomock.Any())

			res, err := h.Create(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.gender, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			tt.expectRes.ChanDone = res.ChanDone
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectRes, res)
			}

			resGet, err := h.Get(ctx, tt.responseID)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			tt.expectRes.ChanDone = resGet.ChanDone
			if !reflect.DeepEqual(resGet, tt.expectRes) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectRes, resGet)
			}

			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingDeleted, tt.expectRes)
			h.Delete(ctx, tt.responseID)
			resDelete, err := h.Get(ctx, tt.responseID)
			if err == nil {
				t.Errorf("Wrong match. expected: error, got: ok")
			}

			if resDelete != nil {
				t.Errorf("Wrong match. expected: nil, got: %v", resDelete)
			}

		})
	}
}
