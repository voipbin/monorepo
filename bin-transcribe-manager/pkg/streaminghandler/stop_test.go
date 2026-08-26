package streaminghandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-transcribe-manager/models/streaming"
	reflect "reflect"
	"testing"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	stderrors "errors"

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),
				},
			},

			id: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),

			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),
			},

			expectExternalMediaID: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),
			expectRes: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0ffda78c-e9de-11ef-80b6-af1f6f9f7939"),
				},
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
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				mapStreaming:  make(map[uuid.UUID]*streaming.Streaming),
			}
			ctx := context.Background()

			h.mapStreaming[tt.streaming.ID] = tt.streaming

			mockReq.EXPECT().CallV1ExternalMediaStop(ctx, tt.expectExternalMediaID).Return(tt.responseExternalMedia, nil)
			// a successful stop must clear the map entry(and emit the stopped
			// event via Delete), otherwise a retried Stop for this same
			// streaming id would call CallV1ExternalMediaStop again for media
			// call-manager no longer knows about.
			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStopped, tt.streaming)

			res, err := h.Stop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectRes, res)
			}

			if _, ok := h.mapStreaming[tt.id]; ok {
				t.Errorf("Wrong match. expected the streaming to be removed from mapStreaming after a successful stop, but it is still present. streaming_id: %s", tt.id)
			}
		})
	}
}

// Test_Stop_externalMediaStopFails verifies that when the call-manager RPC
// fails, Stop returns the error and leaves the map entry in place(so a retry
// can still find and re-attempt stopping it).
func Test_Stop_externalMediaStopFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	id := uuid.FromStringOrNil("2f5b6f8c-2a52-4c1c-9f1a-1a0f8c8f6b21")
	st := &streaming.Streaming{
		Identity: commonidentity.Identity{ID: id},
	}

	h := &streamingHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		mapStreaming:  map[uuid.UUID]*streaming.Streaming{id: st},
	}
	ctx := context.Background()

	mockReq.EXPECT().CallV1ExternalMediaStop(ctx, id).Return(nil, stderrors.New("could not stop the external media"))

	res, err := h.Stop(ctx, id)
	if err == nil {
		t.Errorf("Wrong match. expected: error, got: ok, res: %v", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expected: nil, got: %v", res)
	}

	if _, ok := h.mapStreaming[id]; !ok {
		t.Errorf("Wrong match. expected the streaming to remain in mapStreaming after a failed stop so a retry can find it, but it was removed. streaming_id: %s", id)
	}
}
