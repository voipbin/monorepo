package dbhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

func Test_ExternalMediaGet(t *testing.T) {
	tests := []struct {
		name            string
		externalMediaID uuid.UUID
		cacheReturn     *externalmedia.ExternalMedia
		cacheErr        error
		expectErr       bool
	}{
		{
			name:            "successful get",
			externalMediaID: uuid.Must(uuid.NewV4()),
			cacheReturn: &externalmedia.ExternalMedia{
				ReferenceID: uuid.Must(uuid.NewV4()),
			},
			cacheErr:  nil,
			expectErr: false,
		},
		{
			name:            "cache error",
			externalMediaID: uuid.Must(uuid.NewV4()),
			cacheReturn:     nil,
			cacheErr:        fmt.Errorf("cache error"),
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().
				ExternalMediaGet(ctx, tt.externalMediaID).
				Return(tt.cacheReturn, tt.cacheErr)

			res, err := h.ExternalMediaGet(ctx, tt.externalMediaID)
			if (err != nil) != tt.expectErr {
				t.Errorf("ExternalMediaGet() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && res != tt.cacheReturn {
				t.Errorf("ExternalMediaGet() = %v, want %v", res, tt.cacheReturn)
			}
		})
	}
}

func Test_ExternalMediaGetByReferenceID(t *testing.T) {
	tests := []struct {
		name        string
		referenceID uuid.UUID
		cacheReturn *externalmedia.ExternalMedia
		cacheErr    error
		expectErr   bool
	}{
		{
			name:        "successful get by reference id",
			referenceID: uuid.Must(uuid.NewV4()),
			cacheReturn: &externalmedia.ExternalMedia{
				ReferenceID: uuid.Must(uuid.NewV4()),
			},
			cacheErr:  nil,
			expectErr: false,
		},
		{
			name:        "cache error",
			referenceID: uuid.Must(uuid.NewV4()),
			cacheReturn: nil,
			cacheErr:    fmt.Errorf("cache error"),
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().
				ExternalMediaGetByReferenceID(ctx, tt.referenceID).
				Return(tt.cacheReturn, tt.cacheErr)

			res, err := h.ExternalMediaGetByReferenceID(ctx, tt.referenceID)
			if (err != nil) != tt.expectErr {
				t.Errorf("ExternalMediaGetByReferenceID() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && res != tt.cacheReturn {
				t.Errorf("ExternalMediaGetByReferenceID() = %v, want %v", res, tt.cacheReturn)
			}
		})
	}
}

func Test_ExternalMediaSet(t *testing.T) {
	tests := []struct {
		name      string
		data      *externalmedia.ExternalMedia
		cacheErr  error
		expectErr bool
	}{
		{
			name: "successful set",
			data: &externalmedia.ExternalMedia{
				ReferenceID: uuid.Must(uuid.NewV4()),
			},
			cacheErr:  nil,
			expectErr: false,
		},
		{
			name: "cache error",
			data: &externalmedia.ExternalMedia{
				ReferenceID: uuid.Must(uuid.NewV4()),
			},
			cacheErr:  fmt.Errorf("cache error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().
				ExternalMediaSet(ctx, tt.data).
				Return(tt.cacheErr)

			err := h.ExternalMediaSet(ctx, tt.data)
			if (err != nil) != tt.expectErr {
				t.Errorf("ExternalMediaSet() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func Test_ExternalMediaDelete(t *testing.T) {
	tests := []struct {
		name            string
		externalMediaID uuid.UUID
		cacheErr        error
		expectErr       bool
	}{
		{
			name:            "successful delete",
			externalMediaID: uuid.Must(uuid.NewV4()),
			cacheErr:        nil,
			expectErr:       false,
		},
		{
			name:            "cache error",
			externalMediaID: uuid.Must(uuid.NewV4()),
			cacheErr:        fmt.Errorf("cache error"),
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().
				ExternalMediaDelete(ctx, tt.externalMediaID).
				Return(tt.cacheErr)

			err := h.ExternalMediaDelete(ctx, tt.externalMediaID)
			if (err != nil) != tt.expectErr {
				t.Errorf("ExternalMediaDelete() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
