package dbhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/cachehandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_StreamingCreate(t *testing.T) {
	streamingID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	customerID := uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")

	tests := []struct {
		name      string
		streaming *streaming.Streaming
		mockError error
		expectErr bool
	}{
		{
			name: "normal",
			streaming: &streaming.Streaming{
				Language:  "en-US",
				Provider:  "elevenlabs",
				VoiceID:   "voice123",
				Direction: streaming.DirectionIncoming,
			},
			expectErr: false,
		},
		{
			name: "cache error",
			streaming: &streaming.Streaming{
				Language: "en-US",
			},
			mockError: fmt.Errorf("cache error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			tt.streaming.ID = streamingID
			tt.streaming.CustomerID = customerID

			mockCache.EXPECT().StreamingSet(ctx, tt.streaming).Return(tt.mockError)

			err := h.StreamingCreate(ctx, tt.streaming)
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func Test_StreamingGet(t *testing.T) {
	streamingID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	customerID := uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")

	tests := []struct {
		name       string
		id         uuid.UUID
		mockStream *streaming.Streaming
		mockError  error
		expectErr  bool
	}{
		{
			name: "normal",
			id:   streamingID,
			mockStream: &streaming.Streaming{
				Language: "en-US",
			},
			expectErr: false,
		},
		{
			name:      "not found",
			id:        streamingID,
			mockError: fmt.Errorf("not found"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			if tt.mockStream != nil {
				tt.mockStream.ID = streamingID
				tt.mockStream.CustomerID = customerID
			}

			mockCache.EXPECT().StreamingGet(ctx, tt.id).Return(tt.mockStream, tt.mockError)

			res, err := h.StreamingGet(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if res == nil {
					t.Errorf("expected result, got nil")
				}
			}
		})
	}
}

func Test_StreamingUpdate(t *testing.T) {
	streamingID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	customerID := uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")

	tests := []struct {
		name      string
		streaming *streaming.Streaming
		mockError error
		expectErr bool
	}{
		{
			name: "normal",
			streaming: &streaming.Streaming{
				Language: "en-US",
			},
			expectErr: false,
		},
		{
			name: "cache error",
			streaming: &streaming.Streaming{
				Language: "en-US",
			},
			mockError: fmt.Errorf("cache error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			tt.streaming.ID = streamingID
			tt.streaming.CustomerID = customerID

			mockCache.EXPECT().StreamingSet(ctx, tt.streaming).Return(tt.mockError)

			err := h.StreamingUpdate(ctx, tt.streaming)
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func Test_streamingSetToCache(t *testing.T) {
	streamingID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	customerID := uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")

	tests := []struct {
		name      string
		streaming *streaming.Streaming
		mockError error
		expectErr bool
	}{
		{
			name: "normal",
			streaming: &streaming.Streaming{
				Language: "en-US",
			},
			expectErr: false,
		},
		{
			name: "cache error",
			streaming: &streaming.Streaming{
				Language: "en-US",
			},
			mockError: fmt.Errorf("cache error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			tt.streaming.ID = streamingID
			tt.streaming.CustomerID = customerID

			mockCache.EXPECT().StreamingSet(ctx, tt.streaming).Return(tt.mockError)

			err := h.streamingSetToCache(ctx, tt.streaming)
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func Test_streamingGetFromCache(t *testing.T) {
	streamingID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	customerID := uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")

	tests := []struct {
		name       string
		id         uuid.UUID
		mockStream *streaming.Streaming
		mockError  error
		expectErr  bool
	}{
		{
			name: "normal",
			id:   streamingID,
			mockStream: &streaming.Streaming{
				Language: "en-US",
			},
			expectErr: false,
		},
		{
			name:      "not found",
			id:        streamingID,
			mockError: fmt.Errorf("not found"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &dbHandler{
				util:  utilhandler.NewUtilHandler(),
				cache: mockCache,
			}
			ctx := context.Background()

			if tt.mockStream != nil {
				tt.mockStream.ID = streamingID
				tt.mockStream.CustomerID = customerID
			}

			mockCache.EXPECT().StreamingGet(ctx, tt.id).Return(tt.mockStream, tt.mockError)

			res, err := h.streamingGetFromCache(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if res == nil {
					t.Errorf("expected result, got nil")
				}
			}
		})
	}
}
