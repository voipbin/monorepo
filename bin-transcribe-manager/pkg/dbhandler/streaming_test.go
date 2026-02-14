package dbhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
)

func Test_StreamingCreate(t *testing.T) {
	tests := []struct {
		name      string
		streaming *streaming.Streaming
		wantErr   bool
	}{
		{
			name: "successful_create",
			streaming: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				TranscribeID: uuid.Must(uuid.NewV4()),
				Language:     "en-US",
				Direction:    transcript.DirectionIn,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().StreamingSet(ctx, tt.streaming).Return(nil)

			err := h.StreamingCreate(ctx, tt.streaming)
			if (err != nil) != tt.wantErr {
				t.Errorf("StreamingCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_StreamingGet(t *testing.T) {
	tests := []struct {
		name      string
		id        uuid.UUID
		streaming *streaming.Streaming
		wantErr   bool
	}{
		{
			name: "successful_get",
			id:   uuid.Must(uuid.NewV4()),
			streaming: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				TranscribeID: uuid.Must(uuid.NewV4()),
				Language:     "en-US",
				Direction:    transcript.DirectionIn,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().StreamingGet(ctx, tt.id).Return(tt.streaming, nil)

			res, err := h.StreamingGet(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("StreamingGet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && res == nil {
				t.Error("StreamingGet() returned nil result")
			}
		})
	}
}
