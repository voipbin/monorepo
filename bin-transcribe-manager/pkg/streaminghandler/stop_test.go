package streaminghandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-transcribe-manager/models/streaming"
	reflect "reflect"
	"testing"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Stop(t *testing.T) {

	tests := []struct {
		name      string
		streaming *streaming.Streaming

		id uuid.UUID

		responseExternalMedia *cmexternalmedia.ExternalMedia

		expectExternalMediaID uuid.UUID
		expectRes             *streaming.Streaming
	}{
		{
			name: "normal",
			streaming: &streaming.Streaming{
				ID: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),
			},

			id: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),

			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),
			},

			expectExternalMediaID: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),
			expectRes: &streaming.Streaming{
				ID: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),
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
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)

			h := &streamingHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotfiy,
				mapStreaming:  make(map[uuid.UUID]*streaming.Streaming),
			}
			ctx := context.Background()

			h.mapStreaming[tt.streaming.ID] = tt.streaming

			mockReq.EXPECT().CallV1ExternalMediaStop(ctx, tt.expectExternalMediaID).Return(tt.responseExternalMedia, nil)

			res, err := h.Stop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
