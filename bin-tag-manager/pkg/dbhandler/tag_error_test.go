package dbhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/cachehandler"
)

func Test_TagGet_FromCache(t *testing.T) {
	tests := []struct {
		name      string
		id        uuid.UUID
		cacheTag  *tag.Tag
		cacheErr  error
		expectErr bool
	}{
		{
			name: "get_from_cache_success",
			id:   uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
			cacheTag: &tag.Tag{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				},
				Name: "cached tag",
			},
			cacheErr:  nil,
			expectErr: false,
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

			mockCache.EXPECT().TagGet(ctx, tt.id).Return(tt.cacheTag, tt.cacheErr)

			res, err := h.TagGet(ctx, tt.id)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && res.ID != tt.cacheTag.ID {
				t.Errorf("Wrong tag returned. expect: %s, got: %s", tt.cacheTag.ID, res.ID)
			}
		})
	}
}

func Test_TagUpdate_Error(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name   string
		tag    *tag.Tag
		fields map[tag.Field]any
	}{
		{
			name: "update_with_multiple_fields",
			tag: &tag.Tag{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaaabbbb-50d7-11ec-a6b1-8f9671a9e70e"),
					CustomerID: uuid.FromStringOrNil("bbbbcccc-50d7-11ec-a6b1-8f9671a9e70e"),
				},
				Name:   "original name",
				Detail: "original detail",
			},
			fields: map[tag.Field]any{
				tag.FieldName:   "updated name",
				tag.FieldDetail: "updated detail",
			},
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

			// Create the tag first
			mockUtil.EXPECT().TimeNow().Return(curTime)
			mockCache.EXPECT().TagSet(ctx, gomock.Any())
			if err := h.TagCreate(ctx, tt.tag); err != nil {
				t.Errorf("Failed to create tag: %v", err)
			}

			// Now update it
			mockUtil.EXPECT().TimeNow().Return(curTime)
			mockCache.EXPECT().TagSet(ctx, gomock.Any())
			err := h.TagUpdate(ctx, tt.tag.ID, tt.fields)
			if err != nil {
				t.Errorf("Expected successful update, got error: %v", err)
			}
		})
	}
}

func Test_TagGet_NotFound(t *testing.T) {
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

	nonExistentID := uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff")

	mockCache.EXPECT().TagGet(ctx, nonExistentID).Return(nil, fmt.Errorf("not in cache"))

	_, err := h.TagGet(ctx, nonExistentID)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func Test_tagGetFromCache(t *testing.T) {
	tests := []struct {
		name      string
		id        uuid.UUID
		cacheTag  *tag.Tag
		cacheErr  error
		expectErr bool
	}{
		{
			name: "cache_hit",
			id:   uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
			cacheTag: &tag.Tag{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				},
			},
			cacheErr:  nil,
			expectErr: false,
		},
		{
			name:      "cache_miss",
			id:        uuid.FromStringOrNil("350bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
			cacheTag:  nil,
			cacheErr:  fmt.Errorf("not found"),
			expectErr: true,
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

			mockCache.EXPECT().TagGet(ctx, tt.id).Return(tt.cacheTag, tt.cacheErr)

			res, err := h.tagGetFromCache(ctx, tt.id)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error expectation. expect error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && res == nil {
				t.Errorf("Expected tag, got nil")
			}
		})
	}
}

func Test_tagSetToCache(t *testing.T) {
	tests := []struct {
		name     string
		tag      *tag.Tag
		cacheErr error
	}{
		{
			name: "set_to_cache_success",
			tag: &tag.Tag{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				},
				Name: "test tag",
			},
			cacheErr: nil,
		},
		{
			name: "set_to_cache_error",
			tag: &tag.Tag{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("350bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				},
				Name: "test tag 2",
			},
			cacheErr: fmt.Errorf("cache error"),
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

			mockCache.EXPECT().TagSet(ctx, tt.tag).Return(tt.cacheErr)

			err := h.tagSetToCache(ctx, tt.tag)
			if err != tt.cacheErr {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.cacheErr, err)
			}
		})
	}
}
